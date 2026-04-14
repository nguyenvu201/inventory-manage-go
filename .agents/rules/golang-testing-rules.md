---
trigger: always_on
description: Golang Testing Rules — Inventory Management System (patterns, coverage, integration)
---

# Golang Testing Rules

Reference for ALL test patterns. Load alongside `golang-developer-rules.md`.

---

## 1. Test File Header (FDA Traceability — MANDATORY)

Every test file MUST start with:

```go
// Package xxx_test implements tests for INV-SPR01-TASK-003
// AC Coverage:
//   AC-01: TestDeviceService_GetDevice_Valid
//   AC-02: TestDeviceService_GetDevice_NotFound
// IEC 62304 Classification: Software Safety Class B
package xxx_test
```

---

## 2. Unit Tests — Service / Domain

**Pattern: table-driven always.**

```go
func TestDeviceService_GetDevice(t *testing.T) {
    setupTestLogger(t) // MANDATORY: initialize global.Logger

    tests := []struct {
        name    string
        id      string
        repoRet *model.Device
        repoErr error
        wantErr bool
        errMsg  string
    }{
        {name: "AC-01: valid device", id: "SCALE-001", repoRet: &model.Device{DeviceID: "SCALE-001"}},
        {name: "AC-02: empty id rejected", id: "", wantErr: true, errMsg: "device_id"},
        {name: "AC-03: repo error propagated", id: "X", repoErr: errors.New("db fail"), wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := impl.NewDeviceService(&mockDeviceRepo{ret: tt.repoRet, err: tt.repoErr})
            _, err := svc.GetDevice(context.Background(), tt.id)
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

### Logger Setup Helper

```go
func setupTestLogger(t *testing.T) {
    t.Helper()
    global.Logger = &logger.LoggerZap{Logger: zap.NewNop()}
    t.Cleanup(func() { global.Logger = nil })
}
```

---

## 3. Unit Tests — Redis Caching (MISS / HIT / STORE)

**Use `miniredis` — NEVER real Redis, NEVER nil `global.Rdb`.**

```go
func setupTestRedis(t *testing.T) *miniredis.Miniredis {
    t.Helper()
    mr, err := miniredis.Run()
    require.NoError(t, err)
    global.Rdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
    global.Logger = &logger.LoggerZap{Logger: zap.NewNop()}
    t.Cleanup(func() { mr.Close(); global.Rdb = nil })
    return mr
}

func TestReportService_GetConsumptionTrend(t *testing.T) {
    tests := []struct {
        name     string
        span     string // "3d" = no cache, "10d" = uses cache
        preSeed  func(mr *miniredis.Miniredis)
        wantHit  bool
    }{
        {name: "cache MISS: span < 7d, no Redis call", span: "3d"},
        {name: "cache MISS: span > 7d, store to Redis", span: "10d"},
        {name: "cache HIT: span > 7d, serve from Redis", span: "10d", preSeed: seedCacheEntry, wantHit: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mr := setupTestRedis(t)
            if tt.preSeed != nil { tt.preSeed(mr) }
            // ... invoke service, assert ...
        })
    }
}
```

---

## 4. Controller Tests (Gin httptest)

```go
func TestDeviceController_GetDevice(t *testing.T) {
    gin.SetMode(gin.TestMode)
    tests := []struct{
        name       string
        deviceID   string
        svcReturn  *model.Device
        svcErr     error
        wantStatus int
        wantCode   int
    }{
        {name: "AC-01: success", deviceID: "SCALE-001", svcReturn: &model.Device{DeviceID: "SCALE-001"}, wantStatus: 200, wantCode: response.ErrCodeSuccess},
        {name: "AC-02: not found", deviceID: "X", svcErr: model.ErrDeviceNotFound, wantStatus: 404, wantCode: response.ErrCodeDeviceNotFound},
        {name: "AC-03: internal error", deviceID: "Y", svcErr: errors.New("db fail"), wantStatus: 200, wantCode: response.ErrCodeInternalServer},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            r := gin.New()
            ctrl := controller.NewDeviceController(&mockDeviceService{device: tt.svcReturn, err: tt.svcErr})
            r.GET("/devices/:id", ctrl.GetDevice)

            w := httptest.NewRecorder()
            req, _ := http.NewRequest(http.MethodGet, "/devices/"+tt.deviceID, nil)
            r.ServeHTTP(w, req)

            assert.Equal(t, tt.wantStatus, w.Code)
            var body map[string]interface{}
            json.Unmarshal(w.Body.Bytes(), &body)
            assert.Equal(t, float64(tt.wantCode), body["code"])
        })
    }
}
```

---

## 5. Integration Tests (Repository — testcontainers)

### 5.1 Build Tag — MANDATORY on LINE 1

```go
//go:build integration

package postgres_test
```

**Applies to:** ALL `*_integration_test.go`, ALL benchmark files that call `setupTestDB`.

### 5.2 Shared Helpers — Accept `testing.TB`

```go
//go:build integration

func setupTestDB(t testing.TB) (*pgxpool.Pool, context.Context) {
    ctx := context.Background()
    pgContainer, err := testpg.RunContainer(ctx,
        testcontainers.WithImage("timescale/timescaledb:latest-pg15"),
        testpg.WithDatabase("test_inventory"),
        testpg.WithUsername("test_user"),
        testpg.WithPassword("test_pass"),
    )
    require.NoError(t, err)
    t.Cleanup(func() { pgContainer.Terminate(ctx) })

    connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")
    runMigrations(t, connStr)

    config, _ := pgxpool.ParseConfig(connStr)
    pool, err := pgxpool.NewWithConfig(ctx, config)
    require.NoError(t, err)
    t.Cleanup(func() { pool.Close() })
    return pool, ctx
}

func runMigrations(t testing.TB, connStr string) { ... }
```

> **Why `testing.TB`?** So both `*testing.T` and `*testing.B` (benchmarks) can call it.

### 5.3 FK Seeding Order (CRITICAL)

```
sku_configs → devices → calibration_configs, inventory_snapshots, inventory_history, threshold_rules
```

**Always seed in this order BEFORE truncating/inserting the table under test:**

```go
// ✅ CORRECT
_, err := pool.Exec(ctx, `INSERT INTO sku_configs (...) VALUES (...) ON CONFLICT DO NOTHING`)
require.NoError(t, err)
_, err = pool.Exec(ctx, `INSERT INTO devices (...) VALUES (...) ON CONFLICT DO NOTHING`)
require.NoError(t, err)
_, err = pool.Exec(ctx, "TRUNCATE inventory_history CASCADE")
require.NoError(t, err)
```

### 5.4 UUID Primary Keys — No Hardcoded String IDs

```go
// ✅ CORRECT — let DB generate UUID via DEFAULT gen_random_uuid()
rule := &model.ThresholdRule{SKUCode: "SKU-A", RuleType: model.RuleTypeLowStock}
err := repo.Save(ctx, rule)
require.NotEmpty(t, rule.ID) // populated by RETURNING id

// ✅ CORRECT — non-existent query uses a valid nil UUID
_, err = repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
require.Error(t, err)

// ❌ WRONG — "RULE-001" is not a UUID
rule := &model.ThresholdRule{ID: "RULE-001"}
```

### 5.5 Integration Test Template

```go
//go:build integration

// Package postgres_test implements tests for INV-SPR03-TASK-004
// AC Coverage:
//   AC-03: TestReportRepository_Integration/GetConsumptionTrend
// IEC 62304 Classification: Software Safety Class B
package postgres_test

func TestXxxRepository_Integration(t *testing.T) {
    if testing.Short() { t.Skip("skipping integration test") }

    pool, ctx := setupTestDB(t)

    // 1. Seed FK dependencies (order matters)
    _, err := pool.Exec(ctx, `INSERT INTO sku_configs (...) VALUES (...) ON CONFLICT DO NOTHING`)
    require.NoError(t, err)

    // 2. Clean the table under test
    _, err = pool.Exec(ctx, "TRUNCATE xxx CASCADE")
    require.NoError(t, err)

    repo := postgres.NewXxxRepository(pool)

    t.Run("AC-01: ...", func(t *testing.T) { ... })
    t.Run("AC-02: ...", func(t *testing.T) { ... })
}
```

---

## 6. Benchmark Tests

```go
//go:build integration  ← REQUIRED (uses setupTestDB)

package postgres_test

func BenchmarkXxxRepository_Query(b *testing.B) {
    pool, ctx := setupTestDB(b) // testing.TB allows this
    // ... seed data ...
    repo := postgres.NewXxxRepository(pool)
    query := model.XxxQuery{...}
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := repo.Query(ctx, query)
        if err != nil { b.Fatalf("query failed: %v", err) }
    }
}
```

---

## 7. Coverage Requirements (FDA Mandatory)

| Package | Minimum |
|---------|---------|
| `internal/service/impl/` | ≥ 85% |
| `internal/domain/telemetry/` | ≥ 90% |
| `internal/domain/inventory/` | ≥ 90% |
| `internal/model/` | ≥ 90% |
| `internal/controller/` | ≥ 80% |
| `internal/repository/postgres/` | ≥ 80% (integration counts) |
| `internal/worker/` | ≥ 80% |

```bash
go test ./... -coverprofile=coverage.out -covermode=atomic -short
go tool cover -func=coverage.out | grep -E "(service/impl|controller|domain|total)"
```

---

## 8. Hard Rules — NEVER Break

| ❌ NEVER | ✅ INSTEAD |
|---------|-----------|
| Rename `_test.go` → `_test.go.txt` | Refactor test to match updated model |
| Use `*testing.T` in shared helper | Use `testing.TB` |
| Miss `//go:build integration` on integration file | Add it on LINE 1 |
| Hardcode string ID for UUID table in test | Let DB `RETURNING id` populate it |
| Use real Redis in unit tests | Use `miniredis.Run()` |
| Skip FK seeding before insert | Seed sku_configs → devices first |
| `time.Sleep` in tests | Use `require.Eventually` or channels |
| `context.Background()` in Gin handler | `c.Request.Context()` |
| Import `repository/` in controller | Import `service/` interface only |