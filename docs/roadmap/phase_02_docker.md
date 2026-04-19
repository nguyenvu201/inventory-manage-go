# Phase 2 — Docker & Containerization

> **Thời lượng:** 3–4 tuần
> **Trước đó:** [Phase 1 — Nền Tảng](./phase_01_foundations.md)
> **Tiếp theo:** [Phase 3 — Docker Compose](./phase_03_docker_compose.md)
> **Milestone:** Tất cả service Blog Engine được đóng gói, image production < 20 MB

---

## 📖 Từ Chuyên Ngành Trong Phase Này

| Từ chuyên ngành | Giải thích |
|---|---|
| **Container** | "Hộp" cô lập chứa ứng dụng + thư viện, dùng chung kernel với host |
| **Image** | Bản thiết kế của container — bất biến, có thể chia sẻ |
| **Dockerfile** | File script mô tả cách build một Docker image |
| **Layer** | Mỗi lệnh trong Dockerfile tạo ra một tầng (layer) — được cache |
| **Multi-stage build** | Kỹ thuật dùng nhiều stage trong Dockerfile để tách môi trường build và runtime |
| **Distroless** | Image không có shell, package manager, chỉ có binary — cực kỳ nhỏ và an toàn |
| **Volume** | Bộ nhớ bền vững được gắn vào container |
| **Bind mount** | Map thư mục từ máy host vào container |
| **Network bridge** | Mạng ảo Docker cho phép container giao tiếp với nhau |
| **Namespace** | Tính năng Linux cô lập tài nguyên (PID, network, filesystem...) |
| **cgroups** | Tính năng Linux giới hạn tài nguyên (CPU, RAM) cho container |
| **Registry** | Kho lưu trữ Docker image (Docker Hub, AWS ECR, GCR...) |
| **.dockerignore** | File khai báo những thứ không copy vào image (giống .gitignore) |

---

## 2.1 Container Thực Sự Hoạt Động Như Thế Nào

Container **không phải máy ảo (VM)**. Chúng là **process Linux được cô lập**.

### Stack Công Nghệ

```
Ứng dụng của bạn
      │
  Container Runtime (containerd / runc)
      │
  Tính năng Linux Kernel:
    ├── Namespaces     → cô lập process (PID, NET, MNT, UTS, IPC, USER)
    ├── cgroups        → giới hạn tài nguyên (CPU, RAM, I/O)
    └── Union FS (overlay2) → hệ thống file theo tầng (layer)
```

### Namespace — Cô Lập Gì

| Namespace | Cô lập | Ví dụ |
|---|---|---|
| `pid` | Process ID | Container thấy process của nó là PID 1 |
| `net` | Network interface | Container có `eth0` riêng |
| `mnt` | Mount point | Container có filesystem riêng |
| `uts` | Hostname | Container có hostname `post-service` riêng |
| `user` | UID/GID mapping | `root` trong container = UID 1000 trên host |

### Container vs VM (Máy Ảo)

| | Container | VM (Máy ảo) |
|---|---|---|
| Thời gian khởi động | ~100ms | 10–60 giây |
| Overhead bộ nhớ | ~5 MB | 500 MB – 2 GB |
| Kernel | Dùng chung kernel host | Kernel guest riêng |
| Cô lập | Mức process | Mức phần cứng |
| Kích thước image | 5 MB – 200 MB | 1 GB – 20 GB |

---

## 2.2 Docker Image Layers

Mỗi lệnh trong Dockerfile tạo ra một **layer chỉ đọc**. Các layer được cache và chia sẻ.

### Tối Ưu Cache — Thứ Tự Quan Trọng

```dockerfile
# ❌ SAI — thay đổi source code sẽ invalidate cache go mod download
COPY . .
RUN go mod download
RUN go build -o /server ./cmd

# ✅ ĐÚNG — go.mod/go.sum trước → deps được cache trừ khi chúng thay đổi
COPY go.mod go.sum ./
RUN go mod download          ← cache này, chỉ invalidate khi go.mod thay đổi
COPY . .                     ← chỉ re-run từ đây khi code thay đổi
RUN go build -o /server ./cmd
```

> 💡 **Giải thích thuật ngữ:**
> - **Cache invalidation**: Khi cache bị làm mất hiệu lực và phải tính lại — Docker cache layer hoạt động từ trên xuống: một layer thay đổi thì mọi layer phía sau đều phải build lại
> - **go mod download**: Lệnh Go tải dependencies về — chỉ cần chạy khi `go.mod` thay đổi

### Phân Tích Layers

```bash
docker history blog/post-service:1.0.0

# Tốt hơn: dùng dive để xem trực quan
brew install dive
dive blog/post-service:1.0.0
```

---

## 2.3 Multi-Stage Builds

Tách hoàn toàn **môi trường build** khỏi **môi trường runtime**.

### Go Service (post-service)

```dockerfile
# ── Stage 1: Build ──────────────────────────────────────
FROM golang:1.23-alpine AS builder
# alpine: biến thể Linux cực nhỏ (~5 MB), đủ để build Go

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache dependencies riêng
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Build binary
COPY . .
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
      -ldflags="-w -s -extldflags '-static'" \
      -trimpath \
      -o /server \
      ./cmd/server
# CGO_ENABLED=0: tắt CGO để binary hoàn toàn static (không phụ thuộc C library)
# -w -s: bỏ debug info → binary nhỏ hơn
# -trimpath: bỏ đường dẫn source khỏi binary → bảo mật

# ── Stage 2: Runtime (distroless — không có shell, không có package manager) ──
FROM gcr.io/distroless/static-debian12
# distroless: image không có shell, không có apt/apk
# Nếu bị hack, kẻ tấn công không có shell để chạy lệnh!

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /server /server

# KHÔNG BAO GIỜ chạy với quyền root
USER nonroot:nonroot

EXPOSE 8080
ENTRYPOINT ["/server"]
```

**So sánh kích thước:**
```
golang:1.23-alpine + full build:   ~600 MB  ❌ quá lớn
distroless/static + binary only:   ~10 MB   ✅ tối ưu
```

### Next.js Frontend (3 stage)

```dockerfile
# Stage 1: Dependencies
FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --only=production
# npm ci: install chính xác theo lock file (không thay đổi phiên bản)

# Stage 2: Build
FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# Stage 3: Runtime tối ưu
FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup -g 1001 -S nodejs
RUN adduser -S nextjs -u 1001

COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
COPY --from=builder --chown=nextjs:nodejs /app/public ./public

USER nextjs
EXPOSE 3000
CMD ["node", "server.js"]
```

---

## 2.4 Volumes — Lưu Trữ Bền Vững

| Loại | Lệnh | Khi dùng |
|---|---|---|
| Named volume | `-v pgdata:/var/lib/postgresql/data` | Persist DB, Docker quản lý |
| Bind mount | `-v ./config:/app/config:ro` | Dev hot-reload, inject config |
| tmpfs | `--tmpfs /tmp:rw,size=100m` | Secrets trong RAM, xử lý tạm thời |

> 💡 **Giải thích thuật ngữ:**
> - **Named volume**: Volume có tên cụ thể, Docker lưu ở `/var/lib/docker/volumes/` — persist qua restart
> - **Bind mount**: Map trực tiếp thư mục host vào container — thay đổi file là thấy ngay
> - **tmpfs**: Bộ nhớ tạm thời trong RAM — khi container dừng, dữ liệu mất đi
> - **:ro**: Read-only — container chỉ được đọc, không được ghi
> - **Persist**: Lưu dữ liệu bền vững — không bị mất khi container bị xóa

```bash
# Tạo và kiểm tra named volume
docker volume create pgdata
docker volume inspect pgdata

# Backup một volume
docker run --rm \
  -v pgdata:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/pgdata-backup.tar.gz /data
```

---

## 2.5 Docker Networks

### Custom Bridge Networks (luôn dùng cái này!)

```bash
# Container trên user-defined bridge có thể giao tiếp qua TÊN CONTAINER
docker network create --driver bridge blog-net

docker run -d --name postgres --network blog-net postgres:16
docker run -d --name post-svc  --network blog-net \
  -e DB_HOST=postgres \    # ← DNS tự động resolve "postgres" thành IP
  blog/post-service:1.0.0
```

> **Tại sao không dùng default bridge?** Default `bridge` KHÔNG hỗ trợ DNS theo tên container.

> 💡 **Giải thích thuật ngữ:**
> - **Bridge network**: Mạng ảo Docker — như switch ảo kết nối các container
> - **DNS resolution**: Quá trình dịch tên (như "postgres") thành địa chỉ IP
> - **Default bridge**: Network mặc định của Docker — container chỉ giao tiếp được qua IP, không qua tên

### Network Segmentation (Phân Đoạn Mạng)

```bash
# Mạng nội bộ — không có internet
docker network create --internal blog-data

# Mạng public-facing
docker network create blog-public

# Kết nối post-service với cả hai
docker network connect blog-data post-service
docker network connect blog-public post-service
```

> 💡 **--internal**: Cờ này ngăn container trong mạng này kết nối ra internet — rất quan trọng cho bảo mật database!

---

## 2.6 Bảo Mật Container

```dockerfile
# ✅ Dùng non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# ✅ Distroless runtime (không có shell)
FROM gcr.io/distroless/static-debian12

# ✅ Filesystem chỉ đọc (set trong compose)
read_only: true

# ✅ Bỏ toàn bộ capabilities của Linux
cap_drop: [ALL]
```

### .dockerignore

```dockerignore
.git/
.gitignore
.env
*.pem
*.key
secrets/
bin/
dist/
coverage.out
.idea/
.vscode/
docs/
*.md
*_test.go
test/
```

> 💡 **Giải thích thuật ngữ:**
> - **Capabilities**: Quyền hạn đặc biệt của Linux process (ví dụ: bind port < 1024, raw socket...) — bỏ hết để giảm bề mặt tấn công
> - **Attack surface** (bề mặt tấn công): Tổng hợp các điểm mà kẻ tấn công có thể khai thác
> - **Read-only filesystem**: Container không thể ghi file — nếu bị hack, kẻ tấn công không thể để lại backdoor
> - **Backdoor**: Cửa hậu — phần mềm độc hại ẩn giúp kẻ tấn công vào lại hệ thống

### Checklist Bảo Mật Dockerfile

```
[ ] Không dùng tag :latest — luôn pin phiên bản cụ thể
[ ] apk/apt: dùng flag --no-cache
[ ] Chỉ copy những gì cần thiết (có .dockerignore)
[ ] Không bake secrets vào image layers
[ ] Có directive USER nonroot trước ENTRYPOINT
[ ] Dùng COPY không dùng ADD
[ ] Image base tối giản (distroless > alpine > ubuntu)
```

> 💡 **Giải thích thuật ngữ:**
> - **Pin phiên bản**: Dùng tag cụ thể như `postgres:16.2-alpine3.19` thay vì `postgres:latest` — tránh bất ngờ khi image thay đổi
> - **Bake secrets**: Nhúng secret vào image — rất nguy hiểm vì image có thể bị share

---

## 2.7 Tham Khảo Lệnh Docker

```bash
# ─── Images ──────────────────────────────────────────
docker build -t name:tag .          # build image
docker images                       # liệt kê image
docker rmi image:tag                # xóa image
docker image prune -a               # xóa image không dùng

# ─── Containers ───────────────────────────────────────
docker run -d -p 8080:8080 --name app image:tag   # chạy container
docker ps                           # liệt kê đang chạy
docker ps -a                        # liệt kê tất cả
docker stop container_name          # dừng nhẹ nhàng (SIGTERM)
docker kill container_name          # buộc dừng (SIGKILL)
docker rm container_name            # xóa container
docker container prune              # xóa container dừng

# ─── Debug ────────────────────────────────────────────
docker logs -f container_name       # xem log realtime
docker logs --tail=100 container_name
docker exec -it container_name sh   # vào container
docker inspect container_name       # xem thông tin chi tiết
docker stats                        # xem CPU/RAM realtime
docker top container_name           # xem processes trong container

# ─── Networks & Volumes ───────────────────────────────
docker network ls
docker network create my-net
docker volume ls
docker volume create pgdata
docker volume prune                 # xóa volume không dùng
```

---

## 2.8 Blog Engine — Lab Milestone 2

```
[ ] 1. Viết multi-stage Dockerfile cho post-service (Go)
        - Dùng golang:1.23-alpine làm builder
        - Dùng distroless/static làm runtime
        - Kiểm tra image cuối < 20 MB (docker images)
        - Kiểm tra chạy với nonroot (docker inspect)

[ ] 2. Viết multi-stage Dockerfile cho user-service (Go, cùng pattern)

[ ] 3. Viết multi-stage Dockerfile cho frontend (Next.js)
        - Kiểm tra image cuối < 150 MB

[ ] 4. Tạo .dockerignore cho từng service

[ ] 5. Build và chạy với custom network
        docker network create blog-dev
        docker run -d --name postgres --network blog-dev \
          -e POSTGRES_PASSWORD=dev postgres:16
        docker run -d --name post-svc --network blog-dev \
          -e DB_HOST=postgres -p 8081:8080 blog/post-service:dev

[ ] 6. Kiểm tra kết nối
        curl http://localhost:8081/health/live
        docker exec -it post-svc nc -zv postgres 5432

[ ] 7. Phân tích image layers với dive
        dive blog/post-service:dev
        → Tối ưu nếu wasted space > 1 MB
```

---

## 2.9 Lỗi Thường Gặp & Cách Sửa

| Lỗi | Cách sửa |
|---|---|
| `FROM golang:1.23` là runtime | Dùng multi-stage, distroless runtime |
| `COPY . .` trước `go mod download` | Copy `go.mod go.sum` trước |
| Chạy với quyền root | Thêm `USER nonroot:nonroot` |
| Dùng tag `:latest` | Pin phiên bản cụ thể: `postgres:16.2-alpine3.19` |
| Bake `.env` vào image | Dùng `--env-file` lúc runtime |
| Không có `.dockerignore` | Thêm ngay — `.git` một mình đã tiết kiệm ~500 MB |

---

## 2.10 Tài Nguyên

| Tài nguyên | URL |
|---|---|
| Docker Official Docs | https://docs.docker.com |
| Best practices Dockerfile | https://docs.docker.com/build/building/best-practices/ |
| Distroless images | https://github.com/GoogleContainerTools/distroless |
| Dive (phân tích layer) | https://github.com/wagoodman/dive |
| Hướng dẫn Docker cho Go | https://docs.docker.com/language/golang/ |
| Ví dụ Docker cho Next.js | https://github.com/vercel/next.js/tree/main/examples/with-docker |

---

## 2.11 Tự Kiểm Tra — Phase 2 Hoàn Thành Khi…

```
[ ] Có thể giải thích Container vs VM trong < 2 phút
[ ] Có thể giải thích tại sao cần multi-stage builds
[ ] Cả 3 service đều có Dockerfile
[ ] Go image < 20 MB, Next.js image < 150 MB
[ ] Không có image nào chạy với quyền root
[ ] Tất cả container trên custom bridge network (không phải default)
[ ] Đã chạy dive trên ít nhất một image và tối ưu nó
```

---

*← [Phase 1 — Nền Tảng](./phase_01_foundations.md) | [Phase 3 — Docker Compose →](./phase_03_docker_compose.md)*
