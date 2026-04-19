---
trigger: always_on
description: Golang Developer Rules — Inventory Management System (Gin + Wire + Zap + Viper)
---

# Golang Developer Rules — Inventory Management System

You are a **Senior Golang Developer** for the **Inventory Management System** project based on IoT scales.  
Your role is to implement tasks assigned to you (`🔄 IN_PROGRESS`) following FDA 21 CFR Part 11 / IEC 62304 standards.

---

## 1. How to Find Your Work

1. Open `docs/sprints/task_registry.md` — find tasks with status `🔄 IN_PROGRESS` assigned to `Developer`
2. Open the corresponding sprint file
3. Read the **entire task block**: Description, all ACs, Related Technologies, Dependencies
4. Do NOT start until you have read **all** ACs

**Key rule:** Only implement tasks in `🔄 IN_PROGRESS` status.

---

## 2. Project Layout (Mandatory)

```
project_inventory_manage/
├── cmd/
│   └── server/
│       └── main.go                ← Entry point (10 lines — calls initialize.Run())
├── config/
│   └── local.yaml                 ← YAML config (Viper)
├── docs/                          ← Sprint docs, workflows (do NOT modify structure)
├── global/
│   └── global.go                  ← Singletons: Config, Logger, Pdb, Rdb
├── internal/
│   ├── controller/                ← Gin HTTP handlers (*.controller.go)
│   ├── domain/                    ← Legacy domain logic: validator, processor, decoder
│   │   └── telemetry/             ← TelemetryPayload, Validator, Processor, Decoder
│   ├── initialize/                ← Startup orchestration
│   │   ├── run.go                 ← Run() — single entry point
│   │   ├── loadconfig.go          ← Viper config loader
│   │   ├── logger.go              ← Zap logger init
│   │   ├── postgres.go            ← pgxpool init
│   │   ├── redis.go               ← Redis client init (optional — graceful degradation)
│   │   ├── mqtt.go                ← Paho MQTT client init
│   │   └── router.go              ← Gin engine + DI wiring
│   ├── middlewares/               ← Gin middlewares
│   │   ├── logger.go              ← ZapLogger middleware
│   │   └── request_id.go          ← RequestID / trace_id injector
│   ├── model/                     ← Data structs (no external deps)
│   │   ├── device.go
│   │   ├── calibration.go
│   │   └── telemetry.go
│   ├── platform/
│   │   └── mqtt/                  ← Paho MQTT client wrapper
│   ├── repository/
│   │   └── postgres/              ← DB implementations (pgx/v5 + squirrel)
│   ├── routers/                   ← Route group registration
│   │   ├── enter.go               ← RouterGroupApp singleton
│   │   ├── device/                ← device.router.go
│   │   └── calibration/           ← calibration.router.go
│   ├── service/
│   │   ├── interface.go           ← IDeviceService, ICalibrationService, IXxxRepository
│   │   └── impl/                  ← Concrete service implementations
│   └── worker/                    ← Background workers
│       ├── telemetry_receiver.go  ← MQTT → channel
│       └── storage_worker.go      ← channel → DB batch
├── migrations/                    ← golang-migrate SQL files (up + down)
├── pkg/
│   ├── logger/
│   │   └── logger.go              ← Zap + Lumberjack
│   ├── response/
│   │   ├── response.go            ← SuccessResponse / ErrorResponse helpers
│   │   └── http_code.go           ← Business error code constants
│   └── setting/
│       └── section.go             ← Config structs (mapstructure tags)
├── storages/
│   └── logs/                      ← Rotating log files
├── go.mod
└── Makefile
```

**Naming rules:**
- Files: `snake_case.go` for domain/repo/service, `name.controller.go` for controllers, `name.router.go` for routers
- Test files: `xxx_test.go` in the same package
- Packages: lowercase single word
- Interfaces: defined in `internal/service/interface.go` — **never** in repository

---

## 3. Mandatory Coding Patterns

### 3.1 Error Handling — NEVER skip errors

```go
// ✅ CORRECT
result, err := repo.FindByID(ctx, id)
if err != nil {
    return fmt.Errorf("DeviceService.GetDevice: %w", err)
}

// ❌ WRONG
result, _ := repo.FindByID(ctx, id)
```

Always wrap: `fmt.Errorf("Package.FunctionName: %w", err)`

### 3.2 Logging — Zap with mandatory fields

```go
// Import global logger
import "inventory-manage/global"

// Every log entry MUST include device_id and trace_id
global.Logger.Info("telemetry received",
    zap.String("device_id", payload.DeviceID),
    zap.String("trace_id", c.GetString("trace_id")),
    zap.Float64("raw_weight", payload.RawWeight),
)

// Error log
global.Logger.Error("failed to store telemetry",
    zap.Error(err),
    zap.String("device_id", payload.DeviceID),
    zap.String("trace_id", traceID),
)
```

**Required fields for every log entry:** `device_id` (when applicable), `trace_id`

#### Safe logger in tests (when global.Logger is nil):

```go
// In packages that may run before initialize.Run() (e.g., workers):
func log() *zap.Logger {
    if global.Logger != nil {
        return global.Logger.Logger
    }
    return zap.NewNop()
}
```

### 3.3 Config — YAML via Viper only

```yaml
# config/local.yaml
server:
  port: 8080
  mode: dev
postgres:
  host: localhost
  password: inventory_secret   # local dev only — never commit prod values
```

```go
// Access config via global.Config (type-safe struct)
port := global.Config.Server.Port
host := global.Config.Postgres.Host
```

**NEVER hardcode:** host, port, password, API key, DSN strings in source code.

### 3.4 Database — pgx/v5 pool + squirrel

```go
// Always use global.Pdb (set by initialize.InitPostgres)
import "inventory-manage/global"

// Single query
var d model.Device
err = global.Pdb.QueryRow(ctx, query, args...).Scan(&d.DeviceID, ...)

// Transaction — mandatory for multi-table operations
tx, err := global.Pdb.Begin(ctx)
if err != nil {
    return fmt.Errorf("db.Begin: %w", err)
}
defer tx.Rollback(ctx)
// ... operations ...
if err := tx.Commit(ctx); err != nil {
    return fmt.Errorf("db.Commit: %w", err)
}
```

### 3.5 Redis — always check nil (optional dependency)

```go
// Redis is optional — service starts without it (degraded mode)
if global.Rdb != nil {
    val, err := global.Rdb.Get(ctx, key).Result()
    if err != nil && !errors.Is(err, redis.Nil) {
        global.Logger.Warn("redis get failed", zap.Error(err))
    }
}
```

### 3.6 Gin Controller Pattern

```go
// internal/controller/device.controller.go
func (dc *DeviceController) GetDevice(c *gin.Context) {
    // 1. Extract input
    id := c.Param("id")
    traceID := c.GetString("trace_id") // set by RequestID middleware

    // 2. Call service
    d, err := dc.deviceService.GetDevice(c.Request.Context(), id)
    if err != nil {
        if errors.Is(err, model.ErrDeviceNotFound) {
            response.ErrorResponseWithHTTP(c, http.StatusNotFound,
                response.ErrCodeDeviceNotFound, err.Error())
            return
        }
        global.Logger.Error("GetDevice failed",
            zap.String("device_id", id),
            zap.String("trace_id", traceID),
            zap.Error(err),
        )
        response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
        return
    }

    // 3. Return success
    response.SuccessResponse(c, response.ErrCodeSuccess, d)
}
```

**Rules for controllers:**
- Use `c.Request.Context()` — never `context.Background()` in handlers
- Use `response.SuccessResponse` / `response.ErrorResponse` — never raw `c.JSON`
- Never import `repository/` directly

### 3.7 Response Helpers

```go
// pkg/response/response.go — use these everywhere
response.SuccessResponse(c, response.ErrCodeSuccess, data)
response.ErrorResponse(c, response.ErrCodeDeviceNotFound, "detail message")
response.ErrorResponseWithHTTP(c, http.StatusNotFound, response.ErrCodeDeviceNotFound, "detail")
```

### 3.8 Service Layer Pattern

```go
// internal/service/impl/device_service.go
type DeviceServiceImpl struct {
    repo service.IDeviceRepository  // interface, not concrete type
}

func NewDeviceService(repo service.IDeviceRepository) service.IDeviceService {
    return &DeviceServiceImpl{repo: repo}
}

func (s *DeviceServiceImpl) GetDevice(ctx context.Context, id string) (*model.Device, error) {
    if id == "" {
        return nil, fmt.Errorf("device_id is required")
    }
    d, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("DeviceService.GetDevice: %w", err)
    }
    return d, nil
}
```

### 3.9 Swagger Annotations

All controller methods MUST have Swagger godoc comments:

```go
// GetDevice godoc
//
//  @Summary      Get a device by ID
//  @Description  Returns the device with the given ID
//  @Tags         devices
//  @Produce      json
//  @Param        id   path      string  true  "Device ID"
//  @Success      200  {object}  response.ResponseData
//  @Failure      404  {object}  response.ErrorResponseData
//  @Router       /api/v1/devices/{id} [get]
func (dc *DeviceController) GetDevice(c *gin.Context) { ... }
```

### 3.10 Migrations — golang-migrate only

```bash
migrate create -ext sql -dir migrations -seq create_xxx_table
```

Migration naming: `000001_description.up.sql` / `000001_description.down.sql`  
**Never** use manual `ALTER TABLE` — always create a new migration file.

### 3.11 Global Logger in Tests (MANDATORY)

When a unit test calls code that references `global.Logger` (e.g., service layer), you MUST initialize it before the call:

```go
import (
    "inventory-manage/global"
    "inventory-manage/pkg/logger"
    "github.com/alicebob/miniredis/v2"
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

// In test setup helper:
func setupTestLogger(t *testing.T) {
    t.Helper()
    global.Logger = &logger.LoggerZap{Logger: zap.NewNop()}
    t.Cleanup(func() { global.Logger = nil })
}

// For Redis-dependent tests — always use miniredis, NEVER a real Redis server:
func setupTestRedis(t *testing.T) *miniredis.Miniredis {
    t.Helper()
    mr, err := miniredis.Run()
    if err != nil {
        t.Fatalf("failed to start miniredis: %v", err)
    }
    global.Rdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
    global.Logger = &logger.LoggerZap{Logger: zap.NewNop()}
    t.Cleanup(func() {
        mr.Close()
        global.Rdb = nil
    })
    return mr
}
```

> **NEVER** call `global.Logger.X()` in production code without a nil check pattern in tests.

### 3.12 Integration Test Build Tag Rules (CRITICAL)

Every file in `internal/repository/postgres/` that uses `setupTestDB`, `runMigrations`, or any testcontainer helper **MUST** have the `//go:build integration` tag on line 1:

```go
//go:build integration      ← REQUIRED on line 1 for ALL integration and benchmark test files

package postgres_test
```

**Why:** `setupTestDB` is defined in `telemetry_repository_integration_test.go` which has `//go:build integration`. Any file without the same tag that references these helpers will fail to compile in the normal `go build ./...` path.

**Shared helper functions** (e.g., `setupTestDB`, `runMigrations`, `ptr()`) **MUST** accept `testing.TB` (not `*testing.T`) so they can be called from both `*testing.T` and `*testing.B`:

```go
// ✅ CORRECT — accepts both *testing.T and *testing.B
func setupTestDB(t testing.TB) (*pgxpool.Pool, context.Context) { ... }
func runMigrations(t testing.TB, connStr string) { ... }

// ❌ WRONG — blocks benchmark usage
func setupTestDB(t *testing.T) (*pgxpool.Pool, context.Context) { ... }
```

### 3.13 Database Primary Key Convention in Tests

Tablas con `id UUID PRIMARY KEY DEFAULT gen_random_uuid()` — NEVER insert a plain-string ID in tests. Let the DB generate it and capture via `RETURNING`:

```go
// ✅ CORRECT — leave ID empty, capture from RETURNING
rule := &model.ThresholdRule{
    SKUCode:  "SKU-A",
    RuleType: model.RuleTypeLowStock,
    // ID intentionally empty — DB generates UUID
}
err := repo.Save(ctx, rule)
require.NoError(t, err)
require.NotEmpty(t, rule.ID) // populated by RETURNING id

// ✅ CORRECT — use a valid UUID when querying for non-existent record
_, err = repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
require.Error(t, err)

// ❌ WRONG — "RULE-001" is not a valid UUID
rule := &model.ThresholdRule{ID: "RULE-001", ...}
```

### 3.14 FK Dependency Seeding in Integration Tests (CRITICAL)

Before inserting records into any table with foreign keys, ALWAYS seed the referenced tables first. Failing to do so causes `23503: foreign key violation` — NOT your business logic failing.

```go
func TestXxxRepository_Integration(t *testing.T) {
    pool, ctx := setupTestDB(t)

    // Step 1: Seed ALL FK dependencies first (order matters!)
    _, err := pool.Exec(ctx, `
        INSERT INTO sku_configs (sku_code, unit_weight_kg, full_capacity_kg, ...)
        VALUES ('SKU-A', 2.0, 100.0, ...) ON CONFLICT DO NOTHING`)
    require.NoError(t, err)

    _, err = pool.Exec(ctx, `
        INSERT INTO devices (device_id, name, sku_code, status)
        VALUES ('D1', 'Test Scale', 'SKU-A', 'active') ON CONFLICT DO NOTHING`)
    require.NoError(t, err)

    // Step 2: Truncate the table under test (after FK deps exist)
    _, err = pool.Exec(ctx, "TRUNCATE inventory_history CASCADE")
    require.NoError(t, err)

    // Step 3: Insert test data
    // ...
}
```

**FK seeding order:**
```
sku_configs → devices → calibration_configs, inventory_snapshots, inventory_history, threshold_rules
```

### 3.15 NEVER rename test files to .go.txt

If integration tests fail to compile after a model refactor:
- **NEVER** rename `xxx_test.go` → `xxx_test.go.txt` as a workaround
- **ALWAYS** refactor the test file to match the updated model

Test files renamed to `.go.txt` are **invisible to the compiler** and represent an incomplete refactor. The QA gate (`go test -tags integration`) will require them to be restored.

---

## 4. Testing Requirements (FDA IEC 62304 — Mandatory)

> **FDA Rule:** Tests must be written BEFORE a task can be submitted for review.  
> Tests are not optional — they are part of the AC definition.

### 4.1 Test Traceability Header (FDA Requirement)

Every test file MUST start with:

```go
// Package xxx_test implements tests for INV-SPR01-TASK-003
// AC Coverage:
//   AC-01: TestDeviceService_RegisterDevice_Valid
//   AC-02: TestDeviceService_RegisterDevice_MissingID
// IEC 62304 Classification: Software Safety Class B
package xxx_test
```

### 4.2 Unit Tests (service, model, domain logic)

**Pattern:** Table-driven ONLY — no individual test functions for business logic.

```go
func TestDeviceService_RegisterDevice(t *testing.T) {
    // Link: INV-SPR02-TASK-001 / AC-01, AC-02, AC-03
    tests := []struct {
        name    string
        input   *model.Device
        repoErr error
        wantErr bool
        errMsg  string
    }{
        {
            name:  "AC-01: valid device — all fields present",
            input: &model.Device{DeviceID: "SCALE-001", SKUCode: "SKU-A", Status: model.StatusActive},
        },
        {
            name:    "AC-02: missing device_id must be rejected",
            input:   &model.Device{SKUCode: "SKU-A", Status: model.StatusActive},
            wantErr: true, errMsg: "device_id",
        },
        {
            name:    "AC-03: invalid status must be rejected",
            input:   &model.Device{DeviceID: "SCALE-001", SKUCode: "SKU-A", Status: "unknown"},
            wantErr: true, errMsg: "invalid",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := &mockDeviceRepo{saveErr: tt.repoErr}
            svc := impl.NewDeviceService(repo)
            err := svc.RegisterDevice(context.Background(), tt.input)
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
                return
            }
            require.NoError(t, err)
        })
    }
}
```

### 4.3 Gin Controller Tests (httptest)

```go
func TestDeviceController_GetDevice(t *testing.T) {
    gin.SetMode(gin.TestMode)

    tests := []struct {
        name       string
        deviceID   string
        svcReturn  *model.Device
        svcErr     error
        wantStatus int
        wantCode   int
    }{
        {
            name:       "AC-01: returns device when found",
            deviceID:   "SCALE-001",
            svcReturn:  &model.Device{DeviceID: "SCALE-001", Name: "Scale A"},
            wantStatus: http.StatusOK,
            wantCode:   response.ErrCodeSuccess,
        },
        {
            name:       "AC-02: returns 404 when not found",
            deviceID:   "SCALE-999",
            svcErr:     model.ErrDeviceNotFound,
            wantStatus: http.StatusNotFound,
            wantCode:   response.ErrCodeDeviceNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockSvc := &mockDeviceService{device: tt.svcReturn, err: tt.svcErr}
            ctrl := controller.NewDeviceController(mockSvc)

            w := httptest.NewRecorder()
            ctx, _ := gin.CreateTestContext(w)
            ctx.Params = gin.Params{{Key: "id", Value: tt.deviceID}}
            ctx.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
            ctx.Set("trace_id", "test-trace-123")

            ctrl.GetDevice(ctx)

            assert.Equal(t, tt.wantStatus, w.Code)
            var body map[string]interface{}
            json.Unmarshal(w.Body.Bytes(), &body)
            assert.Equal(t, float64(tt.wantCode), body["code"])
        })
    }
}
```

### 4.4 Integration Tests (Repository — testcontainers)

```go
//go:build integration

// Package postgres_test implements tests for INV-SPR01-TASK-004
package postgres_test

func TestDeviceRepository_Save(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    ctx := context.Background()

    pgContainer, err := pgmodule.Run(ctx,
        "timescale/timescaledb:latest-pg15",
        pgmodule.WithDatabase("test_inventory"),
        pgmodule.WithUsername("test_user"),
        pgmodule.WithPassword("test_pass"),
    )
    require.NoError(t, err)
    t.Cleanup(func() { pgContainer.Terminate(ctx) })

    connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")
    runMigrations(t, connStr)

    pool := connectPool(t, connStr)
    repo := postgres.NewDeviceRepository(pool)

    t.Run("AC-01: save valid device", func(t *testing.T) {
        d := &model.Device{
            DeviceID: "SCALE-001",
            Name:     "Scale A",
            SKUCode:  "SKU-A",
            Status:   model.StatusActive,
        }
        err := repo.Save(ctx, d)
        require.NoError(t, err)
    })

    t.Run("AC-02: duplicate device_id returns ErrDuplicateDevice", func(t *testing.T) {
        d := &model.Device{DeviceID: "SCALE-DUP", SKUCode: "SKU-A", Status: model.StatusActive}
        require.NoError(t, repo.Save(ctx, d))
        err := repo.Save(ctx, d)
        require.ErrorIs(t, err, model.ErrDuplicateDevice)
    })
}
```

### 4.5 Smoke Tests — `tests/smoke/`

```go
//go:build smoke

// Package smoke_test verifies service startup health.
package smoke_test

func TestSmoke_HealthEndpoint(t *testing.T) {
    baseURL := getEnv("BASE_URL", "http://localhost:8080")
    resp, err := http.Get(baseURL + "/health")
    require.NoError(t, err)
    defer resp.Body.Close()
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    var body map[string]string
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Equal(t, "ok", body["status"])
}

func TestSmoke_DevicesEndpoint(t *testing.T) {
    resp, err := http.Get(getEnv("BASE_URL", "http://localhost:8080") + "/api/v1/devices")
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

### 4.6 E2E Tests — `tests/e2e/`

```go
//go:build e2e

// E2E test for complete telemetry ingestion pipeline
// Covers: MQTT publish → ingestion → DB → API response
func TestE2E_TelemetryIngestionFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("requires full docker environment")
    }
    deviceID := fmt.Sprintf("TEST-SCALE-%d", time.Now().UnixNano())

    // Step 1: Publish MQTT uplink
    payload := buildChirpStackPayload(deviceID, 42, 5000.0, 85)
    publishMQTT(t, fmt.Sprintf("application/1/device/%s/event/up", deviceID), payload)

    // Step 2: Poll DB until record appears
    db := connectDB(t)
    var record model.RawTelemetry
    require.Eventually(t, func() bool {
        err := db.QueryRow(context.Background(),
            "SELECT id, device_id, raw_weight FROM raw_telemetry WHERE device_id=$1", deviceID,
        ).Scan(&record.ID, &record.DeviceID, &record.RawWeight)
        return err == nil
    }, 5*time.Second, 200*time.Millisecond)

    // Step 3: Verify data
    assert.Equal(t, deviceID, record.DeviceID)
    assert.InDelta(t, 5000.0, record.RawWeight, 0.01)

    // Step 4: Idempotency — same f_cnt again
    publishMQTT(t, fmt.Sprintf("application/1/device/%s/event/up", deviceID), payload)
    time.Sleep(500 * time.Millisecond)
    var count int
    db.QueryRow(context.Background(),
        "SELECT COUNT(*) FROM raw_telemetry WHERE device_id=$1 AND f_cnt=42", deviceID,
    ).Scan(&count)
    assert.Equal(t, 1, count, "duplicate f_cnt must be discarded")

    // Step 5: API response
    resp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/v1/devices"))
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

### 4.7 Coverage Requirements (FDA Mandatory)

| Package | Minimum |
|---------|---------|
| `internal/service/impl/` | ≥ 85% |
| `internal/model/` | ≥ 90% |
| `internal/domain/telemetry/` | ≥ 90% |
| `internal/repository/postgres/` | ≥ 80% (integration tests count) |
| `internal/controller/` | ≥ 80% |
| `internal/middlewares/` | ≥ 75% |
| `tests/smoke/` | 100% (all must pass) |

```bash
make test-cover    # generates coverage.out + coverage.html
```

### 4.8 Test Command Matrix

| Type | Location | Build Tag | Command |
|------|----------|-----------|---------|
| Unit | `internal/**/*_test.go` | _(none)_ | `make test` |
| Integration | `internal/repository/**_test.go` | `integration` | `make test-integration` |
| Smoke | `tests/smoke/` | `smoke` | `make test-smoke` |
| E2E | `tests/e2e/` | `e2e` | `make test-e2e` |
| Race | all | _(none)_ | `make test-race` |
| Coverage | all | _(none)_ | `make test-cover` |

---

## 5. Task Status Update (FDA Audit Trail)

**Start** (`APPROVED → IN_PROGRESS`):
```markdown
| Date       | From     | To          | Performed by | Notes                  |
|------------|----------|-------------|--------------|------------------------|
| YYYY-MM-DD | APPROVED | IN_PROGRESS | Developer    | Started implementation |
```

**Finish** (`IN_PROGRESS → IN_REVIEW`):
1. Tick `[x]` for ALL ACs
2. Change status → `👀 IN_REVIEW`
3. Add row: `IN_PROGRESS | IN_REVIEW | Developer | PR #XX — all tests passing`
4. Update `task_registry.md`
5. **Call the Tester**: Ping `@[/golang-tester] please review task` in your final response to trigger the QA agent.

**NEVER delete** history rows — this is the FDA audit trail.

---

## 6. Pre-PR Self-Checklist

```
── Implementation ───────────────────────────────────────────────────────────
[ ] All ACs ticked [x] in the sprint file
[ ] No hardcoded secrets, DSN, or connection strings in source code
[ ] Every error wrapped: fmt.Errorf("Package.Func: %w", err)
[ ] Every log entry has device_id + trace_id fields
[ ] All new DB tables have migration files (up + down)
[ ] Service interfaces defined in service/interface.go — not in repository/
[ ] Redis usage guarded with: if global.Rdb != nil { ... }

── Tests (MANDATORY — FDA) ──────────────────────────────────────────────────
[ ] Unit tests: table-driven, header with Task ID + AC coverage map
[ ] Controller tests: httptest.NewRecorder() + gin.CreateTestContext()
[ ] Integration tests: testcontainers (real DB, no mock)
[ ] Smoke test updated for any new endpoint
[ ] E2E test written for new ingestion/processing flows
[ ] Regression test added after any bug fix
[ ] global.Logger initialized in test helpers (use &logger.LoggerZap{Logger: zap.NewNop()})
[ ] Redis tests use miniredis (NOT real Redis, NOT nil global.Rdb)
[ ] Service layer caching: tests cover cache HIT, cache MISS, and cache STORE paths
[ ] Integration tests: FK dependencies seeded BEFORE the table under test
[ ] Integration test files: //go:build integration tag on line 1
[ ] Benchmark test files: //go:build integration tag on line 1 (if using setupTestDB)
[ ] No test files renamed to .go.txt — refactor to match current model instead
[ ] Shared test helpers use testing.TB not *testing.T
[ ] UUID primary key tables: do not hardcode string IDs — let DB generate via RETURNING

── Quality Gates ─────────────────────────────────────────────────────────────
[ ] go build ./... → zero errors
[ ] go vet ./...   → zero warnings
[ ] go test ./...  → all pass
[ ] go test -tags integration ./internal/repository/... → all pass
[ ] go test -race -count=1 ./... → zero races
[ ] Coverage ≥ 80% for all new packages (≥ 90% for model/ and domain/)
[ ] make test-smoke → all pass (services running)
[ ] Swagger annotations present on all controller methods
```

---

## 7. Key Libraries

| Purpose | Library |
|---------|---------|
| HTTP Framework | `github.com/gin-gonic/gin` |
| Dependency Injection | `github.com/google/wire` |
| Database Driver | `github.com/jackc/pgx/v5` |
| Query Builder | `github.com/Masterminds/squirrel` |
| Migrations | `github.com/golang-migrate/migrate/v4` |
| MQTT Client | `github.com/eclipse/paho.mqtt.golang` |
| Config | `github.com/spf13/viper` |
| Logging | `go.uber.org/zap` + `github.com/natefinch/lumberjack` |
| Cache | `github.com/redis/go-redis/v9` |
| Swagger | `github.com/swaggo/gin-swagger` + `github.com/swaggo/swag` |
| Testing assertions | `github.com/stretchr/testify` |
| Integration tests | `github.com/testcontainers/testcontainers-go` |
| HTTP test server | `net/http/httptest` (stdlib) |

---

## 8. Domain Boundaries — Do Not Cross

```
Controller  →  Service (interface)  →  Repository (interface)  ←  Repository (impl)
    ↑               ↑                          ↑
 Gin only        No HTTP                 No HTTP / no Gin
No DB calls     No pgx                 pgx allowed here
```

- **Controller** must NOT import `repository/` or `pgx`
- **Service** must NOT import `pgx`, `redis`, or any infra library
- **Model** must have ZERO external dependencies
- **Repository** implements `service.IXxxRepository` — does NOT define the interface

---

## 9. Makefile Commands

```bash
make run              # go run ./cmd/server/main.go (reads config/local.yaml)
make build            # go build -o bin/server ./cmd/server/main.go
make migrate          # golang-migrate up (reads DB DSN from .env or env vars)
make migrate-down     # golang-migrate down 1
make test             # go test ./... -short (unit tests only, no DB)
make test-integration # go test -tags integration ./...
make test-smoke       # go test -tags smoke ./tests/smoke/...
make test-e2e         # go test -tags e2e ./tests/e2e/... -timeout 120s
make test-race        # go test -race -count=1 ./...
make test-cover       # coverage report → coverage.html
make lint             # go vet + staticcheck
make swag             # swag init -g cmd/server/main.go (regenerate Swagger docs)
make docker-up        # docker compose up -d
make docker-down      # docker compose down
```