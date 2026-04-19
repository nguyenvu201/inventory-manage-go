# Sprint 6: Firebase & Cloud Deployment — Customer Preview Environment

> **Mục tiêu:** Deploy hệ thống lên môi trường cloud để khách hàng có thể truy cập và kiểm tra sản phẩm trực tiếp qua URL công khai, mà không cần chạy Docker Compose trên máy local.

---

## Metadata

| Field           | Value                                                             |
|-----------------|-------------------------------------------------------------------|
| Sprint          | 6 (Deployment Sprint — song song với Sprint 3–5)                  |
| Status          | 🔲 Not Started                                                    |
| Created date    | 2026-04-19                                                        |
| Owner           | —                                                                 |
| Priority        | **CRITICAL** — Customer acceptance testing blocker                |
| Dependencies    | Sprint 1 & 2 đã CLOSED (lõi ingestion & device management hoạt động) |

---

## 📖 Giải Thích Kiến Trúc Deploy — Tại Sao Không Deploy Go Lên Firebase Trực Tiếp

> **Firebase** là nền tảng của Google với các dịch vụ: Hosting (static files), Firestore (database NoSQL), Cloud Functions (Node.js/Python). **Firebase KHÔNG thể chạy Go binary.**

```
Giải pháp tổng thể:

  Browser / Khách hàng
        │
        ▼ https://<project>.web.app
  Firebase Hosting   ← Static UI files (HTML/CSS/JS)
        │ rewrites rules
        ▼ https://go-backend-xxx-run.app/api/*
  Google Cloud Run   ← Go Docker container (managed serverless)
        │
        ▼
  Cloud SQL (PostgreSQL) + Redis (Memorystore hoặc Upstash)
        │
        ▼
  MQTT Broker: HiveMQ Cloud (managed) hoặc Mosquitto trên VM
```

**Lý do chọn Cloud Run cho Go backend:**
- Tương thích Docker — dùng lại `Dockerfile` hiện có
- Trả tiền theo request (free tier rộng rãi để demo)
- Scale tự động, không quản lý server
- Cùng Google Cloud với Firebase → latency thấp, auth tích hợp
- Có thể dùng Firebase Hosting rewrites để che URL backend (client không thấy Cloud Run URL)

---

## Tổng Quan Các Tasks Sprint 6

| Task ID             | Tên Task                              | Ưu tiên | Thời lượng ước tính |
|---------------------|---------------------------------------|---------|---------------------|
| INV-SPR06-TASK-001  | Chuẩn bị môi trường & phân tích gap  | P0      | 0.5 ngày            |
| INV-SPR06-TASK-002  | Containerize & tối ưu Docker cho Cloud Run | P0 | 1 ngày             |
| INV-SPR06-TASK-003  | Cấu hình Cloud SQL (PostgreSQL managed) | P0   | 0.5 ngày            |
| INV-SPR06-TASK-004  | Deploy Go backend lên Google Cloud Run | P0    | 1 ngày              |
| INV-SPR06-TASK-005  | Cấu hình MQTT Broker trên cloud       | P1      | 0.5 ngày            |
| INV-SPR06-TASK-006  | Firebase Hosting + rewrites tới Cloud Run | P0  | 0.5 ngày           |
| INV-SPR06-TASK-007  | CI/CD pipeline với GitHub Actions     | P1      | 1 ngày              |
| INV-SPR06-TASK-008  | Environment config & secrets management | P0   | 0.5 ngày            |
| INV-SPR06-TASK-009  | Customer preview URL & smoke testing  | P0      | 0.5 ngày            |

**Tổng ước tính: ~5–6 ngày làm việc**

---

## [INV-SPR06-TASK-001] — Chuẩn Bị Môi Trường & Phân Tích Gap

> **Task ID:** `INV-SPR06-TASK-001`
> **Status:** 📝 DRAFT
> **Created by:** BA
> **Created date:** 2026-04-19
> **Assignee:** —
> **Sprint:** 6

**Description:**
Kiểm kê toàn bộ các phụ thuộc infrastructure (DB, Redis, MQTT, config) hiện tại đang chạy qua Docker Compose, xác định những gì cần thay thế hoặc điều chỉnh để chạy được trên cloud. Tạo tài khoản GCP và Firebase project.

**Acceptance Criteria:**
- [ ] AC-01: Tạo Google Cloud Project mới (hoặc dùng project hiện có) với billing account được kích hoạt
- [ ] AC-02: Tạo Firebase project liên kết với Google Cloud Project trên
- [ ] AC-03: Cài đặt và xác minh các CLI tools: `gcloud`, `firebase-tools`, `docker`
- [ ] AC-04: Viết tài liệu `docs/deployment/gap-analysis.md` liệt kê rõ:
  - Những gì đang dùng trên local (localhost DB, local MQTT, local Redis)
  - Tương đương cloud của từng service
  - Cấu hình nào cần override cho môi trường cloud
- [ ] AC-05: Xác nhận Free Tier / Chi phí ước tính hàng tháng cho giai đoạn preview (phải < $50/tháng)
- [ ] AC-06: Tạo thư mục `infrastructure/cloud/` để chứa tất cả config liên quan deployment

**Related Technologies:**
- Google Cloud Platform (GCP)
- Firebase CLI (`firebase-tools`)
- `gcloud` CLI

**Notes / Dependencies:**
- **KHÔNG** cần Sprint 3–5 hoàn thành — Sprint 6 có thể chạy song song với các sprint đang dở
- Phiên bản deploy trước sẽ là phiên bản "preview" với data mẫu, không phải production

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-19 | —    | DRAFT | BA           | Task created — customer preview sprint |

---

## [INV-SPR06-TASK-002] — Containerize & Tối Ưu Docker Cho Cloud Run

> **Task ID:** `INV-SPR06-TASK-002`
> **Status:** 📝 DRAFT
> **Created by:** BA
> **Created date:** 2026-04-19
> **Assignee:** —
> **Sprint:** 6

**Description:**
Cloud Run yêu cầu container lắng nghe trên `PORT` env var (không hardcode). Cần audit Dockerfile hiện tại, thêm multi-stage build (nếu chưa có), và đảm bảo ứng dụng đọc cấu hình từ environment variables — không phải file `local.yaml` hardcode.

**Acceptance Criteria:**
- [ ] AC-01: Dockerfile đã dùng **multi-stage build**: stage `builder` (Go compiler) + stage `runtime` (distroless/alpine)
- [ ] AC-02: Go binary lắng nghe trên `$PORT` env var — nếu không set thì fallback về `8080`:
  ```go
  port := os.Getenv("PORT")
  if port == "" { port = "8080" }
  ```
- [ ] AC-03: Tạo file `config/cloud.yaml` (template) với tất cả values là `${ENV_VAR_NAME}` — không có giá trị hardcode
- [ ] AC-04: Build Docker image locally và kiểm tra: `docker build -t inventory-api . && docker run -p 8080:8080 inventory-api`
- [ ] AC-05: Image size sau multi-stage build < 50 MB (kiểm tra bằng `docker images`)
- [ ] AC-06: Tạo file `infrastructure/cloud/cloudbuild.yaml` cho Google Cloud Build (CI/CD pipeline)
- [ ] AC-07: `.dockerignore` đã loại trừ: `.git/`, `storages/`, `*.log`, `config/local.yaml`, `secrets/`

**Related Technologies:**
- Docker multi-stage build (`golang:1.22-alpine` → `alpine:3.19`)
- Google Cloud Build
- Viper config (hỗ trợ đọc env var natively)

**Notes / Dependencies:**
- Phụ thuộc: INV-SPR06-TASK-001 (cần GCP project trước)
- Lưu ý: `config/local.yaml` **giữ nguyên** cho local dev, không xóa

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-19 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR06-TASK-003] — Cấu Hình Cloud SQL (PostgreSQL Managed)

> **Task ID:** `INV-SPR06-TASK-003`
> **Status:** 📝 DRAFT
> **Created by:** BA
> **Created date:** 2026-04-19
> **Assignee:** —
> **Sprint:** 6

**Description:**
Thay thế PostgreSQL + TimescaleDB Docker container bằng **Cloud SQL for PostgreSQL** (managed service của Google). Lưu ý: Cloud SQL không hỗ trợ TimescaleDB extension — cần đánh giá và migrate nếu cần.

**Acceptance Criteria:**
- [ ] AC-01: Tạo Cloud SQL instance (PostgreSQL 15) trong cùng region với Cloud Run service
- [ ] AC-02: Tạo database `inventory_db` và user `inventory_user` trên Cloud SQL
- [ ] AC-03: **Đánh giá TimescaleDB dependency:**
  - Kiểm tra các migration file xem có dùng TimescaleDB extension (`CREATE EXTENSION timescaledb`, `create_hypertable`, `add_continuous_aggregate_policy`) không
  - Nếu CÓ: ghi lại các query cần thay thế bằng native PostgreSQL partitioning hoặc dùng Cloud SQL alternative
  - Nếu KHÔNG: tiến hành migrate bình thường
- [ ] AC-04: Chạy tất cả migration files lên Cloud SQL instance mới (`golang-migrate up`)
- [ ] AC-05: Kết nối thành công từ local tới Cloud SQL qua Cloud SQL Auth Proxy để kiểm tra
- [ ] AC-06: Kết nối Cloud Run tới Cloud SQL qua **Cloud SQL Connector** (không phải IP trực tiếp) — an toàn hơn
- [ ] AC-07: Seed database với **dữ liệu mẫu** (ít nhất 2 device, 3 SKU, 100 telemetry records)
- [ ] AC-08: Viết `docs/deployment/database-migration.md` ghi lại mọi thay đổi so với local setup

**Related Technologies:**
- Google Cloud SQL for PostgreSQL
- Cloud SQL Auth Proxy
- `golang-migrate` CLI
- `pgx/v5` Cloud SQL Connector

**Notes / Dependencies:**
- ⚠️ **Cảnh báo quan trọng:** Cloud SQL không hỗ trợ TimescaleDB. Nếu code đang dùng hypertable, cần điều chỉnh. Tham khảo AC-03 để đánh giá.
- Fallback option: Dùng **Supabase** (PostgreSQL + hỗ trợ extension tốt hơn) nếu TimescaleDB dependency quá sâu

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-19 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR06-TASK-004] — Deploy Go Backend Lên Google Cloud Run

> **Task ID:** `INV-SPR06-TASK-004`
> **Status:** 📝 DRAFT
> **Created by:** BA
> **Created date:** 2026-04-19
> **Assignee:** —
> **Sprint:** 6

**Description:**
Push Docker image lên Google Artifact Registry và deploy lên Cloud Run service. Cấu hình scaling, memory limits, và environment variables qua Secret Manager.

**Acceptance Criteria:**
- [ ] AC-01: Tạo Artifact Registry repository (region: `asia-southeast1` hoặc `us-central1`)
- [ ] AC-02: Build và push image: `gcloud builds submit --tag=gcr.io/{PROJECT}/{IMAGE}:{TAG}`
- [ ] AC-03: Deploy Cloud Run service với cấu hình:
  - `--max-instances=3` (giới hạn chi phí cho giai đoạn preview)
  - `--min-instances=0` (scale về 0 khi không có traffic — tiết kiệm)
  - `--memory=512Mi`
  - `--cpu=1`
  - `--timeout=60s`
  - `--region=asia-southeast1`
- [ ] AC-04: Tất cả sensitive config (DB password, JWT secret, MQTT password) được inject qua **Google Secret Manager**, không phải env var trực tiếp
- [ ] AC-05: Health endpoint `/health/live` và `/health/ready` trả về `200 OK` sau khi deploy
- [ ] AC-06: API endpoint `/api/v1/devices` trả về dữ liệu từ Cloud SQL (không phải 500 error)
- [ ] AC-07: Ghi lại Cloud Run service URL (dạng `https://inventory-api-xxx-run.app`) vào `docs/deployment/endpoints.md`
- [ ] AC-08: Test cold start time: lần gọi đầu tiên (sau khi scale về 0) phải < 10 giây

**Related Technologies:**
- Google Cloud Run (managed serverless containers)
- Google Artifact Registry
- Google Secret Manager
- Cloud Run YAML config (`infrastructure/cloud/cloudrun.yaml`)

**Notes / Dependencies:**
- Phụ thuộc: INV-SPR06-TASK-002 (Dockerfile sẵn sàng), INV-SPR06-TASK-003 (Cloud SQL ready)
- INV-SPR06-TASK-008 phải hoàn thành trước (secrets setup)

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-19 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR06-TASK-005] — Cấu Hình MQTT Broker Trên Cloud

> **Task ID:** `INV-SPR06-TASK-005`
> **Status:** 📝 DRAFT
> **Created by:** BA
> **Created date:** 2026-04-19
> **Assignee:** —
> **Sprint:** 6

**Description:**
Thay thế Mosquitto Docker container bằng MQTT broker được host trên cloud. Dùng **HiveMQ Cloud** (free tier: 100 simultaneous connections) hoặc **EMQX Cloud** là lựa chọn nhanh nhất cho giai đoạn preview.

**Acceptance Criteria:**
- [ ] AC-01: Tạo tài khoản và cluster MQTT trên **HiveMQ Cloud** (free tier)
  - URL: `{cluster}.hivemq.cloud`
  - Protocol: MQTT over TLS (port 8883)
- [ ] AC-02: Tạo credential cho backend service: username/password cho `inventory_app` user
- [ ] AC-03: Cập nhật config để backend kết nối tới HiveMQ Cloud qua TLS:
  - `MQTT_BROKER=tls://{cluster}.hivemq.cloud`
  - `MQTT_PORT=8883`
  - `MQTT_TLS_ENABLED=true`
- [ ] AC-04: Test kết nối: Dùng `mosquitto_pub` hoặc MQTT Explorer để publish message tới broker cloud và xác nhận backend nhận được
- [ ] AC-05: Tạo credential riêng cho **IoT device / simulator** (không dùng chung với backend credential)
- [ ] AC-06: Ghi lại toàn bộ cấu hình MQTT (không bao gồm password) vào `docs/deployment/mqtt-cloud-setup.md`
- [ ] AC-07: **Fallback option nếu HiveMQ không đáp ứng:** document hướng dẫn chạy Mosquitto trên một VM nhỏ (e2-micro, ~$5/tháng)

**Related Technologies:**
- HiveMQ Cloud (hoặc EMQX Cloud) — MQTT 5.0 broker managed
- MQTT over TLS (`paho.mqtt.golang` với TLS config)
- MQTT Explorer (desktop tool để test)

**Notes / Dependencies:**
- Đây là optional cho giai đoạn preview đầu tiên: có thể dùng **MQTT simulator script** thay vì thiết bị thật
- Nếu khách hàng chỉ cần xem UI và REST API (không cần live IoT data): task này có thể là P2

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-19 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR06-TASK-006] — Firebase Hosting & Rewrites Tới Cloud Run

> **Task ID:** `INV-SPR06-TASK-006`
> **Status:** 📝 DRAFT
> **Created by:** BA
> **Created date:** 2026-04-19
> **Assignee:** —
> **Sprint:** 6

**Description:**
Deploy UI static files lên Firebase Hosting. Cấu hình `firebase.json` rewrites để các request `/api/*` được proxy tới Cloud Run service — khách hàng chỉ cần một URL duy nhất (`https://{project}.web.app`).

**Acceptance Criteria:**
- [ ] AC-01: Khởi tạo Firebase project với Hosting: `firebase init hosting`
- [ ] AC-02: Build UI static files (HTML/CSS/JS từ Vue 3 + Vite hoặc embed.FS) ra thư mục `public/`
- [ ] AC-03: Cấu hình `firebase.json` với rewrites rules:
  ```json
  {
    "hosting": {
      "public": "public",
      "ignore": ["firebase.json", "**/.*", "**/node_modules/**"],
      "rewrites": [
        {
          "source": "/api/**",
          "run": {
            "serviceId": "inventory-api",
            "region": "asia-southeast1"
          }
        },
        {
          "source": "**",
          "destination": "/index.html"
        }
      ]
    }
  }
  ```
- [ ] AC-04: Deploy lần đầu: `firebase deploy --only hosting`
- [ ] AC-05: Kiểm tra URL công khai: `https://{project}.web.app` load được UI
- [ ] AC-06: Kiểm tra rewrite: `https://{project}.web.app/api/v1/devices` trả về dữ liệu từ Go backend (không phải 404)
- [ ] AC-07: Cấu hình custom domain nếu khách hàng yêu cầu (ví dụ: `preview.inventory-client.com`)
- [ ] AC-08: Enable Firebase Hosting **preview channels** để có thể deploy nhiều phiên bản preview:
  ```bash
  firebase hosting:channel:deploy preview-v1
  # tạo URL: https://{project}--preview-v1-{hash}.web.app
  ```

**Related Technologies:**
- Firebase Hosting
- Firebase CLI (`firebase deploy`)
- Cloud Run rewrites (Hosting → Cloud Run)
- Vue 3 + Vite build (nếu UI chưa được build riêng)

**Notes / Dependencies:**
- Phụ thuộc: INV-SPR06-TASK-004 (Cloud Run service phải đang chạy)
- `rewrites` sang Cloud Run chỉ hỗ trợ nếu Firebase project và Cloud Run cùng Google Cloud Project

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-19 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR06-TASK-007] — CI/CD Pipeline Với GitHub Actions

> **Task ID:** `INV-SPR06-TASK-007`
> **Status:** 📝 DRAFT
> **Created by:** BA
> **Created date:** 2026-04-19
> **Assignee:** —
> **Sprint:** 6

**Description:**
Tự động hóa quy trình build → test → deploy mỗi khi có commit lên nhánh `main`. Đảm bảo khách hàng luôn thấy phiên bản mới nhất mà không cần developer deploy thủ công.

**Acceptance Criteria:**
- [ ] AC-01: Tạo file `.github/workflows/deploy.yml` với pipeline:
  ```
  Trigger: push to main
  Jobs:
    1. test:    go test ./... (unit tests)
    2. build:   docker build + push to Artifact Registry
    3. deploy:  gcloud run deploy
    4. hosting: firebase deploy --only hosting
  ```
- [ ] AC-02: Lưu các secrets vào GitHub Secrets (không phải trong code):
  - `GCP_SA_KEY` (Service Account JSON)
  - `FIREBASE_TOKEN`
  - `GCP_PROJECT_ID`
- [ ] AC-03: Job `test` phải pass trước khi `build` và `deploy` chạy (fail-fast)
- [ ] AC-04: Deploy chỉ xảy ra khi push lên nhánh `main`, **không** deploy từ feature branches
- [ ] AC-05: Sau mỗi deploy thành công: post comment lên PR (hoặc Slack/email) với Cloud Run URL mới
- [ ] AC-06: Tổng thời gian pipeline (từ push đến URL sống) < 5 phút
- [ ] AC-07: Viết `docs/deployment/cicd-guide.md` giải thích cách pipeline hoạt động

**Related Technologies:**
- GitHub Actions
- `google-github-actions/deploy-cloudrun`
- `FirebaseExtended/action-hosting-deploy`
- Workload Identity Federation (thay cho SA key JSON — secure hơn)

**Notes / Dependencies:**
- Task này có thể làm sau khi INV-SPR06-TASK-006 đã xong (cần deploy manual ít nhất 1 lần trước)
- P1: không chặn customer preview, nhưng cần có để sustainable

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-19 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR06-TASK-008] — Environment Config & Secrets Management

> **Task ID:** `INV-SPR06-TASK-008`
> **Status:** 📝 DRAFT
> **Created by:** BA
> **Created date:** 2026-04-19
> **Assignee:** —
> **Sprint:** 6

**Description:**
Chuẩn bị hệ thống quản lý config và secrets cho môi trường cloud. Đây là task nền tảng phải hoàn thành trước INV-SPR06-TASK-004.

**Acceptance Criteria:**
- [ ] AC-01: Tạo tất cả secrets trong **Google Secret Manager**:
  ```
  inventory/db-password
  inventory/jwt-secret-key
  inventory/mqtt-password
  inventory/redis-password
  ```
- [ ] AC-02: Cấp quyền cho Cloud Run Service Account đọc các secrets trên:
  ```bash
  gcloud secrets add-iam-policy-binding inventory/db-password \
    --member="serviceAccount:{SA_EMAIL}" \
    --role="roles/secretmanager.secretAccessor"
  ```
- [ ] AC-03: Cập nhật Go application để đọc config theo thứ tự ưu tiên:
  1. Environment variables (ưu tiên cao nhất)
  2. `config/cloud.yaml` (nếu file tồn tại)
  3. `config/local.yaml` (fallback cho local dev)
- [ ] AC-04: Tạo `.env.cloud.example` — template các env vars cần cho cloud deployment (không có giá trị thật)
- [ ] AC-05: Kiểm tra bằng `gitleaks` — không có secret nào bị commit vào git:
  ```bash
  gitleaks detect --source . --verbose
  ```
- [ ] AC-06: Viết `docs/deployment/secrets-guide.md` hướng dẫn cách thêm/rotate secret

**Related Technologies:**
- Google Secret Manager
- Viper (hỗ trợ multiple config sources)
- `gitleaks` (secret scanning)

**Notes / Dependencies:**
- Phải hoàn thành TRƯỚC INV-SPR06-TASK-004
- Tuyệt đối không commit file `.env` có giá trị thật

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-19 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR06-TASK-009] — Customer Preview URL & Smoke Testing

> **Task ID:** `INV-SPR06-TASK-009`
> **Status:** 📝 DRAFT
> **Created by:** BA
> **Created date:** 2026-04-19
> **Assignee:** —
> **Sprint:** 6

**Description:**
Sau khi toàn bộ infrastructure đã deploy, thực hiện kiểm tra smoke test toàn diện và chuẩn bị tài liệu hướng dẫn cho khách hàng truy cập và test sản phẩm.

**Acceptance Criteria:**
- [ ] AC-01: **Smoke Test Checklist** — tất cả phải pass:
  - [ ] `GET https://{project}.web.app` → UI load được, không có lỗi console
  - [ ] `GET https://{project}.web.app/api/v1/devices` → trả về JSON danh sách device
  - [ ] `GET https://{project}.web.app/api/v1/inventory/snapshots` → trả về inventory data
  - [ ] `GET https://{project}.web.app/health/live` → `200 OK`
  - [ ] `GET https://{project}.web.app/health/ready` → `200 OK` (DB connected)
  - [ ] UI navigation: Node List → Node Detail → Calibration (nếu đã có UI)
- [ ] AC-02: Seed **dữ liệu demo thực tế** để khách hàng thấy hệ thống có dữ liệu:
  - Ít nhất 3 device (tên thực tế, location thực tế)
  - Ít nhất 5 SKU với inventory data
  - Ít nhất 7 ngày lịch sử telemetry (dùng script simulate)
- [ ] AC-03: Viết **tài liệu hướng dẫn khách hàng** (`docs/customer-preview-guide.md`):
  - URL truy cập
  - Thông tin đăng nhập demo (nếu có auth)
  - Hướng dẫn các tính năng đã có
  - Các tính năng chưa có trong phiên bản preview
  - Cách để lại feedback
- [ ] AC-04: Chuẩn bị **bản trả lời các câu hỏi thường gặp** của khách hàng khi review
- [ ] AC-05: Đảm bảo URL ổn định ít nhất 2 tuần (không có downtime khi không có deploy)
- [ ] AC-06: Cấu hình **uptime monitoring** đơn giản: dùng Firebase Hosting monitoring hoặc UptimeRobot (free) để alert khi URL down

**Related Technologies:**
- `curl` hoặc Postman cho API smoke test
- UptimeRobot hoặc Google Cloud Monitoring
- Script Python/bash để seed demo data

**Notes / Dependencies:**
- Phụ thuộc: Tất cả tasks INV-SPR06-TASK-001 đến -008 phải CLOSED
- **Đây là milestone cuối — khi task này CLOSED thì khách hàng có thể xem sản phẩm**

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-19 | —    | DRAFT | BA           | Task created |

---

## Thứ Tự Thực Hiện (Dependency Graph)

```
TASK-001 (GCP Setup)
    │
    ├── TASK-008 (Secrets Setup)   ← Phải xong trước TASK-004
    │       │
    │       ▼
    ├── TASK-002 (Docker optimize) ─────────┐
    │                                       │
    ├── TASK-003 (Cloud SQL)                ▼
    │       │                         TASK-004 (Cloud Run Deploy)
    │       └─────────────────────────────►│
    │                                       │
    └── TASK-005 (MQTT Cloud)              │
                                            ▼
                                     TASK-006 (Firebase Hosting)
                                            │
                                            ▼
                                     TASK-007 (CI/CD) — có thể song song
                                            │
                                            ▼
                                     TASK-009 (Smoke Test & Customer URL)
```

**Đường quan trọng (Critical Path):** 001 → 008 → 002 → 003 → 004 → 006 → 009

---

## Quyết Định Kiến Trúc — ADR-006

```markdown
# ADR-006: Chọn Firebase Hosting + Cloud Run thay vì Firebase Functions

## Trạng thái: Đề xuất — 2026-04-19

## Bối cảnh
Khách hàng muốn truy cập sản phẩm qua URL công khai để review.
Go backend hiện tại chạy bằng Docker Compose, cần lên cloud.

## Vấn đề
Firebase Functions chỉ hỗ trợ Node.js và Python — không chạy Go binary.

## Quyết định
Firebase Hosting (static UI) + Cloud Run (Go Docker container)
- Firebase Hosting rewrites /api/* → Cloud Run
- Khách hàng chỉ thấy 1 URL: https://{project}.web.app

## Lý do
✅ Tái sử dụng Docker image hiện có — không viết lại code
✅ Cloud Run + Firebase cùng Google Cloud → rewrites native, latency thấp
✅ Cloud Run free tier: 2M requests/tháng — đủ cho preview
✅ Scale về 0 khi không dùng → không tốn tiền
❌ Cold start ~3-5 giây sau period không có traffic
❌ TimescaleDB không có trên Cloud SQL → cần evaluate

## Các lựa chọn đã xem xét
1. Firebase Functions (Go): KHÔNG CÓ — Google không support
2. Railway.app: simple hơn nhưng không tích hợp Firebase tốt
3. Cloud Run standalone (không Firebase): URL xấu, không có CDN cho UI
4. VM (Compute Engine): phải quản lý server, không serverless
```

---

## Definition of Done — Sprint 6

- [ ] URL công khai `https://{project}.web.app` hoạt động ổn định
- [ ] Khách hàng truy cập được và thấy UI với dữ liệu demo thực tế
- [ ] API `/api/v1/devices`, `/api/v1/inventory/snapshots` trả kết quả đúng
- [ ] Không có secret nào lộ ra trong code hoặc logs
- [ ] `docs/customer-preview-guide.md` đã viết xong
- [ ] Uptime monitoring đã cấu hình
- [ ] Tất cả 9 tasks đạt trạng thái 🔒 CLOSED

---

*Managed by: `.agents/rules/golang-ba.md` | Workflow: `docs/workflows/ba_task_creation_workflow.md`*
*Sprint 6 Created: 2026-04-19 | Priority: CRITICAL — Customer Preview Blocker*
