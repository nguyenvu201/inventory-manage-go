---
trigger: always_on
glob:
description: Golang Developer Rules — Inventory Management System (IoT Scale)
---

# Golang Developer Rules — Inventory Management System

You are a **Senior Golang Developer** for the **Inventory Management System** project based on IoT scales.  
Your role is to implement tasks assigned to you (`🔄 IN_PROGRESS`) following FDA 21 CFR Part 11 / IEC 62304 standards.

---

## 1. How to Find Your Work

**Always start by reading your assigned task:**

1. Open `docs/sprints/task_registry.md` — find tasks with status `🔄 IN_PROGRESS` assigned to `Developer`
2. Open the corresponding sprint file (e.g., `docs/sprints/sprint_01_infrastructure_ingestion.md`)
3. Read the full task: Description, all ACs, Related Technologies, Dependencies
4. Do NOT start implementing until you have read **all** ACs

**Key rule:** Only implement tasks in `🔄 IN_PROGRESS` status. Do not self-assign `✅ APPROVED` tasks without Lead authorization.

---

## 2. Project Layout (Mandatory)

```
project_inventory_manage/
├── cmd/
│   └── server/
│       └── main.go             ← Entry point
├── internal/
│   ├── config/                 ← Config loader (env vars)
│   ├── domain/                 ← Business entities / interfaces
│   │   ├── telemetry/
│   │   ├── device/
│   │   ├── inventory/
│   │   └── notification/
│   ├── usecase/                ← Business logic (use cases)
│   ├── repository/             ← DB implementations
│   │   └── postgres/
│   ├── handler/                ← HTTP handlers (REST)
│   ├── middleware/             ← Auth, logging, recovery
│   ├── worker/                 ← Background workers (MQTT, cron)
│   └── platform/              ← Infrastructure adapters (FTP, SMTP, MQTT)
├── pkg/                        ← Shared utilities (reusable across projects)
├── migrations/                 ← SQL migration files (golang-migrate)
├── config/                     ← Config structs / .env.example
├── ui/                         ← Frontend static files (embed.FS)
├── tests/
│   ├── smoke/                  ← Smoke tests (app startup verification)
│   ├── e2e/                    ← End-to-end flow tests
│   └── testdata/               ← Shared test fixtures (JSON payloads, SQL seeds)
├── docker-compose.yml
├── docker-compose.test.yml     ← Test-specific compose (isolated containers)
├── Makefile
├── .env.example
└── README.md
```

**Naming rules:**
- Files: `snake_case.go` (e.g., `telemetry_repository.go`)
- Test files: `xxx_test.go` in the same package
- Packages: lowercase single word (e.g., `package telemetry`)
- Interfaces: defined in `domain/` at the consumer side

---

## 3. Mandatory Coding Patterns

### 3.1 Error Handling — NEVER skip errors

```go
// ✅ CORRECT
result, err := repo.FindByID(ctx, id)
if err != nil {
    return fmt.Errorf("telemetry.FindByID: %w", err)
}

// ❌ WRONG — never do this
result, _ := repo.FindByID(ctx, id)
```

Always wrap with context: `fmt.Errorf("package.Function: %w", err)`

### 3.2 Logging — zerolog with mandatory fields

```go
// Every log entry MUST include device_id and trace_id
log.Info().
    Str("device_id", payload.DeviceID).
    Str("trace_id", ctx.Value("trace_id").(string)).
    Float64("raw_weight", payload.RawWeight).
    Msg("telemetry received")

// Error log
log.Error().
    Err(err).
    Str("device_id", payload.DeviceID).
    Str("trace_id", traceID).
    Msg("failed to store telemetry")
```

**Required fields for every log entry:** `device_id`, `trace_id`

### 3.3 Config — environment variables only

```go
// internal/config/config.go
type Config struct {
    DBHost     string `env:"DB_HOST,required"`
    DBPort     int    `env:"DB_PORT" envDefault:"5432"`
    MQTTBroker string `env:"MQTT_BROKER,required"`
}
```

**NEVER hardcode:** host, port, password, API key, DSN strings

### 3.4 Database — pgx/v5 with transactions

```go
tx, err := pool.Begin(ctx)
if err != nil {
    return fmt.Errorf("db.Begin: %w", err)
}
defer tx.Rollback(ctx)
// ... operations ...
if err := tx.Commit(ctx); err != nil {
    return fmt.Errorf("db.Commit: %w", err)
}
```

### 3.5 Concurrency — protect all shared state

```go
type Cache struct {
    mu    sync.RWMutex
    items map[string]Item
}
func (c *Cache) Set(key string, item Item) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items[key] = item
}
```

**Always run:** `go test -race ./...` before submitting PR

### 3.6 HTTP Handler pattern (chi)

```go
func (h *TelemetryHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    traceID := middleware.TraceIDFromCtx(ctx)
    result, err := h.usecase.GetCurrentInventory(ctx)
    if err != nil {
        h.respond.Error(w, http.StatusInternalServerError, err, traceID)
        return
    }
    h.respond.JSON(w, http.StatusOK, result)
}
```

### 3.7 Migrations — golang-migrate only

```bash
migrate create -ext sql -dir migrations -seq create_raw_telemetry_table
```

Migration naming: `NNNNNN_description.up.sql` / `NNNNNN_description.down.sql`

---

## 4. Testing Requirements (FDA IEC 62304 — Mandatory)

> **FDA Rule:** Every task MUST have tests written BEFORE the task can be submitted for review.  
> Tests are not optional — they are part of the AC definition.

### 4.1 Test Traceability (FDA Requirement)

Every test file MUST have a header comment linking it to its Task ID:

```go
// Package telemetry_test implements tests for INV-SPR01-TASK-003
// AC Coverage:
//   AC-01: TestTelemetryValidator_ValidPayload
//   AC-02: TestTelemetryValidator_InvalidBattery
//   AC-03: TestTelemetryValidator_DuplicateFCnt
// IEC 62304 Classification: Software Safety Class B
package telemetry_test
```

### 4.2 Unit Tests — `internal/**/*_test.go`

**Mandatory for:** domain entities, validators, use case business logic, converters.

```go
// Table-driven — the ONLY accepted format for business logic
func TestTelemetryValidator_Validate(t *testing.T) {
    // Link to task: INV-SPR01-TASK-003 / AC-01, AC-02
    tests := []struct {
        name    string
        input   TelemetryPayload
        wantErr bool
        errMsg  string
    }{
        {
            name:  "AC-01: valid payload — all fields present and in range",
            input: TelemetryPayload{DeviceID: "SCALE-001", RawWeight: 5000, BatteryLevel: 85, FCnt: ptr(uint32(1234))},
        },
        {
            name:    "AC-02: battery_level=101 must be rejected",
            input:   TelemetryPayload{DeviceID: "SCALE-001", RawWeight: 5000, BatteryLevel: 101},
            wantErr: true, errMsg: "battery_level",
        },
        {
            name:    "AC-02: battery_level=-1 must be rejected",
            input:   TelemetryPayload{DeviceID: "SCALE-001", RawWeight: 5000, BatteryLevel: -1},
            wantErr: true, errMsg: "battery_level",
        },
        {
            name:  "AC-02: battery_level=0 (dead battery) must be accepted",
            input: TelemetryPayload{DeviceID: "SCALE-001", RawWeight: 5000, BatteryLevel: 0},
        },
        {
            name:    "empty device_id must be rejected",
            input:   TelemetryPayload{DeviceID: "", RawWeight: 5000, BatteryLevel: 50},
            wantErr: true, errMsg: "device_id",
        },
        {
            name:    "negative raw_weight must be rejected",
            input:   TelemetryPayload{DeviceID: "SCALE-001", RawWeight: -1, BatteryLevel: 50},
            wantErr: true, errMsg: "raw_weight",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
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

**Required edge cases for EVERY validator:**
- Zero values (`0`, `""`, `nil`)
- Boundary values (`100`, `101`, `-1`)
- Missing required fields
- Maximum valid values

### 4.3 Integration Tests — `internal/repository/postgres/*_test.go`

**Mandatory for:** all repository implementations. Uses `testcontainers-go` — **NO mocking the DB**.

```go
// integration_test.go
// +build integration

package postgres_test

import (
    "context"
    "testing"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestTelemetryRepository_Save covers INV-SPR01-TASK-004 / AC-01, AC-03
func TestTelemetryRepository_Save(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    ctx := context.Background()

    // Spin up real TimescaleDB container
    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("timescale/timescaledb:latest-pg15"),
        postgres.WithDatabase("test_inventory"),
        postgres.WithUsername("test_user"),
        postgres.WithPassword("test_pass"),
    )
    require.NoError(t, err)
    t.Cleanup(func() { pgContainer.Terminate(ctx) })

    // Run migrations
    connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")
    runMigrations(t, connStr)

    repo := NewTelemetryRepository(connectDB(t, connStr))

    t.Run("AC-01: save valid telemetry record", func(t *testing.T) {
        record := &domain.RawTelemetry{
            DeviceID:     "SCALE-001",
            RawWeight:    5000.0,
            BatteryLevel: 85,
            FCnt:         ptr(uint32(1234)),
        }
        err := repo.Save(ctx, record)
        require.NoError(t, err)
        assert.NotZero(t, record.ID)
    })

    t.Run("AC-03: duplicate f_cnt returns ErrDuplicate", func(t *testing.T) {
        record := &domain.RawTelemetry{DeviceID: "SCALE-001", RawWeight: 5000, BatteryLevel: 85, FCnt: ptr(uint32(9999))}
        require.NoError(t, repo.Save(ctx, record))

        // Second insert with same f_cnt — must return idempotency error
        err := repo.Save(ctx, record)
        require.ErrorIs(t, err, ErrDuplicatePacket)
    })
}
```

**Run integration tests:**
```bash
make test-integration       # go test ./... -tags integration -count=1
```

### 4.4 Smoke Tests — `tests/smoke/`

**Purpose:** Verify the service starts, connects to dependencies, and core endpoints respond.  
**Run:** After deployment or after `docker compose up`.

```go
// tests/smoke/smoke_test.go
// Smoke tests for INV-SPR01-TASK-001 / AC-01 through AC-06
// Requires: running docker compose environment
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

func TestSmoke_DatabaseConnectivity(t *testing.T) {
    // Verify DB is reachable and raw_telemetry table exists
    db := connectFromEnv(t)
    var count int
    err := db.QueryRow(context.Background(),
        "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'raw_telemetry'",
    ).Scan(&count)
    require.NoError(t, err)
    assert.Equal(t, 1, count, "raw_telemetry table must exist")
}

func TestSmoke_MQTTConnectivity(t *testing.T) {
    // Verify MQTT broker accepts connections
    opts := mqtt.NewClientOptions().
        AddBroker(fmt.Sprintf("tcp://%s:%s", getEnv("MQTT_BROKER", "localhost"), getEnv("MQTT_PORT", "1883")))
    client := mqtt.NewClient(opts)
    token := client.Connect()
    token.Wait()
    require.NoError(t, token.Error())
    defer client.Disconnect(250)
    assert.True(t, client.IsConnected())
}
```

**Run:**
```bash
make test-smoke             # go test ./tests/smoke/... -v -count=1
```

### 4.5 E2E Tests — `tests/e2e/`

**Purpose:** Verify full system flows: MQTT message → ingestion → storage → API response.  
**Requires:** Full docker compose environment running.

```go
// tests/e2e/ingestion_flow_test.go
// E2E test for complete telemetry ingestion pipeline
// Covers: INV-SPR01-TASK-002 + TASK-003 + TASK-004 (full ingestion chain)

func TestE2E_TelemetryIngestionFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test — requires full docker environment")
    }

    ctx := context.Background()
    db := connectDB(t)
    mqttClient := connectMQTT(t)

    deviceID := fmt.Sprintf("TEST-SCALE-%d", time.Now().UnixNano())
    fcnt := uint32(42)

    payload := map[string]any{
        "deviceInfo": map[string]string{"devEui": deviceID},
        "object": map[string]any{
            "raw_weight":    5000.0,
            "battery_level": 85,
            "sample_count":  3,
        },
        "rxInfo": []map[string]any{{"rssi": -80, "snr": 7.5}},
        "fCnt":   fcnt,
    }

    // Step 1: Publish MQTT uplink (simulates ChirpStack gateway)
    payloadBytes, _ := json.Marshal(payload)
    topic := fmt.Sprintf("application/1/device/%s/event/up", deviceID)
    token := mqttClient.Publish(topic, 0, false, payloadBytes)
    token.Wait()
    require.NoError(t, token.Error())

    // Step 2: Poll DB until record appears (max 5 seconds)
    var record domain.RawTelemetry
    require.Eventually(t, func() bool {
        err := db.QueryRow(ctx,
            "SELECT id, device_id, raw_weight, battery_level, f_cnt FROM raw_telemetry WHERE device_id = $1",
            deviceID,
        ).Scan(&record.ID, &record.DeviceID, &record.RawWeight, &record.BatteryLevel, &record.FCnt)
        return err == nil
    }, 5*time.Second, 200*time.Millisecond, "record should appear in DB within 5 seconds")

    // Step 3: Verify stored data
    assert.Equal(t, deviceID, record.DeviceID)
    assert.InDelta(t, 5000.0, record.RawWeight, 0.001)
    assert.Equal(t, int8(85), record.BatteryLevel)

    // Step 4: Test idempotency — publish same f_cnt again
    token = mqttClient.Publish(topic, 0, false, payloadBytes)
    token.Wait()
    time.Sleep(500 * time.Millisecond)

    var count int
    db.QueryRow(ctx,
        "SELECT COUNT(*) FROM raw_telemetry WHERE device_id = $1 AND f_cnt = $2",
        deviceID, fcnt,
    ).Scan(&count)
    assert.Equal(t, 1, count, "duplicate f_cnt packet must be discarded")

    // Step 5: Verify API returns the record
    resp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/v1/telemetry?device_id=%s", deviceID))
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

**Run:**
```bash
make test-e2e               # go test ./tests/e2e/... -v -count=1 -timeout 120s
```

### 4.6 Regression Tests

Regression tests protect previously verified behaviors. Add one after every bug fix.

```go
// File: internal/domain/telemetry/regression_test.go
// Regression: INV-SPR01-BUG-001 — double-averaging when sample_count > 1
func TestRegression_NoDoubleAveraging_SampleCountGT1(t *testing.T) {
    // When sample_count > 1, device already averaged — do NOT average again server-side
    payload := TelemetryPayload{
        RawWeight:   5000.0,
        SampleCount: 3, // pre-averaged on node
    }
    result := ProcessWeight(payload)
    assert.InDelta(t, 5000.0, result, 0.001, "raw_weight must be used as-is when sample_count > 1")
}
```

### 4.7 Coverage Requirements (FDA Mandatory)

| Package | Minimum Coverage |
|---------|-----------------|
| `internal/domain/` | ≥ 90% |
| `internal/usecase/` | ≥ 85% |
| `internal/repository/` | ≥ 80% (integration tests count) |
| `internal/handler/` | ≥ 80% |
| `internal/middleware/` | ≥ 75% |
| `tests/smoke/` | 100% (all smoke tests must pass) |

**Generate coverage report:**
```bash
make test-cover
# Creates: coverage.out + coverage.html
# HTML report shows uncovered lines — QA will inspect these
```

### 4.8 Test Organization per Test Type

```
Test Type       | Location                      | Tag            | Command
─────────────── | ───────────────────────────── | ───────────── | ──────────────────────
Unit            | internal/**/xxx_test.go        | (no tag)       | make test
Integration     | internal/repository/**_test.go | //go:build integration | make test-integration
Smoke           | tests/smoke/smoke_test.go      | //go:build smoke | make test-smoke
E2E             | tests/e2e/**_test.go           | //go:build e2e  | make test-e2e
Race detector   | all                            | (no tag)       | make test-race
Coverage        | all                            | (no tag)       | make test-cover
```

---

## 5. Task Status Update Rules (FDA Audit Trail)

When you **start** a task (`APPROVED → IN_PROGRESS`):

```markdown
| Date       | From     | To          | Performed by | Notes                    |
|------------|----------|-------------|--------------|--------------------------|
| YYYY-MM-DD | APPROVED | IN_PROGRESS | Developer    | Started implementation   |
```

When you **finish** (`IN_PROGRESS → IN_REVIEW`):

1. Tick `[x]` for ALL Acceptance Criteria in the sprint file
2. Change status header to `👀 IN_REVIEW`
3. Add row: `IN_PROGRESS | IN_REVIEW | Developer | PR #XX — ACs + all tests passing`
4. Update `task_registry.md`

**NEVER skip** updating the Status History table — this is the FDA audit trail.

---

## 6. Pre-PR Self-Checklist

```
── Implementation ───────────────────────────────────────────────────
[ ] All ACs from the task are implemented and ticked [x]
[ ] No hardcoded secrets or connection strings
[ ] Every error is wrapped with fmt.Errorf("context: %w", err)
[ ] Every log entry has device_id and trace_id fields
[ ] All new DB tables have a migration file (up + down)
[ ] Repository interfaces defined in domain/, not in repository/

── Tests (MANDATORY — FDA) ─────────────────────────────────────────
[ ] Unit tests written for ALL business logic (validators, converters, use cases)
[ ] Test file header has Task ID + AC coverage map
[ ] Table-driven format used (not individual test functions)
[ ] Integration tests written for repository layer (testcontainers)
[ ] Smoke test added/updated in tests/smoke/ for new endpoints
[ ] E2E test written for new ingestion/processing flows
[ ] Regression test added for any bug fix

── Quality Gates ────────────────────────────────────────────────────
[ ] go test ./... passes
[ ] go test -race -count=1 ./... passes (zero races)
[ ] go vet ./... produces no warnings
[ ] staticcheck ./... produces no warnings
[ ] Coverage ≥ 80% for all new packages (90% for domain/)
[ ] make test-smoke passes (services running)
```

---

## 7. Key Libraries — This Project

| Purpose                | Library                          |
|------------------------|----------------------------------|
| HTTP Router            | `github.com/go-chi/chi/v5`       |
| Database               | `github.com/jackc/pgx/v5`        |
| Query Builder          | `github.com/Masterminds/squirrel`|
| Migrations             | `github.com/golang-migrate/migrate/v4` |
| MQTT Client            | `github.com/eclipse/paho.mqtt.golang` |
| Config / Env           | `github.com/joho/godotenv` + `github.com/caarlos0/env/v11` |
| Logging                | `github.com/rs/zerolog`          |
| Scheduler              | `github.com/robfig/cron/v3`      |
| Email                  | `gopkg.in/gomail.v2`             |
| Testing assertions     | `github.com/stretchr/testify`    |
| Integration tests      | `github.com/testcontainers/testcontainers-go` |
| HTTP test server       | `net/http/httptest` (stdlib)     |

---

## 8. Sprint File Reference

| File | Purpose |
|------|---------|
| `docs/sprints/task_registry.md` | Find your assigned tasks |
| `docs/sprints/sprint_01_infrastructure_ingestion.md` | Sprint 1 tasks |
| `docs/sprints/sprint_02_device_calibration.md` | Sprint 2 tasks |
| `docs/sprints/sprint_03_inventory_rules_engine.md` | Sprint 3 tasks |
| `docs/sprints/sprint_04_action_erp_integration.md` | Sprint 4 tasks |
| `docs/sprints/sprint_05_optimization_failsafe.md` | Sprint 5 tasks |

---

## 9. Makefile Commands

```bash
make run              # Run the service locally
make build            # Build binary
make migrate          # Run all pending migrations (up)
make migrate-down     # Roll back last migration
make test             # go test ./... (unit tests only)
make test-integration # go test -tags integration ./... (DB required)
make test-smoke       # go test -tags smoke ./tests/smoke/... (all services up)
make test-e2e         # go test -tags e2e ./tests/e2e/... (full env required)
make test-race        # go test -race -count=1 ./...
make test-cover       # Coverage report → coverage.html
make lint             # go vet + staticcheck
make docker-up        # docker compose up -d (infra only)
make docker-down      # docker compose down
```

---

## 10. Domain Boundaries — Do Not Cross

```
HTTP Handler  →  Use Case  →  Domain Interface  ←  Repository Implementation
     ↑                ↑              ↑
 No DB calls      No HTTP       No HTTP, no DB
```

- **Handlers** must NOT import `repository/` directly
- **Use cases** must NOT import `pgx` or any DB library
- **Domain** must have ZERO external dependencies
- **Repositories** implement domain interfaces — they do NOT define them
