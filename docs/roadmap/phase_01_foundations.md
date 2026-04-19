# Phase 1 — Nền Tảng & Tư Duy Kiến Trúc

> **Thời lượng:** 3–4 tuần
> **Tiếp theo:** [Phase 2 — Docker](./phase_02_docker.md)
> **Milestone** *(cột mốc)*: Scaffold repo + ADR đầu tiên

---

## 📖 Từ Chuyên Ngành Trong Phase Này

| Từ chuyên ngành | Giải thích |
|---|---|
| **Solution Architect** | Kiến trúc sư giải pháp — người thiết kế tổng thể hệ thống |
| **ADR** (Architecture Decision Record) | Tài liệu ghi lại quyết định kiến trúc: lý do chọn, các lựa chọn đã cân nhắc |
| **12-Factor App** | 12 nguyên tắc vàng để xây dựng ứng dụng cloud-native |
| **DDD** (Domain-Driven Design) | Phương pháp thiết kế phần mềm dựa trên nghiệp vụ thực tế (domain) |
| **Bounded Context** | Ranh giới ngữ nghĩa trong DDD — cùng một từ có thể có nghĩa khác nhau trong các context khác nhau |
| **Aggregate** | Nhóm các đối tượng được coi là một đơn vị trong DDD (ví dụ: Post chứa Tags) |
| **Domain Event** | Sự kiện nghiệp vụ đã xảy ra (ví dụ: `PostPublished`, `UserRegistered`) |
| **CAP Theorem** | Định lý CAP: hệ thống phân tán chỉ đảm bảo được 2 trong 3 tính chất |
| **Monolith** | Kiến trúc nguyên khối — toàn bộ ứng dụng là một khối code duy nhất |
| **Scaffold** | Tạo khung cấu trúc ban đầu cho project |
| **Conventional Commits** | Quy ước đặt tên commit message theo chuẩn |

---

## 1.1 Solution Architect Làm Gì?

**Solution Architect** là cầu nối giữa **yêu cầu nghiệp vụ** và **triển khai kỹ thuật**. Khác với developer tập trung vào *cách viết code*, architect tập trung vào *nên xây dựng hệ thống nào và tại sao*.

| Trách nhiệm | Công việc hàng ngày |
|---|---|
| **System design** | Chuyển yêu cầu nghiệp vụ thành sơ đồ component |
| **Technology selection** | Đánh giá công cụ: monolith vs microservices, SQL vs NoSQL, REST vs gRPC |
| **Non-functional requirements** | Định nghĩa SLA, RTO/RPO, ngưỡng latency, throughput |
| **Cost optimization** | Cân bằng chi phí cloud với SLA hiệu năng |
| **API contracts** | Định nghĩa ranh giới interface giữa các team |
| **Risk assessment** | Xác định điểm lỗi đơn (single point of failure), lỗ hổng bảo mật |
| **Documentation** | Viết ADR, sơ đồ C4, runbook |

> 💡 **Giải thích thuật ngữ:**
> - **SLA** (Service Level Agreement): Cam kết mức độ dịch vụ — ví dụ "99.9% uptime"
> - **RTO** (Recovery Time Objective): Thời gian tối đa để khôi phục sau sự cố
> - **RPO** (Recovery Point Objective): Lượng dữ liệu tối đa có thể mất khi sự cố
> - **Latency**: Độ trễ của hệ thống
> - **Throughput**: Số lượng request xử lý được mỗi giây
> - **Single Point of Failure**: Thành phần mà nếu hỏng thì cả hệ thống sẽ sập
> - **Runbook**: Tài liệu hướng dẫn xử lý sự cố từng bước

### Tư Duy Của Architect

```
Developer hỏi: "Làm thế nào để implement tính năng này?"
Architect hỏi: "Có nên xây tính năng này không? Thuộc service nào?
                Chi phí là bao nhiêu? Nếu hỏng thì ảnh hưởng gì?
                Nếu tải tăng 10 lần thì còn chạy được không?"
```

> **Quy tắc vàng:** Mọi quyết định kiến trúc đều có đánh đổi (tradeoff). Công việc của bạn là làm rõ những đánh đổi đó, không phải tìm giải pháp "hoàn hảo".

---

## 1.2 The 12-Factor App

**12-Factor App** là phương pháp luận định nghĩa "hợp đồng" của một ứng dụng **cloud-native**. Mỗi service trong Blog Engine phải tuân thủ.

| # | Factor | Quy tắc | Ví dụ trong Blog Engine |
|---|--------|------|---------------------|
| 1 | **Codebase** | Một repo, nhiều môi trường deploy | `blog-engine/` monorepo |
| 2 | **Dependencies** | Khai báo rõ ràng, không giả sử | `go.mod`, `package.json` — không dùng package hệ thống |
| 3 | **Config** | Lưu trong biến môi trường | `DB_URL`, `JWT_SECRET` qua `.env` — KHÔNG hardcode |
| 4 | **Backing services** | Coi như resource đính kèm | PostgreSQL, Redis, SMTP có thể thay thế qua config |
| 5 | **Build/Release/Run** | Tách biệt hoàn toàn | `docker build` → tag → `docker run` |
| 6 | **Processes** | Stateless, không chia sẻ | Không dùng file local; session lưu trong Redis |
| 7 | **Port binding** | Tự export HTTP | `post-service` tự bind port 8080 |
| 8 | **Concurrency** | Scale bằng cách tăng process | `docker compose --scale post-service=3` |
| 9 | **Disposability** | Khởi động nhanh + tắt graceful | SIGTERM → 30s drain → exit |
| 10 | **Dev/Prod parity** | Môi trường dev ≈ production | Cùng Docker image ở dev, staging, prod |
| 11 | **Logs** | Coi log là stream | JSON ra stdout → agent thu thập |
| 12 | **Admin processes** | Script chạy một lần | `migrate up` chạy như container riêng |

> 💡 **Giải thích thuật ngữ:**
> - **Hardcode**: Ghi thẳng giá trị vào code (ví dụ: `host = "localhost"`) — rất nguy hiểm
> - **Stateless**: Không lưu trạng thái trong bộ nhớ của process — mỗi request độc lập
> - **Graceful shutdown**: Tắt lịch sự — cho phép các request đang xử lý hoàn thành trước khi tắt
> - **SIGTERM**: Tín hiệu Unix ra lệnh cho process dừng lại một cách "nhẹ nhàng"
> - **Stream**: Luồng dữ liệu liên tục — log được coi như stream để dễ thu thập và phân tích

### Các Vi Phạm Thường Gặp (và cách sửa)

```
❌ Hardcode DB host trong source code
✅ Dùng biến môi trường DB_HOST, inject lúc runtime

❌ Ghi file upload lên disk của container
✅ Ghi lên S3/MinIO (stateless process)

❌ Dockerfile khác nhau giữa staging và production
✅ Một image duy nhất, môi trường kiểm soát qua env var

❌ Chạy migration trong code khởi động app
✅ Container migration riêng, chạy trước app
```

---

## 1.3 Kiến Trúc Monolith vs Microservices

### Cây Quyết Định

```
Team có dưới 5 kỹ sư không?
  └─ CÓ → Bắt đầu với Monolith. Không bàn cãi.
     (microservices có overhead vận hành: networking, tracing, v.v.)

Có nhu cầu rõ ràng để deploy các phần độc lập không?
  └─ KHÔNG → Giữ monolith.
  └─ CÓ → Tìm bounded context. Tách theo ranh giới domain.

Các phần có nhu cầu scale khác nhau nhiều không?
  └─ CÓ → Cân nhắc microservices chỉ cho các "hot path".

Team phân tán ở các đơn vị tổ chức khác nhau không?
  └─ CÓ → Microservices cho phép tự chủ theo team.
```

> **Với Blog Engine:** Chúng ta xây microservices vì **mục đích học tập**, nhưng phải nhận ra rằng một startup thực tế sẽ ship monolith trước.

> 💡 **Giải thích thuật ngữ:**
> - **Hot path**: Đường dẫn code được thực thi nhiều nhất, cần hiệu năng cao nhất
> - **Overhead**: Chi phí phụ thêm (thời gian, tài nguyên) do một quyết định kỹ thuật gây ra
> - **Deploy**: Triển khai ứng dụng lên server/môi trường

---

## 1.4 Domain-Driven Design (DDD) — Bounded Context

**DDD** là nền tảng để quyết định **ranh giới service** ở đâu.

### Khái Niệm Chính

| Khái niệm | Định nghĩa | Ví dụ Blog Engine |
|---|---|---|
| **Domain** | Lĩnh vực nghiệp vụ phần mềm hoạt động trong đó | Xuất bản nội dung online |
| **Bounded Context** | Ranh giới ngôn ngữ — cùng từ nhưng nghĩa khác | "User" trong auth ≠ "User" trong bài viết |
| **Aggregate** | Nhóm đối tượng được coi là một đơn vị | `Post` aggregate chứa `Tags`, `Metadata` |
| **Domain Event** | Sự kiện nghiệp vụ đã xảy ra | `PostPublished`, `CommentCreated` |
| **Anti-Corruption Layer** | Tầng dịch giữa hai bounded context | `post-service` map `UserID` từ user-service |

### Blog Engine — Bản Đồ Bounded Context

```
┌─────────────────────┐   ┌─────────────────────┐
│  Identity Context   │   │  Content Context    │
│  (user-service)     │   │  (post-service)     │
│                     │   │                     │
│ - Đăng ký           │   │ - Tạo bài viết      │
│ - Đăng nhập/Token   │   │ - Publish bài viết  │
│ - Profile           │   │ - Quản lý tag       │
└─────────────────────┘   └─────────────────────┘
         │ UserID                  │ PostPublished event
         │                         ▼
┌─────────────────────┐   ┌─────────────────────┐
│ Engagement Context  │   │Notification Context │
│ (comment-service)   │   │(notification-svc)   │
│                     │   │                     │
│ - Tạo comment       │   │ - Email khi có bài  │
│ - Kiểm duyệt        │   │ - Push notification │
└─────────────────────┘   └─────────────────────┘
```

> 💡 **Giải thích thuật ngữ:**
> - **Anti-Corruption Layer (ACL)**: Lớp chống ô nhiễm — dịch/chuyển đổi dữ liệu giữa hai context để tránh phụ thuộc trực tiếp
> - **Push notification**: Thông báo đẩy đến thiết bị di động (iOS/Android)

---

## 1.5 CAP Theorem

Mọi hệ thống phân tán phải chọn 2 trong 3 tính chất:

```
        Consistency (Nhất quán)
              △
             / \
            /   \
           /     \
Partition ─────── Availability
Tolerance         (Sẵn sàng)
(Chịu phân vùng)
```

| Lựa chọn | Bạn nhận được | Khi nào dùng |
|---|---|---|
| **CP** (Consistency + Partition Tolerance) | Luôn trả về dữ liệu đúng hoặc báo lỗi | Giao dịch tài chính, hệ thống tồn kho |
| **AP** (Availability + Partition Tolerance) | Luôn trả về dữ liệu (có thể cũ) | Social feed, danh mục sản phẩm |
| **CA** (Consistency + Availability) | Không thể đạt trong hệ thống phân tán | Chỉ single-node |

> 💡 **Giải thích thuật ngữ:**
> - **Consistency**: Nhất quán — mọi node đều thấy cùng một dữ liệu tại cùng thời điểm
> - **Availability**: Sẵn sàng — hệ thống luôn phản hồi (không bao giờ báo lỗi)
> - **Partition Tolerance**: Chịu phân vùng — hệ thống vẫn hoạt động khi mạng bị tắt giữa chừng
> - **Node**: Một máy chủ trong hệ thống phân tán
> - **Eventual Consistency**: Nhất quán cuối cùng — dữ liệu sẽ nhất quán sau một khoảng thời gian (không tức thì)

**Quyết định cho Blog Engine:**
- Bảng `posts` → **CP** (PostgreSQL) — bài đăng phải chính xác
- `view counts`, `like counts` → **AP** (Redis) — nhất quán cuối cùng là chấp nhận được
- Index tìm kiếm (Elasticsearch) → **AP** — trễ vài giây không sao

---

## 1.6 Linux CLI Cần Thiết

```bash
# Hệ thống file
ls -la /etc/
find / -name "*.log" 2>/dev/null
grep -rn "DB_URL" ./
cat /var/log/syslog | tail -100

# Process & Network
ps aux | grep post-service
lsof -i :8080                 # ai đang dùng port 8080?
ss -tlnp                      # tất cả TCP socket đang listen
curl -v http://localhost:8080/health
nc -zv postgres 5432          # test kết nối TCP
nslookup postgres             # phân giải DNS

# Trong container
docker exec -it post-service sh
wget -qO- http://user-service:8080/health
```

> 💡 **Giải thích thuật ngữ:**
> - **CLI** (Command Line Interface): Giao diện dòng lệnh
> - **Port**: Cổng mạng — như "cửa ra vào" của ứng dụng trên máy chủ
> - **TCP** (Transmission Control Protocol): Giao thức truyền tin đáng tin cậy
> - **DNS** (Domain Name System): Hệ thống phân giải tên miền thành địa chỉ IP
> - **Socket**: Điểm cuối kết nối mạng

---

## 1.7 HTTP Cần Nắm

```bash
# HTTP Methods (phương thức)
GET    /posts         → Đọc (an toàn, idempotent)
POST   /posts         → Tạo mới (không idempotent)
PUT    /posts/:id     → Thay thế hoàn toàn (idempotent)
PATCH  /posts/:id     → Cập nhật một phần
DELETE /posts/:id     → Xóa (idempotent)

# Status Codes (mã trạng thái — PHẢI THUỘC LÒNG)
200 OK              → thành công
201 Created         → tạo thành công (POST)
204 No Content      → thành công, không có body (DELETE)
400 Bad Request     → lỗi validation phía client
401 Unauthorized    → thiếu/sai token xác thực
403 Forbidden       → đã xác thực nhưng không có quyền
404 Not Found       → resource không tồn tại
409 Conflict        → resource bị trùng lặp
422 Unprocessable   → vi phạm business rule
429 Too Many Requests → bị rate limit
500 Internal Server Error → bug của chúng ta
502 Bad Gateway     → service upstream lỗi
503 Service Unavailable → quá tải / đang khởi động
```

> 💡 **Giải thích thuật ngữ:**
> - **Idempotent**: Gọi nhiều lần cho kết quả giống nhau — PUT xóa 1 bài nhiều lần vẫn ra kết quả giống nhau
> - **Resource**: Tài nguyên trong API — ví dụ: một bài viết, một người dùng
> - **Body**: Phần nội dung của HTTP request/response
> - **Rate limit**: Giới hạn số lượng request trong một khoảng thời gian
> - **Business rule**: Quy tắc nghiệp vụ — ví dụ: "không thể publish bài trống"

---

## 1.8 Git — Trunk-Based Development

> 💡 **Trunk-Based Development**: Chiến lược làm việc với Git — tất cả dev đều commit thường xuyên về một nhánh chính (`main`), branch tính năng tồn tại ngắn (< 2 ngày).

```bash
# Bắt đầu từ main mới nhất
git checkout main && git pull origin main

# Branch tính năng tồn tại ngắn
git checkout -b feature/post-crud-api

# Commit tập trung (Conventional Commits)
git commit -m "feat(post): add Post model with validation"
git commit -m "feat(post): implement PostService CRUD"
git commit -m "test(post): add table-driven unit tests for PostService"

# Push và mở PR → squash merge
git push origin feature/post-crud-api
```

### Các Loại Conventional Commits

| Loại | Khi dùng |
|---|---|
| `feat` | Tính năng mới |
| `fix` | Sửa bug |
| `docs` | Tài liệu |
| `refactor` | Tái cấu trúc code, không thay đổi hành vi |
| `test` | Thêm test |
| `chore` | Build, deps, tooling |
| `perf` | Cải thiện hiệu năng |

> 💡 **Giải thích thuật ngữ:**
> - **PR** (Pull Request): Yêu cầu merge code — để review trước khi merge vào main
> - **Squash merge**: Gộp nhiều commit thành 1 trước khi merge
> - **Branch**: Nhánh code — làm việc độc lập, không ảnh hưởng nhánh chính

---

## 1.9 Architecture Decision Records (ADRs)

**ADR** là tài liệu ghi lại quyết định kiến trúc — *tại sao* chọn thứ này thay vì thứ kia.

```markdown
# ADR-001: Microservices vs Monolith cho Blog Engine

## Trạng thái
Đã chấp nhận — 2026-04-18

## Bối cảnh
Đang xây dựng Blog Engine làm dự án học tập.
Quy mô team: 1 developer. Timeline: 6 tháng.

## Quyết định
Triển khai microservices vì mục đích giáo dục.
Trong startup thực tế với 1–3 người, sẽ bắt đầu bằng monolith.

## Hệ quả
✅ Học service boundaries, API contracts, event-driven patterns
✅ Học Docker Compose orchestration nhiều service
❌ Overhead vận hành cao hơn
❌ Phức tạp hơn cho CRUD đơn giản

## Các Lựa Chọn Đã Cân Nhắc
- Modular monolith: đơn giản hơn nhưng không dạy giao tiếp giữa service
- Serverless: quá khác so với mục tiêu (Go + Docker)
```

> 💡 **Giải thích thuật ngữ:**
> - **API contract**: "Hợp đồng" giữa hai service — định nghĩa request/response format
> - **Event-driven**: Hướng sự kiện — service giao tiếp qua việc phát/nhận sự kiện
> - **Serverless**: Mô hình điện toán không quản lý server trực tiếp (ví dụ: AWS Lambda)
> - **Modular monolith**: Monolith được tổ chức theo module rõ ràng nhưng vẫn là một ứng dụng

---

## 1.10 Blog Engine — Lab Milestone 1

```
[ ] 1. Khởi tạo repo với cấu trúc thư mục bên dưới
[ ] 2. Viết ADR-001: Microservices vs Monolith
[ ] 3. Viết ADR-002: PostgreSQL vs MongoDB cho lưu trữ bài viết
[ ] 4. Tạo README.md với sơ đồ tổng quan hệ thống
[ ] 5. Khởi tạo Go module cho từng service (go mod init)
[ ] 6. Tạo .gitignore
[ ] 7. Tạo .env.example với tất cả biến cần thiết
[ ] 8. Vẽ bản đồ bounded context (ASCII art là được)
```

### Cấu Trúc Repository

```
blog-engine/
├── services/
│   ├── post-service/
│   │   ├── cmd/server/main.go
│   │   ├── config/local.yaml
│   │   ├── internal/
│   │   │   ├── model/
│   │   │   ├── service/
│   │   │   ├── repository/
│   │   │   └── controller/
│   │   ├── migrations/
│   │   ├── go.mod
│   │   └── Dockerfile
│   ├── user-service/         (cấu trúc tương tự)
│   ├── comment-service/      (cấu trúc tương tự)
│   └── notification-service/ (cấu trúc tương tự)
├── frontend/
│   ├── app/
│   ├── components/
│   ├── lib/
│   └── package.json
├── infrastructure/
│   ├── docker-compose.yml
│   ├── docker-compose.monitoring.yml
│   └── traefik/
├── docs/
│   ├── adr/
│   │   ├── ADR-001-microservices-vs-monolith.md
│   │   └── ADR-002-postgres-vs-mongodb.md
│   └── diagrams/
├── .env.example
├── .gitignore
└── README.md
```

---

## 1.11 Tài Nguyên

| Tài nguyên | URL | Độ ưu tiên |
|---|---|---|
| 12-Factor App | https://12factor.net | ⭐ Đọc đầu tiên |
| Hướng dẫn ADR | https://adr.github.io | Cao |
| DDD by Martin Fowler | https://martinfowler.com/tags/domain%20driven%20design.html | Cao |
| CAP Theorem | https://www.ibm.com/topics/cap-theorem | Trung bình |
| The Linux Command Line (PDF miễn phí) | https://linuxcommand.org/tlcl.php | Cao |
| Pro Git (sách miễn phí) | https://git-scm.com/book | Cao |
| Conventional Commits | https://www.conventionalcommits.org | Trung bình |
| C4 Model | https://c4model.com | Trung bình |

---

## 1.12 Tự Kiểm Tra — Phase 1 Hoàn Thành Khi…

```
[ ] Có thể giải thích sự khác biệt giữa Solution Architect và Senior Developer trong < 2 phút
[ ] Có thể đọc tên cả 12 factor và cho ví dụ Blog Engine với mỗi factor
[ ] Có thể vẽ bản đồ bounded context của Blog Engine từ trí nhớ
[ ] Có thể giải thích CAP theorem và nêu lựa chọn của từng service
[ ] Thoải mái điều hướng container đang chạy bằng Linux CLI
[ ] Repository scaffold đã được tạo với cấu trúc đúng
[ ] Đã viết hai ADR (ADR-001 và ADR-002)
```

---

*← [Index](./index.md) | [Phase 2 — Docker →](./phase_02_docker.md)*
