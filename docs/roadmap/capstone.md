# Capstone — Thiết Kế Hệ Thống Từ Đầu

> **Thời lượng:** 2–3 tuần
> **Trước đó:** [Phase 8 — Observability](./phase_08_observability.md)
> **Mục tiêu:** Chứng minh tư duy kiến trúc Solution bằng cách thiết kế hệ thống production-grade mà không có scaffolding
> **Deliverable** (sản phẩm nộp): Tài liệu kiến trúc + 5 ADR + partial Go implementation

---

## 📖 Từ Chuyên Ngành Trong Phase Này

| Từ chuyên ngành | Giải thích |
|---|---|
| **ADR** (Architecture Decision Record) | Tài liệu ghi lại quyết định kiến trúc với lý do và đánh đổi |
| **Fan-out** | Khi một event/action gây ra nhiều hành động song song (ví dụ: 1 post → N follower feeds) |
| **Fan-out on Write** | Ghi vào feed của tất cả follower ngay khi post được tạo |
| **Fan-out on Read** | Tính toán feed khi user mở app (không pre-compute) |
| **Thundering herd** | Vấn đề khi hàng nghìn request tấn công cùng lúc (thường sau cache expiry) |
| **Capacity estimation** | Ước tính dung lượng hệ thống cần: storage, throughput, bandwidth |
| **Denormalization** | Lưu trữ dữ liệu dư thừa để tăng tốc truy vấn (đánh đổi với consistency) |
| **Sorted Set** | Cấu trúc dữ liệu Redis — set có score, có thể sort và range query |
| **Sticky session** | Đảm bảo request từ cùng user luôn đến cùng server |
| **C4 Model** | Phương pháp vẽ sơ đồ kiến trúc 4 cấp: Context, Container, Component, Code |
| **SLA** (Service Level Agreement) | Cam kết mức độ dịch vụ (99.9% uptime = ≤ 8.7h downtime/năm) |
| **Read replica** | Bản sao chỉ đọc của database — scale read operations |
| **Sharding** | Chia database thành nhiều shard để scale write operations |

---

## Thử Thách: Thiết Kế Hệ Thống Dạng Twitter

### Yêu Cầu Chức Năng

- User có thể đăng ký, follow/unfollow người khác
- User có thể đăng bài (≤ 280 ký tự) với ảnh tùy chọn
- Mỗi user có "feed" hiển thị bài của những người họ follow
- User có thể like bài và xem số lượt like
- Full-text search cho bài và user

### Yêu Cầu Phi Chức Năng

| Yêu cầu | Mục tiêu |
|---|---|
| Người dùng | 10M đăng ký, 1M DAU (Daily Active Users) |
| Write throughput | 1M posts/ngày (~12 posts/giây avg, 100/giây peak) |
| Read throughput | 100M feed reads/ngày (~1200/giây avg, 10K/giây peak) |
| Availability | 99.9% (≤ 8.7 giờ downtime/năm) |
| Feed latency | P99 < 200ms |
| Post delivery | Follower thấy bài trong vòng 5 giây |
| Like count accuracy | Nhất quán cuối cùng trong ≤ 30 giây là chấp nhận được |

---

## Bước 1: Capacity Estimation (Ước Tính Dung Lượng)

**Luôn bắt đầu từ đây.** Con số dẫn dắt mọi quyết định kiến trúc.

### Storage (Lưu Trữ)

```
Users:
  10M × 1 KB profile = 10 GB
  10M × 500 B auth   = 5 GB

Posts:
  1M posts/ngày × 200 chars × 2 bytes = 400 MB/ngày
  Văn bản 5 năm: ~730 GB

Images (Ảnh):
  1M posts/ngày × 10% có ảnh × trung bình 1 MB = 1 TB/ngày
  → CDN + Object Storage (MinIO / S3)

Timeline cache (Redis):
  1M DAU × 50 posts trong feed × 300 bytes/post = 15 GB trong Redis
  TTL: 24 giờ (xóa feed của user không active)
```

### Throughput (Thông Lượng)

```
Peak write:       100 posts/giây
Peak read:     10,000 feed reads/giây    (100x nhiều hơn write!)
Peak likes:    50,000 likes/giây         (bùng nổ với post viral)
Peak search:      500 queries/giây

→ HỆ THỐNG ĐỌC NHIỀU (ratio 100:1 read:write)
→ Caching là YẾU TỐ SỐNG CÒN
→ Chiến lược fan-out phải tối ưu cho reads
```

> 💡 **Tại sao capacity estimation quan trọng?**
> - Nếu ước tính sai, chọn sai công nghệ: Redis đủ? hay cần Redis Cluster?
> - Xác định bottleneck trước khi xây dựng
> - Cơ sở để ước tính chi phí (cost estimation)
> - **DAU** (Daily Active User): Số người dùng hoạt động mỗi ngày

---

## Bước 2: Architecture Decision Records

### ADR-001: Fan-out on Write vs Fan-out on Read

```markdown
## Trạng thái: Đã Chấp Nhận

## Bối cảnh
Khi user A (có 100K follower) đăng bài, cần build feed cho tất cả follower.
Hai cách tiếp cận:

## Các Lựa Chọn

### Fan-out on Write
Khi A đăng bài → ghi ngay vào feed của tất cả 100K follower
  + O(1) read (feed đã được pre-compute trong Redis)
  − O(N) write mỗi post — "celebrity problem": Lady Gaga đăng = 100M writes

### Fan-out on Read
Khi B mở feed → query bài mới nhất từ tất cả followee trong realtime
  + O(1) write mỗi post
  − O(N) read mỗi lần load feed — chậm nếu follow nhiều người
  − Cache invalidation phức tạp

## Quyết Định: Hybrid (Kết Hợp)
  User thường (≤ 10K follower): Fan-out on WRITE
    → Post đăng → worker ghi post_id vào Redis sorted set của tất cả follower

  Celebrity user (> 10K follower): Fan-out on READ
    → Bài celebrity KHÔNG được pre-distribute
    → Khi load feed: merge feed pre-built + bài celebrity trong realtime

## Hệ Quả
✅ Read nhanh cho 99.9% user (feed đã được compute)
✅ Write amplification có giới hạn (tối đa 10K write mỗi post)
❌ Feed generation phức tạp hơn (cần bước merge cho celebrity post)
```

### ADR-002: Chọn Database Theo Service

| Dữ liệu | Store | Lý do |
|---|---|---|
| Users, Auth | PostgreSQL | ACID, dữ liệu user phải nhất quán |
| Posts (nguồn sự thật) | PostgreSQL | Bền vững, có thể query |
| Timeline (feed cache) | Redis Sorted Set | O(log N) insert/range, 15 GB là vừa phải |
| Like count | Redis Counter | Write rate cao; nhất quán cuối cùng là OK |
| Full-text search | Elasticsearch | Full-text tự nhiên, faceting, ranking |
| Images/files | S3 + CDN | Scale không giới hạn, edge delivery toàn cầu |

### ADR-003: Phòng Chống Thundering Herd

```markdown
## Vấn Đề
Khi cache một post nổi tiếng hết hạn, hàng nghìn request
tấn công PostgreSQL cùng lúc.

## Quyết Định: TTL Jitter (Thêm Nhiễu TTL)
  TTL = base_ttl + random(0, base_ttl * 0.1)
  → Các item khác nhau hết hạn vào thời điểm khác nhau
  → Không có spike hết hạn đồng loạt → không có thundering herd

## Nâng cao: Probabilistic Early Expiration (PER)
  Một phần nhỏ request sẽ chủ động refresh cache TRƯỚC khi nó hết hạn
  → Loại bỏ hoàn toàn thundering herd
```

> 💡 **Jitter**: Thêm ngẫu nhiên nhỏ để tránh mọi thứ xảy ra đồng thời.

### ADR-004: Delivery Thông Báo Realtime

```markdown
## Quyết Định: WebSocket cho user active, queue cho user offline

User active (đang mở app):
  → WebSocket push < 1 giây

User offline:
  → Thông báo được queue trong Redis (TTL 7 ngày)
  → Serve khi user mở app lần sau
  → Push notification qua FCM/APNs cho mobile

Implementation:
  Notification service duy trì map: user_id → WebSocket connection
  Multi-instance: dùng Redis Pub/Sub để fan out giữa các instance
```

### ADR-005: Chiến Lược Rate Limiting

| Endpoint | Giới hạn | Thuật toán |
|---|---|---|
| POST /posts | 10 posts/giờ theo user | Token Bucket |
| GET /feed | 100 req/phút theo user | Sliding Window Log |
| POST /likes | 500 likes/giờ theo user | Token Bucket |
| GET /search | 60 req/phút theo IP | Fixed Window |
| POST /auth/* | 10 req/phút theo IP | Fixed Window (strict) |

> 💡 **Token Bucket**: Thuật toán như xô token — token được thêm ở tốc độ cố định, request tiêu thụ token. Cho phép burst ngắn hạn.
> **Sliding Window**: Đếm request trong cửa sổ thời gian trượt — chính xác hơn Fixed Window.

---

## Bước 3: Kiến Trúc Hệ Thống

```
Mobile App / Browser
       │ HTTPS
       ▼
   CDN (CloudFlare / CloudFront)
     ├── Static assets (JS, CSS, images) ← cache ở edge CDN
     └── API requests ──────────────────────────────────────▶
                                                               │
                                                      API Gateway (Traefik)
                                                      Rate Limit · Auth
                                                             │
           ┌─────────────────┬────────────────┬──────────────┴──────────┐
           │                 │                │                         │
     user-service      post-service     feed-service            search-service
     (auth, profile)   (tạo/xóa bài)   (timeline)              (full-text)
           │                 │                │                         │
           ▼                 ▼                ▼                         ▼
      users_db          posts_db        Redis Sorted Sets          Elasticsearch
     (Postgres)        (Postgres)       (timeline cache)
                           │
                           ▼
               Message Broker (Kafka / Redis Streams)
                    │              │              │
             fanout-worker    like-worker   notification-worker
                    │              │              │
             Redis timelines  Redis counters  WebSocket hub
```

---

## Bước 4: Data Models (Mô Hình Dữ Liệu)

```sql
-- Users DB
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(50)  UNIQUE NOT NULL,
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   BYTEA NOT NULL,
    display_name    VARCHAR(100) NOT NULL,
    follower_count  INT NOT NULL DEFAULT 0,     -- Denormalized để tránh COUNT(*)
    following_count INT NOT NULL DEFAULT 0,
    is_celebrity    BOOLEAN NOT NULL DEFAULT FALSE,  -- > 10K follower
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE follows (
    follower_id UUID NOT NULL,
    followee_id UUID NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, followee_id)
);

-- Posts DB
CREATE TABLE posts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id    UUID NOT NULL,
    content      VARCHAR(280) NOT NULL,
    image_url    TEXT,
    like_count   INT NOT NULL DEFAULT 0,   -- Denormalized, sync định kỳ từ Redis
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_posts_author_created ON posts (author_id, created_at DESC);
```

### Redis Data Structures

```
Timeline của mỗi user:
  Key:   timeline:{user_id}
  Type:  Sorted Set (ZSET)
  Score: Unix timestamp (cho phép range query theo thời gian)
  Value: post_id

  ZADD timeline:user123 1713400000 post_abc   → thêm post vào feed
  ZREVRANGE timeline:user123 0 49              → lấy 50 bài mới nhất
  TTL: 86400 (24 giờ)

Like count:
  Key:   likes:{post_id}
  Type:  String (counter)
  INCR   likes:post_abc                        → atomic increment
  GET    likes:post_abc                        → đọc count

User đã like chưa?
  Key:   post_likes:{post_id}
  Type:  Set
  SADD   post_likes:post_abc user123
  SISMEMBER post_likes:post_abc user123        → O(1) check
```

---

## Bước 5: Feed Service — Partial Implementation (Go)

```go
// feed-service/internal/service/impl/feed_service.go

func (s *FeedServiceImpl) GetFeed(ctx context.Context,
    userID, cursor string, limit int) (*FeedResponse, error) {

    ctx, span := otel.Tracer("feed-service").Start(ctx, "FeedService.GetFeed")
    defer span.End()

    cacheKey := fmt.Sprintf("timeline:%s", userID)
    maxScore := "+inf"
    if cursor != "" { maxScore = cursor }  // cursor là timestamp bài cuối cùng đã thấy

    // 1. Thử Redis (hot cache — nhanh)
    postIDs, err := s.redis.ZRevRangeByScore(ctx, cacheKey, &redis.ZRangeBy{
        Max: maxScore, Min: "-inf", Count: int64(limit),
    }).Result()

    if err != nil && !errors.Is(err, redis.Nil) {
        global.Logger.Warn("Redis feed fetch thất bại, dùng DB fallback",
            zap.String("user_id", userID), zap.Error(err))
    }

    if len(postIDs) == 0 {
        // Cache miss → rebuild từ DB (user mới hoặc cache đã expire)
        return s.buildFeedFromDB(ctx, userID, cursor, limit)
    }

    // 2. Lấy chi tiết posts theo IDs
    posts, err := s.postClient.GetPostsByIDs(ctx, postIDs)
    if err != nil {
        return nil, fmt.Errorf("FeedService.GetFeed: lấy posts: %w", err)
    }

    // 3. Merge bài celebrity (fan-out on read cho celebrity)
    if celebPosts, err := s.getCelebrityPosts(ctx, userID, limit); err == nil {
        posts = mergePosts(posts, celebPosts)
        sortByTimestampDesc(posts)
    }

    var nextCursor string
    if len(posts) == limit {
        nextCursor = posts[len(posts)-1].CreatedAt.Format(time.RFC3339Nano)
    }

    return &FeedResponse{
        Posts:      posts[:min(len(posts), limit)],
        NextCursor: nextCursor,
    }, nil
}

// Fan-out worker: chạy khi có post mới được publish
func (s *FeedServiceImpl) FanOutPost(ctx context.Context, event PostPublishedEvent) error {
    followerCount, err := s.userClient.GetFollowerCount(ctx, event.AuthorID)
    if err != nil {
        return fmt.Errorf("FanOutPost: lấy follower count: %w", err)
    }

    // Celebrity: bỏ qua fan-out (sẽ được serve qua fan-out on read)
    if followerCount > 10_000 {
        return nil
    }

    // User thường: fan-out theo batch qua Redis pipeline
    cursor := ""
    for {
        // Lấy 1000 follower mỗi lần để tránh overload
        followers, nextCursor, err := s.userClient.GetFollowers(ctx, event.AuthorID, cursor, 1000)
        if err != nil {
            return fmt.Errorf("FanOutPost: lấy followers: %w", err)
        }

        // Pipeline: gộp nhiều lệnh Redis thành một batch → nhanh hơn nhiều
        pipe := s.redis.Pipeline()
        score := float64(event.CreatedAt.Unix())
        for _, followerID := range followers {
            key := fmt.Sprintf("timeline:%s", followerID)
            pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: event.PostID})
            pipe.ZRemRangeByRank(ctx, key, 0, -1001) // Giữ tối đa 1000 post trong cache
            pipe.Expire(ctx, key, 24*time.Hour)       // Reset TTL
        }
        if _, err := pipe.Exec(ctx); err != nil {
            // Không fail hoàn toàn — partial fan-out vẫn tốt hơn không có gì
            global.Logger.Error("FanOutPost: Redis pipeline thất bại", zap.Error(err))
        }

        if nextCursor == "" { break }  // Đã xử lý hết follower
        cursor = nextCursor
    }
    return nil
}
```

---

## Bước 6: Phân Tích Scalability

| Component | Capacity Hiện Tại | Bottleneck | Chiến Lược Scale |
|---|---|---|---|
| API Gateway | ~50K req/giây | CPU | Scale ngang (horizontal) |
| Feed Service | ~10K reads/giây | Redis throughput | Redis Cluster |
| Fan-out Worker | 12 posts/giây avg | Redis write rate | Nhiều worker + queue |
| PostgreSQL | 10K read IOPS | Disk/CPU | Read replica |
| Redis | 100K ops/giây | Memory 15GB | Redis Cluster 3+ node |
| Elasticsearch | 500 queries/giây | CPU | Cluster 3 node |

> 💡 **Scale ngang** (Horizontal scaling): Thêm nhiều server — rẻ hơn và phổ biến hơn scale dọc.
> **Scale dọc** (Vertical scaling): Nâng cấp phần cứng server — có giới hạn và đắt hơn.
> **IOPS** (Input/Output Operations Per Second): Số thao tác đọc/ghi mỗi giây của ổ đĩa/database.
> **Read replica**: Bản sao chỉ đọc của DB master — phân tải read ra nhiều server.

---

## Bước 7: Failure Scenarios (Kịch Bản Lỗi)

| Lỗi | Phát hiện | Ảnh hưởng | Biện pháp |
|---|---|---|---|
| Redis down | Health probe fail | Feed load từ DB (chậm) | Fallback sang PostgreSQL |
| Post-service down | 5xx rate spike | Không tạo được bài | Circuit breaker |
| Fan-out worker down | Queue depth tăng | Feed delay > 5s | Alert, restart, replay queue |
| Elasticsearch down | 5xx trên search | Search không khả dụng | Graceful 503, disable search |
| PostgreSQL primary down | Write fail | Read-only mode | Promote replica |

> 💡 **Replay queue**: Phát lại các message trong queue từ điểm nhất định — dùng khi consumer bị down và cần xử lý lại message đã bị bỏ qua.
> **Promote replica**: Nâng replica đọc thành primary để nhận ghi — tự động hoặc thủ công.

---

## Bước 8: Checklist Deliverables

```
Tài liệu 1: Architecture Overview
[ ] C4 Level 1 Context diagram: user, mobile app, browser, 3rd party
[ ] C4 Level 2 Container diagram: tất cả service với tech và giao tiếp
[ ] Data flow diagram: tạo post → fan-out → đọc feed

Tài liệu 2: 5 ADR
[ ] ADR-001: Chiến lược fan-out
[ ] ADR-002: Chọn database theo service
[ ] ADR-003: Cache invalidation strategy
[ ] ADR-004: Delivery thông báo realtime
[ ] ADR-005: Thuật toán rate limiting

Tài liệu 3: Phân Tích Scalability
[ ] Bảng: component / capacity / bottleneck / scale strategy

Tài liệu 4: Failure Scenarios
[ ] 5 kịch bản lỗi với detect / ảnh hưởng / biện pháp / khôi phục

Tài liệu 5: Partial Implementation (chọn MỘT)
[ ] Feed service: GetFeed + FanOut (Go) ← khuyến nghị
[ ] Post service: Create + Publish với Kafka event (Go)
[ ] Like service: Atomic like/unlike + Redis periodic sync về DB (Go)
```

---

## Tự Đánh Giá Solution Architect Readiness

```
Tư Duy Kiến Trúc
[ ] Có thể biện minh mọi lựa chọn công nghệ với tradeoff rõ ràng
[ ] Có thể vẽ component diagram mà không cần scaffolding
[ ] Có thể ước tính dung lượng hệ thống: storage, throughput, cost
[ ] Hiểu YAGNI — kháng lại sự cám dỗ thêm complexity không cần thiết

Distributed Systems (Hệ Thống Phân Tán)
[ ] Áp dụng CAP theorem vào quyết định kiến trúc thực tế
[ ] Thiết kế cho partial failure (graceful degradation)
[ ] Hiểu eventual consistency và khi nào nó chấp nhận được
[ ] Có thể giải thích tradeoff fan-out một cách rõ ràng

Microservices
[ ] Xác định đúng service boundaries từ domain analysis
[ ] Mỗi service có database riêng (không có shared DB)
[ ] Service giao tiếp qua contract (OpenAPI / events)
[ ] Có thể trace một request qua 4 service

Docker & Compose
[ ] Viết Dockerfile production từ đầu (< 20 MB Go image)
[ ] Orchestrate 8 service với healthcheck và startup order đúng
[ ] Network được phân đoạn: data tier không accessible từ gateway

Security (Bảo Mật)
[ ] JWT RS256 đã implement (không phải HS256)
[ ] RBAC được enforce tại controller layer
[ ] Secrets được quản lý mà không hardcode
[ ] OWASP Top 10 mitigation đã áp dụng

Fullstack
[ ] Hiểu SSR vs CSR vs ISR và khi nào chọn cái nào
[ ] Implement auth flow đầu cuối: JWT + httpOnly cookie + refresh
[ ] Realtime UI không cần polling (WebSocket)

Observability (Quan Sát)
[ ] Có thể trả lời "tại sao request X chậm?" chỉ từ traces
[ ] Có thể trả lời "error rate spike lúc mấy giờ?" chỉ từ metrics
[ ] Alerting đã cấu hình và test thực tế
[ ] Graceful shutdown đã implement và verify

─────────────────────────────────────────────────────────────
Bạn sẵn sàng làm Solution Architect nếu ≥ 80% checkbox được tích ✅
```

> 💡 **YAGNI** (You Aren't Gonna Need It): Nguyên tắc không xây dựng tính năng cho đến khi thực sự cần — tránh over-engineering.

---

## Recommended Next Steps (Bước Tiếp Theo)

| Lĩnh vực | Công nghệ | Tài nguyên |
|---|---|---|
| Container Orchestration | Kubernetes | https://kubernetes.io/docs/tutorials/ |
| Service Mesh | Istio | https://istio.io/latest/docs/setup/ |
| GitOps / CI-CD | ArgoCD | https://argo-cd.readthedocs.io |
| Cloud Architecture | AWS SA Associate cert | AWS official prep |
| Event Streaming | Apache Kafka | Confluent tutorials |
| Infrastructure as Code | Terraform | https://developer.hashicorp.com/terraform |
| Database Internals | PostgreSQL deep-dive | "Database Internals" — Alex Petrov |

> 💡 **Infrastructure as Code (IaC)**: Quản lý hạ tầng bằng code — Terraform mô tả server, network, DB như code, có thể version control.
> **GitOps**: Phương pháp quản lý deployment bằng Git — mọi thay đổi infrastructure đều qua pull request.

---

*← [Phase 8 — Observability](./phase_08_observability.md) | [Index →](./index.md)*
