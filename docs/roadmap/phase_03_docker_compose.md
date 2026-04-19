# Phase 3 — Docker Compose & Môi Trường Đa Dịch Vụ

> **Thời lượng:** 2–3 tuần
> **Trước đó:** [Phase 2 — Docker](./phase_02_docker.md)
> **Tiếp theo:** [Phase 4 — Microservices](./phase_04_microservices.md)
> **Milestone:** Toàn bộ stack Blog Engine chạy local với healthcheck, thứ tự khởi động đúng, và quản lý môi trường

---

## 📖 Từ Chuyên Ngành Trong Phase Này

| Từ chuyên ngành | Giải thích |
|---|---|
| **Docker Compose** | Công cụ định nghĩa và chạy nhiều container bằng một file YAML |
| **Healthcheck** | Kiểm tra định kỳ xem service có hoạt động tốt không |
| **depends_on** | Khai báo phụ thuộc giữa service — service A chờ service B healthy mới khởi động |
| **Named volume** | Volume có tên, Docker quản lý — data persist qua restart |
| **Network segmentation** | Phân đoạn mạng — giới hạn service nào được giao tiếp với service nào |
| **Environment variable** | Biến môi trường — cách inject cấu hình vào container mà không hardcode |
| **Profile** | Nhóm service tùy chọn trong Compose — chỉ khởi động khi được chỉ định |
| **Override file** | File Compose phụ dùng để ghi đè cấu hình (dùng cho dev, staging, prod) |
| **Scale** | Chạy nhiều bản sao của một service để xử lý nhiều traffic hơn |
| **Load balancing** | Phân phối đều request đến nhiều instance của một service |

---

## 3.1 Docker Compose Giải Quyết Vấn Đề Gì

```bash
# Không có Docker Compose (rất cực):
docker network create blog-net
docker run -d --name postgres --network blog-net -e POSTGRES_PASSWORD=dev postgres:16
docker run -d --name redis    --network blog-net redis:7
docker run -d --name post-svc --network blog-net \
  -e DB_HOST=postgres -e REDIS_HOST=redis blog/post-service:dev
# ... lặp lại cho 4 service nữa

# Với Docker Compose: một file, một lệnh
docker compose up -d
```

Docker Compose định nghĩa **toàn bộ application stack** dưới dạng code (YAML).

---

## 3.2 Cấu Trúc File Compose

```yaml
version: "3.9"

services:

  # ── Database ─────────────────────────────────────────────────────────────────
  postgres:
    image: postgres:16-alpine
    container_name: blog_postgres
    restart: unless-stopped         # tự động restart khi crash, trừ khi dừng thủ công
    environment:
      POSTGRES_USER: ${DB_USER}     # đọc từ file .env
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: blog
    volumes:
      - pgdata:/var/lib/postgresql/data      # persist data
      - ./infrastructure/init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    ports:
      - "5432:5432"     # Chỉ dùng trong dev! Không expose trong production.
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER} -d blog"]
      interval: 10s     # kiểm tra mỗi 10 giây
      timeout: 5s       # timeout nếu không phản hồi trong 5 giây
      retries: 5        # thử lại 5 lần trước khi báo unhealthy
      start_period: 30s # bỏ qua healthcheck trong 30s đầu (để postgres khởi động)
    networks:
      - blog-data       # chỉ kết nối với mạng data — không cần internet
    logging:
      driver: "json-file"
      options:
        max-size: "10m"   # file log tối đa 10 MB
        max-file: "3"     # giữ tối đa 3 file log

  # ── Cache ─────────────────────────────────────────────────────────────────────
  redis:
    image: redis:7-alpine
    container_name: blog_redis
    restart: unless-stopped
    command: >
      redis-server
      --requirepass ${REDIS_PASSWORD}
      --maxmemory 256mb
      --maxmemory-policy allkeys-lru    # xóa key ít dùng nhất khi hết memory
    volumes:
      - redisdata:/data
    healthcheck:
      test: ["CMD", "redis-cli", "--no-auth-warning", "-a", "${REDIS_PASSWORD}", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3
    networks:
      - blog-data

  # ── Post Service ─────────────────────────────────────────────────────────────
  post-service:
    build:
      context: ./services/post-service
      dockerfile: Dockerfile
    image: blog/post-service:local
    container_name: blog_post_service
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy     # ← CHỜ postgres healthcheck pass
      redis:
        condition: service_healthy
    environment:
      - SERVER_PORT=8080
      - DB_URL=postgres://${DB_USER}:${DB_PASSWORD}@postgres:5432/blog?sslmode=disable
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - LOG_LEVEL=${LOG_LEVEL:-info}   # mặc định là "info" nếu không set
    ports:
      - "8081:8080"     # Chỉ dùng dev
    networks:
      - blog-data       # giao tiếp với DB/cache
      - blog-services   # giao tiếp với các service khác và gateway
    read_only: true     # filesystem chỉ đọc — bảo mật
    tmpfs:
      - /tmp:rw,size=50m  # thư mục /tmp trong RAM
    security_opt:
      - no-new-privileges:true   # ngăn leo thang quyền
    labels:
      # Traefik tự động đọc nhãn này để tạo route
      - "traefik.enable=true"
      - "traefik.http.routers.posts.rule=Host(`localhost`) && PathPrefix(`/api/posts`)"
      - "traefik.http.services.posts.loadbalancer.server.port=8080"
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  # ── API Gateway ──────────────────────────────────────────────────────────────
  gateway:
    image: traefik:v3.0
    container_name: blog_gateway
    restart: unless-stopped
    command:
      - "--api.dashboard=true"
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
    ports:
      - "80:80"
      - "8080:8080"     # Traefik Dashboard (chỉ dev)
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro   # Traefik đọc Docker events
    networks:
      - blog-services
      - blog-public

  # ── Frontend ─────────────────────────────────────────────────────────────────
  frontend:
    build: ./frontend
    image: blog/frontend:local
    container_name: blog_frontend
    restart: unless-stopped
    depends_on: [post-service, user-service]
    environment:
      - NEXT_PUBLIC_API_URL=http://localhost/api
      - NODE_ENV=production
    networks:
      - blog-services
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.frontend.rule=Host(`localhost`)"
      - "traefik.http.services.frontend.loadbalancer.server.port=3000"

# ── Volumes ───────────────────────────────────────────────────────────────────
volumes:
  pgdata:
    name: blog_pgdata
  redisdata:
    name: blog_redisdata

# ── Networks ──────────────────────────────────────────────────────────────────
networks:
  blog-public:            # Bên ngoài: browser → gateway
  blog-services:          # Giữa: gateway ↔ services
  blog-data:              # Bên trong: services ↔ DB/cache
    internal: true        # KHÔNG có internet access từ đây!

# ── Secrets ───────────────────────────────────────────────────────────────────
secrets:
  jwt_private_key:
    file: ./secrets/jwt_private_key.pem
```

---

## 3.3 Quản Lý Môi Trường

### Cấu Trúc File

```
blog-engine/
├── .env               ← giá trị local dev của bạn (ĐÃ GITIGNORE)
├── .env.example       ← template với giá trị rỗng (ĐÃ COMMIT)
└── .env.staging       ← giá trị staging (ĐÃ GITIGNORE)
```

### .env.example (luôn commit vào git)

```bash
# Database
DB_USER=
DB_PASSWORD=
DB_NAME=blog

# Redis
REDIS_PASSWORD=

# Auth
JWT_SECRET_KEY=

# App
LOG_LEVEL=info
SERVER_PORT=8080
```

### .env (local dev — đã gitignore)

```bash
DB_USER=blog_user
DB_PASSWORD=dev_password_local
DB_NAME=blog
REDIS_PASSWORD=redis_dev_pass
JWT_SECRET_KEY=dev_secret_min_32_chars_padded!!
LOG_LEVEL=debug
```

### Chuyển Đổi Môi Trường

```bash
docker compose up -d                                  # dùng .env mặc định
docker compose --env-file .env.staging up -d          # dùng staging config
docker compose --env-file .env.staging \
  -f docker-compose.yml \
  -f docker-compose.staging.yml \
  up -d                                              # nhiều file compose
```

---

## 3.4 Healthchecks — Giải Thích Sâu

### Healthcheck Cho Từng Loại Service

```yaml
# PostgreSQL
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U ${DB_USER} -d ${DB_NAME}"]
  interval: 10s
  timeout: 5s
  retries: 5
  start_period: 30s   # cho postgres 30s để khởi động trước khi check

# Redis
healthcheck:
  test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]
  interval: 10s
  timeout: 3s
  retries: 3

# Go HTTP service
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider",
         "http://localhost:8080/health/live"]
  interval: 15s
  timeout: 5s
  retries: 3
  start_period: 10s
```

### Các Điều Kiện depends_on

```yaml
post-service:
  depends_on:
    postgres:
      condition: service_healthy    # chờ healthcheck của postgres pass
    redis:
      condition: service_healthy
    user-service:
      condition: service_started    # chỉ cần user-service đã start (không cần healthy)
```

> 💡 **Tại sao cần healthcheck?** Nếu chỉ dùng `depends_on` mà không có condition, Compose chỉ chờ container *khởi động*, không chờ service *sẵn sàng nhận request*. PostgreSQL cần ~5-10 giây để load xong sau khi container start!

---

## 3.5 Network Segmentation (Phân Đoạn Mạng)

| Service | Networks | Lý do |
|---|---|---|
| `gateway` | `blog-public`, `blog-services` | Cầu nối giữa internet và service tier |
| `post-service` | `blog-services`, `blog-data` | Giao tiếp với gateway và DB |
| `user-service` | `blog-services`, `blog-data` | Tương tự |
| `frontend` | `blog-services` | Chỉ giao tiếp với gateway/service |
| `postgres` | `blog-data` | KHÔNG bao giờ có external access |
| `redis` | `blog-data` | KHÔNG bao giờ có external access |

> 💡 **Nguyên tắc least privilege**: Database không nên bao giờ tiếp xúc với internet. Chỉ các service backend mới nên nói chuyện với DB.

---

## 3.6 Override Files (File Ghi Đè)

```yaml
# docker-compose.override.yml (dev: tự động load)
services:
  post-service:
    volumes:
      - ./services/post-service:/app   # bind-mount để hot-reload
    command: air -c .air.toml          # live-reload với air
    environment:
      - LOG_LEVEL=debug
      - GIN_MODE=debug

  postgres:
    ports:
      - "5432:5432"     # Expose để dùng TablePlus/pgAdmin trong dev
```

```yaml
# docker-compose.prod.yml
services:
  post-service:
    image: registry.company.com/blog/post-service:${VERSION}
    ports: []           # KHÔNG expose port trực tiếp — chỉ qua gateway

  postgres:
    ports: []           # KHÔNG BAO GIỜ expose DB trong production!
```

> 💡 **Giải thích thuật ngữ:**
> - **air**: Tool live-reload cho Go — tự động rebuild và restart khi thay đổi code
> - **hot-reload**: Cập nhật ứng dụng mà không cần restart thủ công
> - **GIN_MODE=debug**: Chế độ debug của Gin framework — log nhiều hơn, hiệu năng thấp hơn

---

## 3.7 Profiles — Service Tùy Chọn

```yaml
services:
  postgres:
    image: postgres:16            # Luôn chạy

  prometheus:
    image: prom/prometheus:latest
    profiles: [monitoring]        # Chỉ chạy với --profile monitoring

  grafana:
    image: grafana/grafana:latest
    profiles: [monitoring]

  pgadmin:
    image: dpage/pgadmin4
    profiles: [debug]             # Chỉ chạy với --profile debug
```

```bash
docker compose up -d                           # Chỉ core services
docker compose --profile monitoring up -d      # Thêm monitoring
docker compose --profile debug up -d           # Thêm debug tools
```

> 💡 **prometheus**: Công cụ thu thập và lưu trữ metrics (số liệu) — sẽ học trong Phase 8
> **grafana**: Dashboard để visualize metrics từ Prometheus
> **pgadmin**: Web UI để quản lý PostgreSQL

---

## 3.8 Tham Khảo Lệnh Compose

```bash
# ─── Vòng đời ─────────────────────────────────────────
docker compose up -d                # Khởi động tất cả service
docker compose up -d --build        # Build lại image rồi mới start
docker compose up -d post-service   # Khởi động một service cụ thể
docker compose down                 # Dừng + xóa container + network
docker compose down -v              # Cũng xóa luôn volumes (mất data!)
docker compose stop
docker compose start
docker compose restart post-service

# ─── Theo dõi ─────────────────────────────────────────
docker compose ps                   # Trạng thái các service
docker compose logs -f              # Log realtime tất cả service
docker compose logs -f post-service # Log realtime một service
docker compose logs --tail=100 -f
docker compose top                  # Process trong từng container
docker compose stats                # CPU/RAM realtime

# ─── Phát triển ───────────────────────────────────────
docker compose exec post-service sh  # Vào container
docker compose run --rm post-service migrate up   # Chạy lệnh một lần
docker compose build post-service    # Build lại image
docker compose pull                  # Pull image mới nhất
docker compose config                # Validate và xem config đã merge

# ─── Scale ────────────────────────────────────────────
docker compose up -d --scale post-service=3    # Chạy 3 instance
docker compose up -d --scale post-service=1    # Reduce về 1
```

---

## 3.9 Blog Engine — Lab Milestone 3

```
[ ] 1. Viết docker-compose.yml hoàn chỉnh
        Services: postgres, redis, post-service, user-service,
                  comment-service, gateway, frontend

[ ] 2. Thêm healthcheck cho: postgres, redis, post-service, user-service

[ ] 3. Cấu hình depends_on: condition: service_healthy
        post-service chờ postgres và redis healthy

[ ] 4. Tạo .env.example và .env

[ ] 5. Thiết lập phân đoạn mạng 3 tầng:
        blog-public, blog-services, blog-data (internal: true)

[ ] 6. Viết docker-compose.override.yml cho dev hot-reload

[ ] 7. Kiểm tra toàn bộ stack:
        docker compose up -d
        curl http://localhost/api/posts        → 200 qua Traefik
        docker compose logs -f                 → xem tất cả log

[ ] 8. Xác minh network isolation:
        docker exec -it blog_postgres curl http://google.com
        → phải FAIL (blog-data là internal: true)

[ ] 9. Scale post-service lên 3 và kiểm tra load balancing:
        docker compose up -d --scale post-service=3
        for i in $(seq 1 10); do curl -s http://localhost/api/posts; done
        → request nên xuất hiện trong log của cả 3 container
```

---

## 3.10 Hướng Dẫn Troubleshooting

| Triệu chứng | Nguyên nhân có thể | Cách sửa |
|---|---|---|
| Service thoát ngay lập tức | DB chưa ready | Thêm `depends_on: condition: service_healthy` |
| `connection refused` | Sai hostname | Dùng tên container, không phải `localhost` |
| Config cũ | File `.env` hoặc override cũ | Chạy `docker compose config` để xem config đã merge |
| Data vẫn còn sau `down` | Named volume không bị xóa bởi `down` | Dùng `docker compose down -v` |
| Port đã bị dùng | Tiến trình khác | `lsof -i :5432` → `kill PID` |
| Service báo unhealthy | Khởi động chậm | Tăng `start_period` và `retries` |
| Không reach được qua tên container | Đang dùng default bridge | Tạo custom network |

---

## 3.11 Tài Nguyên

| Tài nguyên | URL |
|---|---|
| Docker Compose docs | https://docs.docker.com/compose/ |
| Compose file reference | https://docs.docker.com/compose/compose-file/ |
| Compose profiles | https://docs.docker.com/compose/profiles/ |
| air (Go live-reload) | https://github.com/air-verse/air |

---

## 3.12 Tự Kiểm Tra — Phase 3 Hoàn Thành Khi…

```
[ ] Có thể giải thích tại sao cần depends_on: condition: service_healthy
[ ] docker compose up mang tất cả service lên trạng thái healthy
[ ] Blog Engine truy cập được tại http://localhost/api/posts
[ ] Postgres không truy cập được từ gateway container (network isolation)
[ ] .env.example đã commit, .env đã gitignore
[ ] Scale post-service lên 3 và kiểm tra load balancing trong log
```

---

*← [Phase 2 — Docker](./phase_02_docker.md) | [Phase 4 — Microservices →](./phase_04_microservices.md)*
