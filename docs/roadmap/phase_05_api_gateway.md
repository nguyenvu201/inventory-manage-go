# Phase 5 — API Gateway, Reverse Proxy & Service Mesh

> **Thời lượng:** 2–3 tuần
> **Trước đó:** [Phase 4 — Microservices](./phase_04_microservices.md)
> **Tiếp theo:** [Phase 6 — Security](./phase_06_security.md)
> **Milestone:** Traefik routing toàn bộ traffic Blog Engine với middleware chain, HTTPS local, và load balancing

---

## 📖 Từ Chuyên Ngành Trong Phase Này

| Từ chuyên ngành | Giải thích |
|---|---|
| **API Gateway** | Cổng vào duy nhất cho hệ thống — xử lý routing, auth, rate limit, CORS |
| **Reverse Proxy** | Proxy ngồi giữa client và server — client không biết đang kết nối server nào |
| **Load Balancer** | Phân phối đều traffic đến nhiều instance của một service |
| **Service Mesh** | Kiến trúc sidecar — Istio/Linkerd xử lý mTLS, tracing, circuit breaker tự động |
| **Traefik** | API Gateway/Reverse proxy hiện đại, tự động discover service qua Docker labels |
| **EntryPoint** | Cổng lắng nghe của Traefik (ví dụ: :80, :443) |
| **Router** | Quy tắc matching request trong Traefik |
| **Middleware** | Lớp xử lý xen giữa request và response (rate-limit, auth, CORS...) |
| **CORS** (Cross-Origin Resource Sharing) | Cơ chế cho phép browser gọi API từ domain khác |
| **Rate limiting** | Giới hạn số lượng request trong khoảng thời gian — chống lạm dụng |
| **mTLS** (mutual TLS) | TLS hai chiều — cả client và server đều phải xác thực |
| **Canary deployment** | Triển khai phiên bản mới cho một phần nhỏ traffic để kiểm tra |
| **Forward Auth** | Pattern Traefik chuyển request đến auth service để validate token |
| **Sticky session** | Đảm bảo request từ cùng một client luôn đến cùng một server |
| **mkcert** | Tool tạo certificate SSL/TLS tự ký cho development local |

---

## 5.1 Tại Sao Cần API Gateway?

```
KHÔNG CÓ GATEWAY:
  Browser → post-service:8081     (CORS ở đây)
  Browser → user-service:8082     (CORS lại ở đây)
  Browser → comment-service:8083  (CORS lần nữa)
  → Rate limit ở 3 nơi, auth ở 3 nơi: trùng lặp khắp nơi!

CÓ GATEWAY:
  Browser ─────▶ Gateway :443
                   ├── /api/posts     → post-service:8080
                   ├── /api/auth      → user-service:8080
                   └── /api/comments  → comment-service:8080
  → CORS, Rate limit, Auth: CHỈ MỘT LẦN tại edge
```

> 💡 **Edge**: Điểm tiếp xúc đầu tiên với traffic từ bên ngoài — đây là nơi lý tưởng để áp dụng cross-cutting concerns.
> **Cross-cutting concerns**: Những vấn đề áp dụng cho toàn bộ hệ thống (auth, logging, rate-limit) — không thuộc về business logic của bất kỳ service cụ thể nào.

### Gateway vs Load Balancer vs Reverse Proxy

| Công cụ | Tầng | Mục đích chính |
|---|---|---|
| **Load Balancer** | L4 (TCP) | Phân phối kết nối đến nhiều server |
| **Reverse Proxy** | L7 (HTTP) | Forward request, SSL termination |
| **API Gateway** | L7 + App | Reverse proxy + auth + rate limit + transform |
| **Service Mesh** | L7 (sidecar) | Traffic giữa các service (east-west) |

> 💡 **Giải thích thuật ngữ:**
> - **L4/L7**: Layer 4 (Transport) và Layer 7 (Application) trong mô hình OSI — L7 hiểu HTTP, L4 chỉ hiểu TCP
> - **SSL termination**: Giải mã HTTPS tại gateway, sau đó giao tiếp bên trong bằng HTTP thuần
> - **East-west traffic**: Traffic giữa các service trong cùng data center (đối lập với north-south: client → server)
> - **Sidecar**: Container phụ chạy cạnh container chính để xử lý networking

---

## 5.2 Traefik — Khái Niệm Cốt Lõi

```
EntryPoints:  Port Traefik lắng nghe (:80, :443)
    │
Routers:      Khớp rule (Host, PathPrefix, Method)
    │
Middlewares:  Transform request (auth, rate-limit, CORS, strip-prefix)
    │
Services:     Đích đến upstream (container của bạn)
```

### Dynamic Configuration qua Docker Labels

Traefik đọc Docker labels và **tự động tạo route** — không cần reload config.

```
Container khởi động với labels
    → Docker daemon thông báo cho Traefik
    → Traefik tạo rule router
    → Request khớp rule được route đến container đó
```

---

## 5.3 Cấu Hình Traefik Cho Blog Engine

```yaml
# docker-compose.yml — gateway service
gateway:
  image: traefik:v3.0
  container_name: blog_gateway
  command:
    # Entrypoints — cổng lắng nghe
    - "--entrypoints.web.address=:80"
    - "--entrypoints.websecure.address=:443"
    # Tự động redirect HTTP → HTTPS
    - "--entrypoints.web.http.redirections.entryPoint.to=websecure"
    - "--entrypoints.web.http.redirections.entryPoint.scheme=https"
    - "--entrypoints.web.http.redirections.entryPoint.permanent=true"

    # Docker provider — discover service từ labels
    - "--providers.docker=true"
    - "--providers.docker.exposedbydefault=false"  # mặc định: không expose service
    - "--providers.docker.network=blog-services"

    # Static file config (TLS, custom middleware)
    - "--providers.file.directory=/etc/traefik/conf.d"
    - "--providers.file.watch=true"   # tự động reload khi file thay đổi

    # Dashboard & Metrics
    - "--api.dashboard=true"
    - "--api.insecure=false"
    - "--metrics.prometheus=true"
    - "--accesslog=true"
    - "--accesslog.format=json"
    - "--log.level=INFO"
  ports:
    - "80:80"
    - "443:443"
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock:ro  # đọc Docker events
    - ./infrastructure/traefik/conf.d:/etc/traefik/conf.d:ro
    - ./infrastructure/traefik/certs:/etc/traefik/certs:ro
  networks:
    - blog-services
    - blog-public
```

### HTTPS Local với mkcert

```bash
brew install mkcert
mkcert -install              # Cài CA vào system trust store
mkcert localhost 127.0.0.1   # Tạo certificate
mv localhost.pem      infrastructure/traefik/certs/
mv localhost-key.pem  infrastructure/traefik/certs/
```

```yaml
# infrastructure/traefik/conf.d/tls.yml
tls:
  certificates:
    - certFile: /etc/traefik/certs/localhost.pem
      keyFile: /etc/traefik/certs/localhost-key.pem
```

> 💡 **Giải thích thuật ngữ:**
> - **CA** (Certificate Authority): Tổ chức cấp chứng chỉ SSL — mkcert tạo CA cục bộ trên máy bạn
> - **Trust store**: Kho chứng chỉ tin cậy của hệ điều hành — sau khi `mkcert -install`, browser sẽ tin certificate từ mkcert CA
> - **Self-signed certificate**: Certificate tự ký — không tin cậy thật sự nhưng dùng được cho dev

---

## 5.4 Cú Pháp Routing Rules

```bash
Host(`blog.example.com`)                                   # theo hostname
PathPrefix(`/api/posts`)                                   # theo path prefix
Host(`localhost`) && PathPrefix(`/api/posts`)              # host + path
Method(`GET`, `POST`)                                      # theo HTTP method
Headers(`X-Custom-Header`, `expected-value`)               # theo header
```

---

## 5.5 Middleware Chain Cho Blog Engine

```yaml
# infrastructure/traefik/conf.d/middlewares.yml
http:
  middlewares:

    # ── Rate Limiting ──────────────────────────────────────────
    rate-limit-api:
      rateLimit:
        average: 100    # 100 request/giây
        burst: 50       # cho phép burst 50 request ngay lập tức
        period: 1s

    rate-limit-auth:
      rateLimit:
        average: 10     # Strict hơn cho endpoint auth
        burst: 5
        period: 1s

    # ── CORS ──────────────────────────────────────────────────
    cors-policy:
      headers:
        accessControlAllowMethods: ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"]
        accessControlAllowHeaders: ["Content-Type", "Authorization", "X-Request-ID"]
        accessControlAllowOriginList:
          - "http://localhost:3000"
          - "https://blog.example.com"
        accessControlMaxAge: 3600        # cache preflight 1 tiếng
        accessControlAllowCredentials: true
        addVaryHeader: true

    # ── Security Headers ──────────────────────────────────────
    security-headers:
      headers:
        frameDeny: true                         # chống clickjacking
        browserXssFilter: true                  # bật XSS protection
        contentTypeNosniff: true                # chống MIME sniffing
        forceSTSHeader: true                    # bật HSTS
        stsSeconds: 31536000                    # HSTS 1 năm
        stsIncludeSubdomains: true
        contentSecurityPolicy: "default-src 'self'"  # CSP

    # ── Compression ────────────────────────────────────────────
    compress:
      compress:
        excludedContentTypes: ["text/event-stream"]
```

> 💡 **Giải thích thuật ngữ:**
> - **CORS**: Cơ chế bảo mật browser — ngăn website A gọi API của website B mà không được phép
> - **Preflight**: Request OPTIONS browser gửi trước request thật để hỏi server có cho phép CORS không
> - **Clickjacking**: Tấn công nhúng site vào iframe ẩn để đánh lừa người dùng click
> - **XSS** (Cross-Site Scripting): Tấn công inject script độc vào trang web
> - **MIME sniffing**: Browser đoán loại file dựa vào nội dung — có thể bị lợi dụng
> - **HSTS** (HTTP Strict Transport Security): Bắt browser luôn dùng HTTPS
> - **CSP** (Content Security Policy): Chính sách quy định browser được load content từ đâu

### Service Labels với Middleware Chain

```yaml
post-service:
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.posts.rule=Host(`localhost`) && PathPrefix(`/api/posts`)"
    - "traefik.http.routers.posts.entrypoints=websecure"   # chỉ HTTPS
    - "traefik.http.routers.posts.tls=true"
    # Middleware chain: rate-limit → cors → security headers → compress
    - "traefik.http.routers.posts.middlewares=rate-limit-api,cors-policy,security-headers,compress"
    - "traefik.http.services.posts.loadbalancer.server.port=8080"
    - "traefik.http.services.posts.loadbalancer.healthcheck.path=/health/live"
    - "traefik.http.services.posts.loadbalancer.healthcheck.interval=10s"

user-service:
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.auth.rule=Host(`localhost`) && PathPrefix(`/api/auth`)"
    - "traefik.http.routers.auth.entrypoints=websecure"
    - "traefik.http.routers.auth.tls=true"
    # Auth endpoint dùng rate-limit strict hơn
    - "traefik.http.routers.auth.middlewares=rate-limit-auth,cors-policy,security-headers"
    - "traefik.http.services.auth.loadbalancer.server.port=8080"
```

---

## 5.6 Forward Authentication (Xác Thực Uỷ Quyền)

Traefik forward mọi request đến auth service để validate JWT:

```yaml
# Middleware: Forward Auth
http:
  middlewares:
    auth-middleware:
      forwardAuth:
        address: "http://user-service:8080/internal/auth/verify"
        authResponseHeaders:
          - "X-User-ID"       # user-service trả về header này khi thành công
          - "X-User-Email"
          - "X-User-Roles"
```

```go
// user-service: GET /internal/auth/verify
func (uc *AuthController) VerifyToken(c *gin.Context) {
    header := c.GetHeader("Authorization")
    if !strings.HasPrefix(header, "Bearer ") {
        c.Status(http.StatusUnauthorized)  // 401 → Traefik sẽ block request
        return
    }
    claims, err := uc.authService.ValidateToken(c.Request.Context(),
        strings.TrimPrefix(header, "Bearer "))
    if err != nil {
        c.Status(http.StatusUnauthorized)
        return
    }
    // Traefik forward các header này đến upstream service
    c.Header("X-User-ID", claims.UserID)
    c.Header("X-User-Email", claims.Email)
    c.Header("X-User-Roles", strings.Join(claims.Roles, ","))
    c.Status(http.StatusOK)  // 200 → Traefik cho phép request tiếp tục
}

// post-service dùng header mà không cần validate JWT lại
func (pc *PostController) CreatePost(c *gin.Context) {
    userID := c.GetHeader("X-User-ID")    // Đã được Traefik validate
    roles  := c.GetHeader("X-User-Roles")
    // ...
}
```

> 💡 **Lợi ích Forward Auth:**
> - Chỉ một nơi validate token (user-service)
> - Các service khác chỉ cần đọc header — không cần import JWT library
> - Dễ thay đổi logic auth mà không sửa các service khác

---

## 5.7 Load Balancing & Canary Deployments

```yaml
# Weighted round-robin (canary: 10% traffic đến v2)
services:
  posts-lb:
    loadbalancer:
      servers:
        - url: "http://post-service-v1:8080"
          weight: 9     # 90% đến v1 ổn định
        - url: "http://post-service-v2:8080"
          weight: 1     # 10% đến v2 canary
```

### Sticky Sessions (cho WebSocket)

```yaml
services:
  websocket-service:
    loadbalancer:
      sticky:
        cookie:
          name: blog_sticky
          secure: true
          httpOnly: true
```

> 💡 **Tại sao WebSocket cần sticky session?** WebSocket là kết nối lâu dài — nếu request đến server 1 để handshake, request tiếp theo phải đến đúng server 1 đó để dùng cùng connection.

---

## 5.8 Khi Nào Chuyển Sang Service Mesh (Istio/Linkerd)

| Nhu cầu | Gateway | Service Mesh |
|---|---|---|
| Client authentication | ✅ | — |
| Rate limiting (bên ngoài) | ✅ | — |
| mTLS giữa các service | ❌ | ✅ (tự động) |
| Traffic splitting (canary) | ✅ | ✅ (tốt hơn) |
| Distributed tracing | Thủ công | ✅ (tự động qua sidecar) |
| Circuit breaking | Thủ công | ✅ (tự động) |

> **Với Blog Engine:** Gateway là đủ. Service mesh thêm overhead vận hành đáng kể — phù hợp để học ở giai đoạn capstone.

---

## 5.9 Blog Engine — Lab Milestone 5

```
[ ] 1. Cấu hình Traefik v3 với Docker provider

[ ] 2. Thiết lập HTTPS local với mkcert
        mkcert localhost 127.0.0.1
        Đặt cert vào infrastructure/traefik/certs/

[ ] 3. Áp dụng middleware chain cho tất cả API route:
        rate-limit → cors-policy → security-headers → compress

[ ] 4. Áp dụng rate-limit-auth strict hơn cho /api/auth

[ ] 5. Cấu hình Forward Auth cho protected routes
        - POST /api/posts yêu cầu Forward Auth → user-service
        - GET /api/posts là public (không cần auth middleware)

[ ] 6. Xác minh routing:
        curl -v https://localhost/api/posts         → 200 post-service
        curl -v https://localhost/api/auth/login    → 200/400 user-service
        curl -v https://localhost/                  → 200 frontend

[ ] 7. Xác minh security headers:
        curl -I https://localhost/api/posts | grep -i "strict-transport"
        → Strict-Transport-Security: max-age=31536000

[ ] 8. Test rate limiting:
        for i in $(seq 1 20); do
          curl -s -o /dev/null -w "%{http_code}\n" https://localhost/api/auth/login
        done
        → 10 request đầu: 200/400, sau 10 request: 429 Too Many Requests

[ ] 9. Scale post-service lên 3 và xác minh qua dashboard:
        docker compose up -d --scale post-service=3
        open http://localhost:8080/dashboard/
        → Phải thấy 3 server healthy
```

---

## 5.10 Tài Nguyên

| Tài nguyên | URL |
|---|---|
| Traefik v3 docs | https://doc.traefik.io/traefik/ |
| Traefik Docker provider | https://doc.traefik.io/traefik/providers/docker/ |
| mkcert (HTTPS local) | https://github.com/FiloSottile/mkcert |
| Forward Auth pattern | https://doc.traefik.io/traefik/middlewares/http/forwardauth/ |
| Traefik Grafana dashboard | https://grafana.com/grafana/dashboards/17346 |

---

## 5.11 Tự Kiểm Tra — Phase 5 Hoàn Thành Khi…

```
[ ] Tất cả route truy cập được qua https://localhost (không cần port)
[ ] HTTP → HTTPS redirect hoạt động (301)
[ ] Rate limiting kích hoạt trên /api/auth sau 10 req/s (429)
[ ] Security headers có trên tất cả response
[ ] Forward Auth từ chối request không có JWT hợp lệ
[ ] Traefik dashboard hiển thị tất cả 4 service healthy
[ ] Có thể giải thích khi nào nên dùng service mesh thay vì gateway
```

---

*← [Phase 4 — Microservices](./phase_04_microservices.md) | [Phase 6 — Security →](./phase_06_security.md)*
