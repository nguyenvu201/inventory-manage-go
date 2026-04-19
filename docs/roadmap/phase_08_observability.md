# Phase 8 — Observability & Production Hardening

> **Thời lượng:** 3–4 tuần
> **Trước đó:** [Phase 7 — Fullstack](./phase_07_fullstack.md)
> **Tiếp theo:** [Capstone →](./capstone.md)
> **Milestone:** Grafana dashboard, Jaeger traces, Loki logs, alerting rules, graceful shutdown — sẵn sàng production

---

## 📖 Từ Chuyên Ngành Trong Phase Này

| Từ chuyên ngành | Giải thích |
|---|---|
| **Observability** | Khả năng quan sát hệ thống từ bên ngoài bằng output của nó |
| **Metrics** | Số liệu định lượng về trạng thái hệ thống (RPS, latency, error rate...) |
| **Traces** | Theo dõi một request qua nhiều service từ đầu đến cuối |
| **Logs** | Bản ghi sự kiện xảy ra trong hệ thống |
| **Prometheus** | Database time-series để lưu metrics, hỗ trợ PromQL để query |
| **Grafana** | Dashboard để visualize metrics từ Prometheus/Loki |
| **Jaeger** | Hệ thống distributed tracing — visualize luồng request qua nhiều service |
| **OpenTelemetry** | Standard mở để thu thập traces, metrics, logs — vendor-neutral |
| **Loki** | Log aggregation của Grafana — như Elasticsearch nhưng nhẹ hơn |
| **Promtail** | Agent thu thập log và gửi về Loki |
| **Span** | Một đơn vị công việc trong trace (ví dụ: một DB query) |
| **Trace ID** | ID duy nhất theo dõi một request qua toàn bộ hệ thống |
| **4 Golden Signals** | 4 chỉ số vàng của SRE Google: Latency, Traffic, Errors, Saturation |
| **SRE** (Site Reliability Engineering) | Kỹ thuật đảm bảo độ tin cậy hệ thống — Google đã phát minh |
| **Graceful shutdown** | Tắt lịch sự: cho phép request đang xử lý hoàn thành |
| **Chaos engineering** | Cố tình gây lỗi vào hệ thống để kiểm tra khả năng chịu đựng |
| **PromQL** | Ngôn ngữ query của Prometheus để truy vấn metrics |
| **LogQL** | Ngôn ngữ query của Loki để truy vấn logs |
| **Runbook** | Tài liệu hướng dẫn xử lý sự cố từng bước |

---

## 8.1 Ba Trụ Cột Của Observability

```
Bạn không thể sửa những gì bạn không thể thấy.

Observability = Logs + Metrics + Traces

LOGS         → "Chuyện gì đã xảy ra?"   Zap → Loki → Grafana
METRICS      → "Bao nhiêu / nhanh cỡ?" Prometheus → Grafana
TRACES       → "Tại sao request chậm?"  OpenTelemetry → Jaeger
```

| Công cụ | Câu hỏi trả lời | Ví dụ |
|---|---|---|
| **Logs** | Chuyện gì xảy ra lúc T? | `ERROR PostRepository.FindByID: context deadline exceeded` |
| **Metrics** | Hệ thống đang hoạt động thế nào? | P99 latency: 230ms; Error rate: 0.1% |
| **Traces** | Tại sao request cụ thể này chậm? | `GET /api/posts/abc → DB query mất 180ms, cache MISS` |

---

## 8.2 Structured Logging qua Zap

### Thiết Lập Logger

```go
// global/logger.go

func InitLogger(level string, pretty bool) (*zap.Logger, error) {
    var config zap.Config
    if pretty {
        config = zap.NewDevelopmentConfig()  // Output dễ đọc cho dev
    } else {
        config = zap.NewProductionConfig()   // JSON output cho production
        // JSON → dễ parse bởi Loki/Elasticsearch
    }

    logLevel, err := zapcore.ParseLevel(level)
    if err != nil {
        return nil, fmt.Errorf("InitLogger: level không hợp lệ %q: %w", level, err)
    }
    config.Level = zap.NewAtomicLevelAt(logLevel)

    return config.Build(
        zap.Fields(
            // Mọi log đều có context về service
            zap.String("service", global.Config.ServiceName),
            zap.String("version", global.Config.Version),
            zap.String("env", global.Config.Environment),
        ),
    )
}
```

### Chuẩn Trường Log — Bắt Buộc

```go
// MỌI request log đều PHẢI có các trường này
global.Logger.Info("hoàn thành request",
    zap.String("trace_id",   c.GetString("trace_id")),  // theo dõi request xuyên service
    zap.String("request_id", c.GetString("request_id")),
    zap.String("method",     c.Request.Method),
    zap.String("path",       c.Request.URL.Path),
    zap.Int("status",        statusCode),
    zap.Duration("latency",  time.Since(start)),
    zap.String("client_ip",  c.ClientIP()),
)

// Domain event phải có context
global.Logger.Info("bài viết được publish",
    zap.String("trace_id",  traceID),
    zap.String("post_id",   post.ID),
    zap.String("author_id", post.AuthorID),
)

// Error phải có đầy đủ context
global.Logger.Error("truy vấn database thất bại",
    zap.String("trace_id",  traceID),
    zap.String("operation", "PostRepository.FindByID"),
    zap.String("post_id",   postID),
    zap.Error(err),  // Toàn bộ error chain
)
```

> 💡 **Tại sao structured logging quan trọng?**
> - Log text thuần: `"ERROR: db failed"` → không thể filter, không thể aggregate
> - Structured JSON: `{"level":"error","trace_id":"abc","operation":"FindByID"}` → Loki/Grafana có thể query theo trace_id, filter theo operation

### Gin Logging Middleware

```go
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        // Tạo hoặc đọc trace ID từ header
        traceID := c.GetHeader("X-Trace-ID")
        if traceID == "" {
            traceID = uuid.New().String()
        }
        c.Set("trace_id", traceID)
        c.Header("X-Trace-ID", traceID)  // Echo lại cho client

        ctx := context.WithValue(c.Request.Context(), "trace_id", traceID)
        c.Request = c.Request.WithContext(ctx)

        c.Next()  // Xử lý request

        status := c.Writer.Status()
        // Log level dựa trên HTTP status
        logFn := logger.Info
        if status >= 500 { logFn = logger.Error }
        if status >= 400 { logFn = logger.Warn }

        logFn("HTTP request",
            zap.String("trace_id", traceID),
            zap.String("method", c.Request.Method),
            zap.String("path", c.Request.URL.Path),
            zap.Int("status", status),
            zap.Duration("latency", time.Since(start)),
            zap.String("client_ip", c.ClientIP()),
        )
    }
}
```

---

## 8.3 Prometheus Metrics

```go
// internal/metrics/metrics.go

var (
    // Đếm tổng số HTTP request (Counter — chỉ tăng, không giảm)
    HttpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "blog", Subsystem: "http",
            Name: "requests_total", Help: "Tổng số HTTP request",
        },
        []string{"service", "method", "path", "status_code"},
    )

    // Đo duration của request (Histogram — phân phối thời gian)
    HttpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: "blog", Subsystem: "http",
            Name:    "request_duration_seconds",
            Help:    "Duration của HTTP request",
            Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
            // Buckets: phân loại theo "bucket thời gian" để tính percentile
        },
        []string{"service", "method", "path"},
    )

    // Đo duration của DB query (Histogram)
    DbQueryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: "blog", Subsystem: "db",
            Name:    "query_duration_seconds",
            Help:    "Duration của DB query",
            Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
        },
        []string{"operation", "table"},
    )
)

// Expose metrics tại /metrics
router.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

> 💡 **Giải thích metric types:**
> - **Counter**: Chỉ tăng — đếm tổng số (requests, errors, events)
> - **Gauge**: Tăng giảm — đo giá trị hiện tại (goroutines, connections, queue size)
> - **Histogram**: Phân phối — đo duration, kích thước để tính P50, P95, P99

### Cấu Hình Prometheus Scrape

```yaml
# infrastructure/prometheus/prometheus.yml
global:
  scrape_interval: 15s    # Prometheus pull metrics mỗi 15 giây

scrape_configs:
  - job_name: 'post-service'
    static_configs:
      - targets: ['post-service:8080']

  - job_name: 'user-service'
    static_configs:
      - targets: ['user-service:8080']

  - job_name: 'traefik'
    static_configs:
      - targets: ['gateway:8080']
```

---

## 8.4 Grafana — 4 Golden Signals (SRE Google)

```
1. LATENCY (Độ trễ)     → Request mất bao lâu?
   metric: http_request_duration_seconds (P50, P95, P99)

2. TRAFFIC (Lưu lượng)  → Bao nhiêu request mỗi giây?
   metric: http_requests_total

3. ERRORS (Lỗi)         → Bao nhiêu % request bị lỗi?
   metric: http_requests_total{status_code=~"5.."}

4. SATURATION (Bão hòa) → Hệ thống đầy chưa?
   metric: container_memory_usage_bytes, go_goroutines
```

### Các Query PromQL Quan Trọng

```promql
# Request rate (số request mỗi giây)
rate(blog_http_requests_total[5m])

# Error rate (% request lỗi)
rate(blog_http_requests_total{status_code=~"5.."}[5m])
/ rate(blog_http_requests_total[5m]) * 100

# P99 latency (99% request hoàn thành trong thời gian này)
histogram_quantile(0.99,
  rate(blog_http_request_duration_seconds_bucket[5m])
)

# DB query P95
histogram_quantile(0.95,
  rate(blog_db_query_duration_seconds_bucket[5m])
)

# Phát hiện goroutine leak
go_goroutines{job="post-service"}
```

> 💡 **P99 latency**: Percentile 99 — 99% request hoàn thành trong thời gian này. Nếu P99 = 500ms, có nghĩa là 1% request chậm hơn 500ms.
> **Goroutine leak**: Goroutine không bao giờ kết thúc — biểu hiện qua go_goroutines tăng liên tục.

---

## 8.5 Distributed Tracing — OpenTelemetry + Jaeger

### Tại Sao Cần Tracing?

```
GET /api/posts/:id — tổng: 203ms. Thời gian đi đâu?

Không có tracing: biết mất 203ms nhưng không biết TẠI SAO.
Có tracing: thấy từng span:
  1. Validate JWT:               5ms
  2. Kiểm tra Redis cache (MISS): 2ms
  3. PostgreSQL query:          180ms  ← ĐÂY RỒI!
  4. Gọi HTTP user-service:     15ms
  5. Build response:             1ms
```

### Thiết Lập OpenTelemetry

```go
func InitTracer(serviceName, jaegerURL string) (func(), error) {
    exporter, err := jaeger.New(
        jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerURL)),
    )
    if err != nil {
        return nil, fmt.Errorf("InitTracer: %w", err)
    }

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
        // Sampling 10%: không trace mọi request (quá nhiều data)
        // Production: 1-10%, Dev: 100%
        trace.WithSampler(trace.TraceIDRatioBased(0.1)),
    )

    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.TraceContext{})
    // propagation: cơ chế truyền trace context qua HTTP header giữa service

    return func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        tp.Shutdown(ctx)  // Flush spans chưa gửi trước khi shutdown
    }, nil
}
```

### Instrument Repository Method (Thêm Span)

```go
func (r *postRepository) FindByID(ctx context.Context, postID string) (*model.Post, error) {
    // Tạo span con — sẽ hiển thị trong Jaeger là một block trong trace
    ctx, span := otel.Tracer("post-repository").Start(ctx, "PostRepository.FindByID")
    defer span.End()

    span.SetAttributes(
        attribute.String("db.system", "postgresql"),
        attribute.String("post.id", postID),
    )

    query, args, _ := sq.Select("*").From("posts").
        Where(sq.Eq{"id": postID}).PlaceholderFormat(sq.Dollar).ToSql()

    row := r.pool.QueryRow(ctx, query, args...)
    var post model.Post
    if err := scanPost(row, &post); err != nil {
        span.RecordError(err)                    // Ghi error vào span
        span.SetStatus(codes.Error, err.Error()) // Đánh dấu span là lỗi
        return nil, fmt.Errorf("PostRepository.FindByID: %w", err)
    }
    return &post, nil
}
```

### Truyền Trace Context Giữa Các Service

```go
// Inject trace context vào HTTP call ra bên ngoài
otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
// → Thêm header: traceparent: 00-abc123-def456-01
// Jaeger dùng header này để kết nối span của các service thành một trace

// Extract trace context từ request đến (ở service nhận)
func ExtractTraceContext() gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := otel.GetTextMapPropagator().Extract(
            c.Request.Context(),
            propagation.HeaderCarrier(c.Request.Header),
        )
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}
```

---

## 8.6 Alerting Rules (Quy Tắc Cảnh Báo)

```yaml
# infrastructure/prometheus/alerts.yml
groups:
  - name: blog-engine
    rules:
      # Error rate cao
      - alert: HighErrorRate
        expr: |
          rate(blog_http_requests_total{status_code=~"5.."}[5m]) /
          rate(blog_http_requests_total[5m]) > 0.05
        for: 2m        # chỉ cảnh báo nếu tình trạng kéo dài 2 phút
        labels: { severity: critical }
        annotations:
          summary: "Error rate cao trên {{ $labels.service }}"

      # P99 latency cao
      - alert: HighLatency
        expr: |
          histogram_quantile(0.99,
            rate(blog_http_request_duration_seconds_bucket[5m])
          ) > 1.0
        for: 5m
        labels: { severity: warning }

      # Service bị down
      - alert: ServiceDown
        expr: up{job=~"post-service|user-service|comment-service"} == 0
        for: 1m
        labels: { severity: critical }
        annotations:
          summary: "Service {{ $labels.job }} bị down"

      # Memory usage cao
      - alert: HighMemoryUsage
        expr: |
          container_memory_usage_bytes{container=~"blog_.*"} /
          container_spec_memory_limit_bytes{container=~"blog_.*"} > 0.85
        for: 5m
        labels: { severity: warning }
```

---

## 8.7 Health Checks (Kiểm Tra Sức Khỏe)

```go
// Liveness: process còn sống không? (Docker dùng để restart nếu dead)
func (h *HealthController) Liveness(c *gin.Context) {
    c.JSON(200, gin.H{
        "status":  "ok",
        "service": global.Config.ServiceName,
        "version": global.Config.Version,
    })
}

// Readiness: có thể nhận traffic không? (dừng traffic khi đang startup/migration)
func (h *HealthController) Readiness(c *gin.Context) {
    checks := map[string]string{"postgres": "ok", "redis": "ok"}
    overall := "ok"

    // Check DB
    if err := h.db.Ping(c.Request.Context()); err != nil {
        checks["postgres"] = err.Error()
        overall = "not ready"   // Không thể phục vụ request nếu DB down
    }

    if h.redis != nil {
        if err := h.redis.Ping(c.Request.Context()).Err(); err != nil {
            checks["redis"] = fmt.Sprintf("degraded: %s", err.Error())
            // KHÔNG đánh dấu not ready — có thể hoạt động không có Redis (chậm hơn)
            // Đây là "graceful degradation"
        }
    }

    statusCode := 200
    if overall != "ok" { statusCode = 503 }

    c.JSON(statusCode, gin.H{"status": overall, "checks": checks})
}
```

> 💡 **Liveness vs Readiness:**
> - **Liveness** = "Tôi còn sống không?" → Docker restart nếu fail
> - **Readiness** = "Tôi có sẵn sàng nhận request không?" → Load balancer tạm ngừng gửi traffic nếu fail (không restart)
> - **Graceful degradation**: Hệ thống hoạt động được (chậm hơn) khi một số component không khả dụng — tốt hơn là crash hoàn toàn

---

## 8.8 Graceful Shutdown (Tắt Lịch Sự)

```go
func main() {
    srv := &http.Server{
        Addr:         fmt.Sprintf(":%d", global.Config.Server.Port),
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Chạy server trong goroutine riêng
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            global.Logger.Fatal("server start thất bại", zap.Error(err))
        }
    }()
    global.Logger.Info("server đã start", zap.Int("port", global.Config.Server.Port))

    // Channel chờ tín hiệu dừng
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    sig := <-quit  // Chặn tại đây cho đến khi nhận signal
    global.Logger.Info("đang shutdown", zap.String("signal", sig.String()))

    // Context với timeout 30 giây — thời gian để hoàn thành request đang xử lý
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {  // Ngừng nhận request mới, đợi request cũ xong
        global.Logger.Error("lỗi khi shutdown", zap.Error(err))
    }

    global.Pdb.Close()  // Đóng DB pool
    global.Logger.Info("server đã dừng")
    global.Logger.Sync()  // Flush log buffer
}
```

> 💡 **SIGTERM**: Tín hiệu "terminate" — Docker gửi SIGTERM khi `docker stop`. Sau 10 giây không dừng thì gửi SIGKILL (không thể bắt được).
> **SIGKILL**: Buộc kill ngay lập tức — request đang xử lý sẽ bị drop!

---

## 8.9 Resource Limits Trong Compose

```yaml
services:
  post-service:
    deploy:
      resources:
        limits:
          memory: 256m      # Bị OOM kill nếu vượt quá
          cpus: "0.5"       # Tối đa 50% của một CPU core
        reservations:
          memory: 128m      # Đảm bảo tối thiểu
          cpus: "0.25"
    read_only: true         # Filesystem read-only
    tmpfs:
      - /tmp:rw,size=50m   # /tmp trong RAM, có thể ghi
    security_opt:
      - no-new-privileges:true
    cap_drop: [ALL]         # Bỏ toàn bộ Linux capabilities
    pids_limit: 100         # Ngăn fork bomb
```

> 💡 **OOM Kill** (Out Of Memory): Linux kernel buộc kill process khi máy hết RAM.
> **Fork bomb**: Chương trình độc cứ spawn process mới mãi cho đến khi máy hết tài nguyên — `pids_limit` ngăn điều này.
> **Linux capabilities**: Các quyền hạn chi tiết của process (bind port < 1024, mount filesystem...) — bỏ hết để giảm attack surface.

---

## 8.10 Blog Engine — Lab Milestone 8

```
[ ] 1. Instrument tất cả Go service với Prometheus metrics
        - HTTP request duration + count
        - DB query duration theo operation
        - Redis cache hit/miss counter

[ ] 2. Thiết lập OpenTelemetry tracing
        - JaegerExporter trong mỗi service
        - Trace context được propagate qua HTTP header
        - Repository method có span

[ ] 3. Khởi động monitoring stack
        docker compose --profile monitoring up -d
        http://localhost:9090   → Prometheus
        http://localhost:3001   → Grafana
        http://localhost:16686  → Jaeger

[ ] 4. Xây dựng Grafana dashboard với 4 Golden Signals
        Import Traefik dashboard (ID: 17346)
        Tạo custom blog-engine dashboard

[ ] 5. Cấu hình alerting rules
        Error rate cao (>5%, 2 phút)     → critical
        P99 latency > 1s (5 phút)        → warning
        Service down (1 phút)            → critical

[ ] 6. Xác minh health endpoints
        curl http://localhost:8081/health/live  → 200 ok
        docker compose stop postgres
        curl http://localhost:8081/health/ready → 503 not ready

[ ] 7. Test graceful shutdown
        docker compose stop post-service
        → Request đang xử lý hoàn thành (≤ 30s)
        → Không có 500 error khi shutdown
        → SIGTERM được log

[ ] 8. Chạy chaos experiment
        Kill post-service trong lúc có traffic
        → Grafana hiển thị error spike
        → Alert kích hoạt
        → Restart service, xác minh recovery
        → Ghi lại kết quả trong runbook

[ ] 9. Viết runbook cho "post-service bị down"
        Kiểm tra gì trước tiên?
        Cách diagnose? (Grafana → Loki → Jaeger)
        Leo thang như thế nào?
        Cách khôi phục?
```

---

## 8.11 Checklist Sẵn Sàng Production

```
Reliability (Độ tin cậy)
[ ] /health/live + /health/ready endpoints
[ ] Graceful shutdown (SIGTERM → 30s drain)
[ ] restart: unless-stopped
[ ] Healthcheck trong Docker Compose
[ ] Circuit breaker trên tất cả external service call

Security (Bảo mật)
[ ] Tất cả secret qua Docker Secrets hoặc Vault
[ ] TLS 1.2+, HTTP → HTTPS redirect
[ ] Security headers: CSP, HSTS, X-Frame-Options
[ ] govulncheck → 0 lỗ hổng

Observability (Quan sát)
[ ] Structured JSON logging
[ ] Không có PII trong log
[ ] Prometheus metrics tại /metrics
[ ] Distributed tracing (OpenTelemetry → Jaeger)
[ ] Grafana dashboard với 4 Golden Signals
[ ] Alerting rules đã cấu hình và test

Performance (Hiệu năng)
[ ] Resource limits đặt (memory + CPU)
[ ] Connection pooling: DB + Redis
[ ] Filesystem read_only: true
[ ] Image size tối ưu

Operations (Vận hành)
[ ] Migrations: file up + down
[ ] Runbook cho: service down, DB down, error rate cao
[ ] Backup strategy cho postgres volumes
[ ] Log retention policy (max-size, max-file)
```

---

## 8.12 Tài Nguyên

| Tài nguyên | URL |
|---|---|
| Observability Engineering (sách) | Honeycomb team — O'Reilly |
| Google SRE Book (miễn phí) | https://sre.google/sre-book |
| OpenTelemetry Go | https://opentelemetry.io/docs/languages/go/ |
| Jaeger tracing | https://www.jaegertracing.io |
| Prometheus Go client | https://github.com/prometheus/client_golang |
| Grafana dashboards | https://grafana.com/grafana/dashboards/ |
| Chaos engineering principles | https://principlesofchaos.org |

---

## 8.13 Tự Kiểm Tra — Phase 8 Hoàn Thành Khi…

```
[ ] Generate load → thấy metrics realtime trong Grafana (RPS, latency, error rate)
[ ] Trace request chậm trong Jaeger: thấy từng span (gateway → service → DB)
[ ] Kill postgres → health/ready trả về 503, alert kích hoạt trong Grafana
[ ] Restart postgres → service tự phục hồi mà không cần can thiệp thủ công
[ ] Trả lời "Tại sao request này chậm?" chỉ dùng traces
[ ] Trả lời "Error rate spike lúc mấy giờ?" chỉ dùng metrics
[ ] Checklist sẵn sàng production 100% checked
```

---

*← [Phase 7 — Fullstack](./phase_07_fullstack.md) | [Capstone →](./capstone.md)*
