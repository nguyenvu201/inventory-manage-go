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
├── docker-compose.yml
├── Makefile
├── .env.example
└── README.md
```

**Naming rules:**
- Files: `snake_case.go` (e.g., `telemetry_repository.go`)
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

// Load with caarlos0/env or godotenv
```

**NEVER hardcode:** host, port, password, API key, DSN strings

### 3.4 Database — pgx/v5 with transactions

```go
// Use transactions for multi-table operations
tx, err := pool.Begin(ctx)
if err != nil {
    return fmt.Errorf("db.Begin: %w", err)
}
defer tx.Rollback(ctx) // safe no-op after Commit

// ... operations ...

if err := tx.Commit(ctx); err != nil {
    return fmt.Errorf("db.Commit: %w", err)
}
```

**Repository pattern — always use interfaces:**

```go
// domain/telemetry/repository.go (interface — consumer side)
type Repository interface {
    Save(ctx context.Context, t *Telemetry) error
    FindByDeviceID(ctx context.Context, deviceID string, from, to time.Time) ([]*Telemetry, error)
}

// repository/postgres/telemetry_repository.go (implementation)
type telemetryRepository struct {
    db *pgxpool.Pool
}
```

### 3.5 Concurrency — protect all shared state

```go
// Use sync.RWMutex for shared maps/caches
type Cache struct {
    mu    sync.RWMutex
    items map[string]Item
}

func (c *Cache) Set(key string, item Item) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items[key] = item
}

func (c *Cache) Get(key string) (Item, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    item, ok := c.items[key]
    return item, ok
}
```

**Always run:** `go test -race ./...` before submitting PR

### 3.6 HTTP Handler pattern (chi/gin)

```go
// handler/telemetry_handler.go
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

// Standard JSON response wrapper
type Response struct {
    Data    any    `json:"data,omitempty"`
    Message string `json:"message"`
    Status  int    `json:"status"`
}
```

### 3.7 Migrations — golang-migrate only

```bash
# Create new migration
migrate create -ext sql -dir migrations -seq create_raw_telemetry_table

# Never ALTER tables manually — always create a new migration file
```

Migration file naming: `NNNNNN_description.up.sql` / `NNNNNN_description.down.sql`

---

## 4. Testing Requirements

### 4.1 Minimum coverage: ≥ 80% for all business logic

```go
// Table-driven tests — mandatory for business logic
func TestWeightConverter_Convert(t *testing.T) {
    tests := []struct {
        name      string
        rawWeight float64
        config    CalibrationConfig
        want      float64
        wantErr   bool
    }{
        {name: "normal reading", rawWeight: 5000, config: defaultCal, want: 25.0},
        {name: "zero reading", rawWeight: 0, config: defaultCal, want: 0.0},
        {name: "no calibration", rawWeight: 100, config: CalibrationConfig{}, wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Convert(tt.rawWeight, tt.config)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.InDelta(t, tt.want, got, 0.001)
        })
    }
}
```

### 4.2 Integration tests — testcontainers-go

```go
// Use testcontainers for DB integration tests
func TestTelemetryRepository_Save(t *testing.T) {
    ctx := context.Background()
    pgContainer, err := testcontainers.RunContainer(ctx, ...)
    // ...
}
```

### 4.3 Run before every PR

```bash
make test              # go test ./...
make test-race         # go test -race ./...
make lint              # go vet ./... && staticcheck ./...
```

---

## 5. Task Status Update Rules (FDA Audit Trail)

When you **start** a task (`APPROVED → IN_PROGRESS`):

```markdown
| Date       | From     | To          | Performed by | Notes              |
|------------|----------|-------------|--------------|-------------------|
| YYYY-MM-DD | APPROVED | IN_PROGRESS | Developer    | Started implementation |
```

When you **finish** a task and submit PR (`IN_PROGRESS → IN_REVIEW`):

1. Tick `[x]` for ALL Acceptance Criteria in the sprint file
2. Change status header to `👀 IN_REVIEW`
3. Add PR link in Notes column
4. Update `task_registry.md`

```markdown
| Date       | From        | To        | Performed by | Notes                          |
|------------|-------------|-----------|--------------|-------------------------------|
| YYYY-MM-DD | IN_PROGRESS | IN_REVIEW | Developer    | PR #XX — all ACs implemented  |
```

**NEVER skip** updating the Status History table — this is the FDA audit trail.

---

## 6. Code Review — Self-Checklist Before Submitting PR

```
[ ] All ACs from the task are implemented and ticked [x]
[ ] go test ./... passes
[ ] go test -race ./... passes (no race conditions)
[ ] go vet ./... produces no warnings
[ ] staticcheck ./... produces no warnings
[ ] No hardcoded secrets or connection strings
[ ] Every error is wrapped with fmt.Errorf("context: %w", err)
[ ] Every log entry has device_id and trace_id fields
[ ] All new DB tables have a migration file
[ ] Repository interfaces defined in domain/, not in repository/
[ ] Integration tests written for repository layer
[ ] Unit test coverage ≥ 80% for all new business logic
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
| FTP                    | `github.com/jlaffaye/ftp`        |
| SFTP                   | `github.com/pkg/sftp`            |
| Validation             | `github.com/go-playground/validator/v10` |
| Testing                | `github.com/stretchr/testify`    |
| Integration tests      | `github.com/testcontainers/testcontainers-go` |

---

## 8. Sprint File Reference (Current Sprint)

| File | Purpose |
|------|---------|
| `docs/sprints/task_registry.md` | Find your assigned tasks |
| `docs/sprints/sprint_01_infrastructure_ingestion.md` | Sprint 1 task details |
| `docs/sprints/sprint_02_device_calibration.md` | Sprint 2 task details |
| `docs/sprints/sprint_03_inventory_rules_engine.md` | Sprint 3 task details |
| `docs/sprints/sprint_04_action_erp_integration.md` | Sprint 4 task details |
| `docs/sprints/sprint_05_optimization_failsafe.md` | Sprint 5 task details |

**Current active task:** `INV-SPR01-TASK-001` — Setup Infrastructure → `🔄 IN_PROGRESS`

---

## 9. Makefile Commands

```bash
make run          # Run the service locally
make build        # Build binary
make migrate      # Run all pending migrations (up)
make migrate-down # Roll back last migration
make test         # go test ./...
make test-race    # go test -race -count=1 ./...
make lint         # go vet + staticcheck
make docker-up    # docker-compose up -d
make docker-down  # docker-compose down
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
