# Phase 4 — Kiến Trúc Microservices

> **Thời lượng:** 4–6 tuần
> **Trước đó:** [Phase 3 — Docker Compose](./phase_03_docker_compose.md)
> **Tiếp theo:** [Phase 5 — API Gateway](./phase_05_api_gateway.md)
> **Milestone:** 4 Go service độc lập với database riêng, REST API, và giao tiếp event-driven

---

## 📖 Từ Chuyên Ngành Trong Phase Này

| Từ chuyên ngành | Giải thích |
|---|---|
| **Circuit Breaker** | Cầu dao ngắt mạch — tự động ngắt kết nối đến service lỗi để tránh cascading failure |
| **Cascading failure** | Lỗi dây chuyền — một service lỗi kéo theo nhiều service khác lỗi |
| **Event-driven** | Hướng sự kiện — service giao tiếp bằng cách phát/nhận sự kiện, không gọi trực tiếp |
| **Idempotency** | Tính idempotent — gọi nhiều lần cho kết quả giống nhau |
| **Idempotency Key** | Khóa duy nhất kèm theo request để phát hiện và bỏ qua request trùng lặp |
| **Outbox Pattern** | Pattern đảm bảo ghi DB và publish event là nguyên tử (atomic) |
| **Saga Pattern** | Pattern xử lý transaction phân tán qua nhiều service |
| **CQRS** (Command Query Responsibility Segregation) | Tách model đọc và model ghi |
| **Strangler Fig Pattern** | Kỹ thuật dần dần thay thế monolith bằng microservices |
| **Anti-Corruption Layer** | Lớp dịch giữa hai bounded context để tránh phụ thuộc trực tiếp |
| **REST** (Representational State Transfer) | Kiến trúc API dựa trên HTTP |
| **gRPC** | Framework RPC của Google — nhị phân, nhanh hơn REST |
| **OpenAPI** | Chuẩn mô tả REST API (trước đây là Swagger) |
| **Database-per-service** | Mỗi microservice có database riêng — nguyên tắc căn bản |

---

## 4.1 Khi Nào Dùng Microservices

**Không bao giờ bắt đầu với microservices.** Đây là khi nên tách:

| Tín hiệu | Mô tả |
|---|---|
| **Khả năng deploy độc lập** | Các team deploy theo lịch khác nhau |
| **Scaling asymmetry** | Post-service: đọc nhiều; notification: async |
| **Đa dạng công nghệ** | ML model bằng Python, core logic bằng Go |
| **Tự chủ team** | Các team khác nhau sở hữu domain khác nhau |
| **Fault isolation** | Notification sập không nên ảnh hưởng tạo bài viết |

> 💡 **Giải thích thuật ngữ:**
> - **Scaling asymmetry**: Các phần khác nhau cần scale theo cách khác nhau — post-service cần 10 replica, notification-service chỉ cần 1
> - **Fault isolation**: Cô lập lỗi — thiết kế sao cho lỗi ở một service không lan sang service khác
> - **Async**: Bất đồng bộ — xử lý không cần chờ kết quả ngay

### Strangler Fig Pattern (Kỹ Thuật Thay Thế Dần)

```
Phase 1: Monolith xử lý tất cả

Phase 2: Tách service có giá trị cao (auth)
         ┌─────────────────────────┐   ┌──────────────────┐
         │ Monolith (posts+cmts)   │   │ user-service     │
         └─────────────────────────┘   └──────────────────┘

Phase 3: Tách service tiếp theo → cho đến khi monolith rỗng
```

> 💡 **Tên "Strangler Fig"** lấy từ loài cây sung siết — phát triển quanh cây lớn rồi dần thay thế cây đó, giống như microservice dần thay thế monolith.

---

## 4.2 Phân Tách Service — Blog Engine

### Bounded Context → Service Mapping

| Service | Domain | Sở hữu | KHÔNG sở hữu |
|---|---|---|---|
| `user-service` | Identity | users, sessions, JWT | posts, comments |
| `post-service` | Content | posts, tags, categories | user details, comments |
| `comment-service` | Engagement | comments, reactions | posts (chỉ lưu post_id) |
| `notification-service` | Notifications | notification preferences, delivery | business logic |
| `search-service` | Discovery | search index | source of truth data |

### Database-per-Service (Bắt Buộc)

```
post-service    → blog_posts_db    (PostgreSQL)
user-service    → blog_users_db    (PostgreSQL)
comment-service → blog_comments_db (PostgreSQL)
search-service  → Elasticsearch
```

### Anti-Pattern: Distributed Monolith (Monolith Phân Tán)

```
❌ SAI — database chung = distributed monolith (coupling ở tầng dữ liệu)
post-service  ─┐
user-service   ├─── cùng một PostgreSQL DB ← phụ thuộc ẩn!
comment-service┘

✅ ĐÚNG — database riêng cho từng service
post-service    → blog_posts_db (port 5433)
user-service    → blog_users_db (port 5434)
comment-service → blog_comments_db (port 5435)
```

> 💡 **Tại sao database riêng quan trọng?**
> - Nếu dùng DB chung, thay đổi schema của một service có thể phá vỡ service khác
> - Không thể scale từng service độc lập
> - Không có ranh giới rõ ràng về ownership của data

---

## 4.3 Giao Tiếp Đồng Bộ — REST

Dùng REST khi caller **cần phản hồi ngay lập tức** để tiếp tục.

### Service Client Pattern trong Go

```go
// post-service/internal/client/user_client.go

type UserServiceClient struct {
    baseURL    string
    httpClient *http.Client
}

func NewUserServiceClient(baseURL string, timeout time.Duration) *UserServiceClient {
    return &UserServiceClient{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: timeout,
            Transport: &http.Transport{
                MaxIdleConns:    100,
                IdleConnTimeout: 90 * time.Second,
            },
        },
    }
}

func (c *UserServiceClient) GetUser(ctx context.Context, userID string) (*UserResponse, error) {
    req, err := http.NewRequestWithContext(ctx,
        http.MethodGet,
        c.baseURL+"/internal/users/"+userID,
        nil,
    )
    if err != nil {
        return nil, fmt.Errorf("UserServiceClient.GetUser: tạo request: %w", err)
    }

    // Truyền trace ID để theo dõi request xuyên suốt các service
    if traceID := ctx.Value("trace_id"); traceID != nil {
        req.Header.Set("X-Trace-ID", traceID.(string))
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("UserServiceClient.GetUser: HTTP call: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusNotFound {
        return nil, fmt.Errorf("user %s không tìm thấy", userID)
    }
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("status không mong đợi %d", resp.StatusCode)
    }

    var user UserResponse
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        return nil, fmt.Errorf("UserServiceClient.GetUser: decode: %w", err)
    }
    return &user, nil
}
```

---

## 4.4 Giao Tiếp Bất Đồng Bộ — Event-Driven

Dùng events khi publisher **KHÔNG cần phản hồi ngay lập tức**.

### Thiết Kế Event Contract (Hợp Đồng Sự Kiện)

```go
type EventEnvelope struct {
    EventID    string          `json:"event_id"`   // UUID để idempotency
    EventType  string          `json:"event_type"` // "post.published"
    Version    string          `json:"version"`    // "1.0" — versioning quan trọng!
    OccurredAt time.Time       `json:"occurred_at"`
    Source     string          `json:"source"`     // "post-service"
    Payload    json.RawMessage `json:"payload"`    // nội dung sự kiện
}

type PostPublishedPayload struct {
    PostID      string    `json:"post_id"`
    AuthorID    string    `json:"author_id"`
    Title       string    `json:"title"`
    Tags        []string  `json:"tags"`
    PublishedAt time.Time `json:"published_at"`
}

const (
    EventPostPublished  = "post.published"
    EventCommentCreated = "comment.created"
    EventUserRegistered = "user.registered"
)
```

> 💡 **Tại sao cần versioning trong event?** Event được lưu và có thể được replay (phát lại). Nếu thay đổi structure event, consumer cũ sẽ không hiểu. Versioning giúp handle backward compatibility.

### Publisher (post-service phát sự kiện)

```go
func (s *PostServiceImpl) PublishPost(ctx context.Context, postID string) error {
    post, err := s.repo.UpdateStatus(ctx, postID, model.PostStatusPublished)
    if err != nil {
        return fmt.Errorf("PostService.PublishPost: %w", err)
    }

    payload, _ := json.Marshal(PostPublishedPayload{
        PostID:      post.ID,
        AuthorID:    post.AuthorID,
        Title:       post.Title,
        PublishedAt: post.UpdatedAt,
    })

    event := EventEnvelope{
        EventID:    uuid.New().String(),
        EventType:  EventPostPublished,
        Version:    "1.0",
        OccurredAt: time.Now(),
        Source:     "post-service",
        Payload:    payload,
    }

    eventJSON, _ := json.Marshal(event)
    if err := s.publisher.Publish(ctx, EventPostPublished, eventJSON); err != nil {
        // Log warning nhưng không fail — event publishing là best-effort
        global.Logger.Warn("không publish được event",
            zap.String("post_id", postID), zap.Error(err))
    }
    return nil
}
```

### Consumer (notification-service nhận sự kiện)

```go
func (c *EventConsumer) handlePostPublished(ctx context.Context, raw []byte) error {
    var envelope EventEnvelope
    if err := json.Unmarshal(raw, &envelope); err != nil {
        return fmt.Errorf("handlePostPublished: unmarshal: %w", err)
    }

    var payload PostPublishedPayload
    if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
        return fmt.Errorf("handlePostPublished: payload: %w", err)
    }

    // Kiểm tra idempotency — tránh xử lý event trùng lặp
    if err := c.notifSvc.EnsureNotProcessed(ctx, envelope.EventID); err != nil {
        c.logger.Info("event đã được xử lý", zap.String("event_id", envelope.EventID))
        return nil  // bỏ qua, không phải lỗi
    }

    return c.notifSvc.NotifyFollowers(ctx, payload.AuthorID, payload.Title)
}
```

---

## 4.5 Circuit Breaker Pattern (Cầu Dao Ngắt Mạch)

```
Closed (bình thường) → số lỗi vượt ngưỡng → Open (fail fast)
Open → timer hết → Half-Open (thử một request)
Half-Open → thành công → Closed | thất bại → Open lại
```

> 💡 **Tại sao cần Circuit Breaker?** Nếu user-service down, không có circuit breaker thì:
> - Mỗi request đến post-service sẽ chờ timeout 30s gọi user-service
> - post-service sẽ có hàng nghìn goroutine đang chờ
> - post-service cũng sẽ sập (cascading failure)
> - Với circuit breaker: sau 3 lần lỗi, ngắt ngay, trả lỗi trong < 1ms

```go
import "github.com/sony/gobreaker"

var cb = gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "user-service",
    MaxRequests: 1,              // Half-Open: thử tối đa 1 request
    Interval:    60 * time.Second,
    Timeout:     30 * time.Second,
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
        return counts.Requests >= 3 && failureRatio >= 0.6
        // Open khi: ít nhất 3 request và 60% bị lỗi
    },
})

func (c *UserServiceClient) GetUserWithBreaker(ctx context.Context, userID string) (*UserResponse, error) {
    result, err := cb.Execute(func() (interface{}, error) {
        return c.GetUser(ctx, userID)
    })
    if err != nil {
        if err == gobreaker.ErrOpenState {
            return nil, fmt.Errorf("user-service không khả dụng (circuit đang mở): %w", err)
        }
        return nil, err
    }
    return result.(*UserResponse), nil
}
```

---

## 4.6 Idempotency Keys (Khóa Idempotency)

> 💡 **Vấn đề:** Client gửi request tạo bài viết → mạng bị timeout → client không biết bài đã tạo chưa → gửi lại → tạo 2 bài trùng lặp!
> **Giải pháp:** Client gắn `Idempotency-Key` UUID duy nhất → server dùng key này để phát hiện và bỏ qua request trùng.

```go
// POST /api/posts với header Idempotency-Key
func (pc *PostController) CreatePost(c *gin.Context) {
    idempotencyKey := c.GetHeader("Idempotency-Key")
    if idempotencyKey == "" {
        response.ErrorResponseWithHTTP(c, 400, response.ErrCodeBadRequest,
            "Header Idempotency-Key là bắt buộc")
        return
    }

    // Nếu request này đã được xử lý, trả về kết quả cached
    if existing, err := pc.postService.GetByIdempotencyKey(c.Request.Context(), idempotencyKey); err == nil {
        response.SuccessResponse(c, response.ErrCodeSuccess, existing)
        return
    }

    var req CreatePostRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.ErrorResponseWithHTTP(c, 400, response.ErrCodeBadRequest, err.Error())
        return
    }
    post, err := pc.postService.CreatePost(c.Request.Context(), req, idempotencyKey)
    if err != nil {
        response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
        return
    }
    response.SuccessResponse(c, response.ErrCodeSuccess, post)
}
```

---

## 4.7 Reference Các Distributed Patterns (Pattern Hệ Thống Phân Tán)

| Pattern | Vấn đề giải quyết | Triển khai |
|---|---|---|
| **Circuit Breaker** | Cascading failure | `sony/gobreaker` |
| **Retry with Backoff** | Lỗi tạm thời | Exponential: 100ms → 200ms → 400ms |
| **Idempotency Keys** | Request trùng lặp | UUID mỗi request, lưu trong Redis |
| **Saga** | Transaction phân tán | Choreography (events) hoặc Orchestration |
| **CQRS** | Scale đọc/ghi riêng | Model đọc tách khỏi model ghi |
| **Outbox** | Publish event đáng tin cậy | Ghi event vào DB trong cùng transaction |

### Outbox Pattern — Giải Thích

```
Vấn đề: "Ghi DB và publish event" có thể fail ở giữa:
  - Ghi DB thành công nhưng publish event fail → người dùng mất thông báo
  - Đây là lỗi rất khó debug

Giải pháp Outbox Pattern:
  Ghi event vào bảng outbox TRONG CÙNG TRANSACTION với ghi data:
   BEGIN TRANSACTION
     UPDATE posts SET status = 'published'   ← ghi data
     INSERT INTO outbox_events (...) VALUES (...)  ← ghi event
   COMMIT

   Outbox Worker (worker riêng, poll bảng outbox):
     SELECT * FROM outbox_events WHERE processed = false
     → publish lên message broker
     → UPDATE outbox_events SET processed = true
```

> 💡 **Ưu điểm:** Nếu commit transaction thành công thì chắc chắn event sẽ được publish (worker sẽ retry). Nếu commit fail thì cả hai đều bị rollback — không bao giờ mất đồng bộ.

---

## 4.8 OpenAPI 3.0 — Thiết Kế API Contract

```yaml
openapi: "3.0.3"
info:
  title: Post Service API
  version: 1.0.0

paths:
  /posts:
    post:
      operationId: createPost
      summary: Tạo bài viết mới
      security: [{ bearerAuth: [] }]
      parameters:
        - name: Idempotency-Key
          in: header
          required: true
          schema: { type: string, format: uuid }
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "#/components/schemas/CreatePostRequest" }
      responses:
        "201":
          description: Bài viết đã tạo
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Post" }
        "400": { $ref: "#/components/responses/BadRequest" }
        "401": { $ref: "#/components/responses/Unauthorized" }

components:
  schemas:
    Post:
      type: object
      required: [id, title, author_id, status, created_at]
      properties:
        id:         { type: string, format: uuid }
        title:      { type: string, maxLength: 255 }
        content:    { type: string }
        author_id:  { type: string, format: uuid }
        status:     { type: string, enum: [draft, published, archived] }
        created_at: { type: string, format: date-time }
```

---

## 4.9 Database Migration Cho Từng Service

```bash
services/post-service/migrations/
  000001_create_posts_table.up.sql      # migration lên
  000001_create_posts_table.down.sql    # migration xuống (rollback)
  000002_add_outbox_events_table.up.sql
  000002_add_outbox_events_table.down.sql
```

```sql
-- 000001_create_posts_table.up.sql
CREATE TABLE posts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           VARCHAR(255) NOT NULL,
    content         TEXT NOT NULL,
    author_id       UUID NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'published', 'archived')),
    idempotency_key UUID UNIQUE,   -- để phát hiện request trùng lặp
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE outbox_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type  VARCHAR(100) NOT NULL,
    payload     JSONB NOT NULL,
    processed   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_posts_author_id ON posts (author_id);
-- Index partial: chỉ index các row chưa processed → nhanh hơn
CREATE INDEX idx_outbox_unprocessed ON outbox_events (processed) WHERE processed = FALSE;
```

---

## 4.10 Blog Engine — Lab Milestone 4

```
[ ] 1. Scaffold 4 service với database riêng trong Compose
[ ] 2. Viết migrations cho từng service
[ ] 3. Implement full CRUD cho post-service (GET list, GET/:id, POST, PUT, DELETE)
[ ] 4. Implement UserServiceClient trong post-service với circuit breaker
[ ] 5. Implement event publishing (post publish → outbox → Redis Pub/Sub)
[ ] 6. Implement event consumer trong notification-service với idempotency check
[ ] 7. Viết OpenAPI 3.0 spec cho post-service, serve tại /swagger/doc.json
[ ] 8. Test circuit breaker:
        docker compose stop user-service
        → Gửi 3 POST /api/posts → circuit mở
        → Request tiếp theo fail ngay (không chờ timeout)
        docker compose start user-service
        → Circuit đóng lại sau khi recovery
```

---

## 4.11 Tài Nguyên

| Tài nguyên | URL |
|---|---|
| Building Microservices | Sam Newman — O'Reilly |
| Microservices.io patterns | https://microservices.io/patterns/index.html |
| sony/gobreaker | https://github.com/sony/gobreaker |
| OpenAPI Spec | https://spec.openapis.org/oas/v3.0.3 |
| Outbox pattern | https://microservices.io/patterns/data/transactional-outbox.html |

---

## 4.12 Tự Kiểm Tra — Phase 4 Hoàn Thành Khi…

```
[ ] Có thể giải thích tại sao database-per-service là bắt buộc
[ ] post-service gọi user-service qua HTTP (không access DB trực tiếp)
[ ] Circuit breaker kích hoạt khi user-service bị dừng
[ ] Event post publish chạy qua: post-service → Redis → notification-service
[ ] Idempotency key ngăn tạo post trùng lặp khi retry
[ ] OpenAPI spec browse được tại /swagger
[ ] Tất cả service có migration up/down SQL
```

---

*← [Phase 3 — Docker Compose](./phase_03_docker_compose.md) | [Phase 5 — API Gateway →](./phase_05_api_gateway.md)*
