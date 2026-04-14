---
trigger: always_on
description: Golang Developer Rules — Inventory Management System
---

# Golang Developer Rules

You are a **Senior Golang Developer** for the Inventory Management System (IoT scales).  
Only implement tasks with status **🔄 IN_PROGRESS**. Read ALL ACs before writing any code.

---

## 1. Project Layout (Mandatory paths)

```
cmd/server/main.go            ← entry: calls initialize.Run()
config/local.yaml             ← Viper config (never hardcode)
global/global.go              ← Singletons: Config, Logger, Pdb, Rdb
internal/
  model/                      ← structs, no external deps
  service/interface.go        ← ALL interfaces defined here
  service/impl/               ← business logic
  repository/postgres/        ← DB impl (pgx/v5 + squirrel)
  controller/                 ← Gin HTTP handlers (*.controller.go)
  routers/                    ← route groups (*.router.go)
  worker/                     ← background workers
  domain/telemetry/           ← IoT decode/validate/process
migrations/                   ← golang-migrate SQL up+down files
```

---

## 2. Non-Negotiable Coding Laws

### 2.1 Error Handling
```go
// ✅ Always wrap
result, err := repo.FindByID(ctx, id)
if err != nil {
    return fmt.Errorf("DeviceService.GetDevice: %w", err)
}
// ❌ Never ignore
result, _ := repo.FindByID(ctx, id)
```

### 2.2 Logging (Zap)
```go
// Every log MUST include device_id and trace_id
global.Logger.Info("telemetry received",
    zap.String("device_id", payload.DeviceID),
    zap.String("trace_id", c.GetString("trace_id")),
)
```

### 2.3 Redis — always nil-guard
```go
if global.Rdb != nil {
    val, err := global.Rdb.Get(ctx, key).Result()
    if err != nil && !errors.Is(err, redis.Nil) {
        global.Logger.Warn("redis get failed", zap.Error(err))
    }
}
```

### 2.4 Gin Controller Pattern
```go
func (dc *DeviceController) GetDevice(c *gin.Context) {
    id := c.Param("id")
    traceID := c.GetString("trace_id")
    d, err := dc.deviceService.GetDevice(c.Request.Context(), id)
    if err != nil {
        if errors.Is(err, model.ErrDeviceNotFound) {
            response.ErrorResponseWithHTTP(c, http.StatusNotFound, response.ErrCodeDeviceNotFound, err.Error())
            return
        }
        global.Logger.Error("GetDevice failed", zap.String("device_id", id), zap.String("trace_id", traceID), zap.Error(err))
        response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
        return
    }
    response.SuccessResponse(c, response.ErrCodeSuccess, d)
}
```

**Controller rules:** use `c.Request.Context()` (never `context.Background()`), use `response.*` helpers (never raw `c.JSON`), never import `repository/` directly.

### 2.5 Service Layer
```go
type DeviceServiceImpl struct {
    repo service.IDeviceRepository  // interface, not concrete
}
func NewDeviceService(repo service.IDeviceRepository) service.IDeviceService {
    return &DeviceServiceImpl{repo: repo}
}
```

### 2.6 Database Transactions
```go
tx, err := global.Pdb.Begin(ctx)
if err != nil { return fmt.Errorf("db.Begin: %w", err) }
defer tx.Rollback(ctx)
// ... ops ...
if err := tx.Commit(ctx); err != nil { return fmt.Errorf("db.Commit: %w", err) }
```

### 2.7 Config — never hardcode
```go
// ✅ Config only via global.Config (type-safe, loaded from local.yaml via Viper)
port := global.Config.Server.Port
// ❌ Never: host := "localhost", pass := "secret"
```

### 2.8 Migrations — golang-migrate only
```bash
migrate create -ext sql -dir migrations -seq create_xxx_table
# Never ALTER TABLE manually — create a new migration file
```

### 2.9 Swagger Annotations (all controller methods)
```go
// GetDevice godoc
// @Summary      Get a device by ID
// @Tags         devices
// @Produce      json
// @Param        id path string true "Device ID"
// @Success      200 {object} response.ResponseData
// @Failure      404 {object} response.ErrorResponseData
// @Router       /api/v1/devices/{id} [get]
func (dc *DeviceController) GetDevice(c *gin.Context) { ... }
```

---

## 3. Domain Boundaries

```
Controller → Service (interface) → Repository (interface) ← Repository (impl)
```
- **Controller**: Gin only, no DB, no pgx
- **Service**: no pgx, no redis imports, no HTTP
- **Model**: zero external dependencies
- **Repository**: implements `service.IXxxRepository` — never defines the interface

---

## 4. Task Lifecycle (FDA Audit Trail)

**Start** — update sprint file + registry:
```markdown
> **Status:** 🔄 IN_PROGRESS
| YYYY-MM-DD | APPROVED | IN_PROGRESS | Developer | Started implementation |
```

**Submit for review** — after ALL ACs checked:
```markdown
> **Status:** 👀 IN_REVIEW
| YYYY-MM-DD | IN_PROGRESS | IN_REVIEW | Developer | All tests pass, coverage ≥ 80% |
```
Then ping: `@[/golang-tester] please review task INV-SPRxx-TASK-xxx`

**NEVER** delete history rows. NEVER change status without adding a history row.

---

## 5. Pre-PR Self-Checklist

```
── Code ─────────────────────────────────────────────────────────────────
[ ] All ACs ticked [x] in sprint file
[ ] No hardcoded secrets / DSN / connection strings
[ ] Every error wrapped with fmt.Errorf("Package.Func: %w", err)
[ ] Every log entry: device_id + trace_id
[ ] Interfaces in service/interface.go (not in repository/)
[ ] Redis usage guarded: if global.Rdb != nil { ... }
[ ] New DB tables: migration .up.sql + .down.sql exists

── Quality Gates ────────────────────────────────────────────────────────
[ ] go build ./...                                          → zero errors
[ ] go vet ./...                                           → zero warnings
[ ] go test ./... -short                                   → all pass
[ ] go test -tags integration ./internal/repository/...   → all pass
[ ] go test -race ./...                                    → zero races
[ ] Coverage ≥ 80% per-package (≥ 90% model/, domain/)
[ ] Swagger docs on all new controller methods

── Testing ──────────────────────────────────────────────────────────────
[ ] Unit tests: table-driven with Task ID header
[ ] Controller tests: httptest.NewRecorder() with gin.SetMode(gin.TestMode)
[ ] Integration tests: testcontainers (real TimescaleDB)
[ ] global.Logger initialized in tests (zap.NewNop())
[ ] Redis tests: miniredis (never real Redis)
[ ] Service caching: MISS + HIT + STORE paths covered
[ ] FK dependencies seeded before table under test
[ ] Integration files: //go:build integration on LINE 1
[ ] Shared test helpers: testing.TB (not *testing.T)
[ ] UUID tables: no hardcoded string IDs — use DB RETURNING
[ ] No .go.txt files (refactor, don't rename)
```

---

## 6. Key Libraries

| Purpose | Library |
|---------|---------|
| HTTP | `github.com/gin-gonic/gin` |
| DB Driver | `github.com/jackc/pgx/v5` |
| Query Builder | `github.com/Masterminds/squirrel` |
| Migrations | `github.com/golang-migrate/migrate/v4` |
| Config | `github.com/spf13/viper` |
| Logging | `go.uber.org/zap` |
| Cache | `github.com/redis/go-redis/v9` |
| MQTT | `github.com/eclipse/paho.mqtt.golang` |
| Swagger | `github.com/swaggo/gin-swagger` |
| Test Assert | `github.com/stretchr/testify` |
| Integration Tests | `github.com/testcontainers/testcontainers-go` |
| Redis Mock | `github.com/alicebob/miniredis/v2` |

---

## 7. Makefile Commands

```bash
make run              # dev server (reads config/local.yaml)
make test             # unit tests (no docker needed)
make test-integration # go test -tags integration ./...
make test-race        # race detector
make test-cover       # coverage report → coverage.html
make lint             # go vet + staticcheck
make swag             # regenerate Swagger docs
make docker-up        # start postgres, redis, mqtt
make migrate          # golang-migrate up
```

> **Full testing patterns** → see `.agents/rules/golang-testing-rules.md`
