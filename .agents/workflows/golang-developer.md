# /golang-developer — Developer Task Implementation Workflow

Use this workflow when you are assigned a task (`🔄 IN_PROGRESS`) and need to implement it following FDA 21 CFR Part 11 / IEC 62304 standards.

---

## Step 1 — Read Your Task

```bash
cat docs/sprints/task_registry.md
```

1. Note the **Task ID** (e.g., `INV-SPR01-TASK-003`)
2. Open the corresponding sprint file
3. Read the **entire task block**: Description, ALL ACs, Related Technologies, Dependencies
4. Build a mental map: **which AC → which test type?**
   - Validator logic → Unit test
   - DB operations → Integration test
   - Service startup → Smoke test
   - Full flow (MQTT → DB → API) → E2E test

---

## Step 2 — Update Task: APPROVED → IN_PROGRESS

```markdown
> **Status:** 🔄 IN_PROGRESS

| Date       | From     | To          | Performed by | Notes                    |
|------------|----------|-------------|--------------|--------------------------|
| YYYY-MM-DD | APPROVED | IN_PROGRESS | Developer    | Started implementation   |
```

Update `docs/sprints/task_registry.md` status column as well.

---

## Step 3 — Set Up Environment

```bash
# Infrastructure: TimescaleDB + MQTT (infra only, no app build needed)
make docker-up
docker compose ps   # verify: inventory_db (healthy), inventory_mqtt (healthy)

# Apply migrations
docker exec -i inventory_db psql -U inventory_user -d inventory_db < migrations/000001_create_raw_telemetry.up.sql

# Or if golang-migrate is installed:
make migrate
```

---

## Step 4 — Plan Your Tests BEFORE Writing Code (TDD)

Before writing any implementation code, answer:

```
For each AC in the task:
  1. What is the UNIT being tested? (function, method, type)
  2. What are ALL valid inputs?
  3. What are ALL error cases / edge cases?
  4. What is the EXPECTED behavior?
```

Create the test file first with the traceability header:

```go
// Package xxx_test implements tests for [TASK-ID]
// AC Coverage:
//   AC-01: Test_FunctionName_ValidCase
//   AC-02: Test_FunctionName_EdgeCase
//   AC-03: TestRepository_Save_DuplicateFCnt
// IEC 62304 Classification: Software Safety Class B
package xxx_test
```

---

## Step 5 — Implement Code + Write Tests (Layer by Layer)

Work through ACs **in order**. For each AC:

### Layer 1 — Domain & Validators (Unit Tests)

```bash
# Create test file first
touch internal/domain/telemetry/validator_test.go

# Write table-driven tests
# Run continuously while implementing
go test ./internal/domain/... -v -run TestValidator
```

**Unit test requirements:**
- Table-driven format (mandatory)
- Cover: zero values, boundary values (`0`, `100`, `101`, `-1`), missing fields
- Each test case name MUST reference its AC (e.g., `"AC-02: battery_level=101 rejected"`)
- `require.NoError` / `require.Error` — never just `assert`

```go
func TestTelemetryValidator_Validate(t *testing.T) {
    tests := []struct {
        name    string
        input   TelemetryPayload
        wantErr bool
        errMsg  string
    }{
        {"AC-01: valid full payload", validPayload(), false, ""},
        {"AC-02: battery=101 rejected", payloadWith(BatteryLevel: 101), true, "battery_level"},
        {"AC-02: battery=0 accepted",   payloadWith(BatteryLevel: 0),   false, ""},
        {"AC-02: battery=-1 rejected",  payloadWith(BatteryLevel: -1),  true, "battery_level"},
        {"empty device_id rejected",    payloadWith(DeviceID: ""),      true, "device_id"},
        {"negative raw_weight rejected",payloadWith(RawWeight: -1),     true, "raw_weight"},
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

### Layer 2 — Use Cases (Unit Tests with mocks)

```bash
touch internal/usecase/telemetry_usecase_test.go
go test ./internal/usecase/... -v -run TestTelemetryUsecase
```

**Mock the repository interface** (defined in domain/):
```go
type mockTelemetryRepo struct{ mock.Mock }
func (m *mockTelemetryRepo) Save(ctx context.Context, t *domain.RawTelemetry) error {
    return m.Called(ctx, t).Error(0)
}
```

### Layer 3 — Repository (Integration Tests with testcontainers)

```bash
touch internal/repository/postgres/telemetry_repository_test.go
go test ./internal/repository/... -tags integration -v -run TestTelemetryRepository
```

**Always use testcontainers — NEVER mock the DB:**
```go
//go:build integration

func TestTelemetryRepository_Save(t *testing.T) {
    ctx := context.Background()
    pgContainer := startTimescaleContainer(t, ctx)
    repo := setupRepo(t, pgContainer)

    t.Run("AC-01: save valid record", func(t *testing.T) { ... })
    t.Run("AC-03: duplicate f_cnt → ErrDuplicatePacket", func(t *testing.T) { ... })
    t.Run("battery_level=0 stored correctly", func(t *testing.T) { ... })
}
```

### Layer 4 — HTTP Handlers (Unit Tests with httptest)

```bash
touch internal/handler/telemetry_handler_test.go
go test ./internal/handler/... -v -run TestTelemetryHandler
```

```go
func TestTelemetryHandler_GetCurrent(t *testing.T) {
    tests := []struct {
        name       string
        setupMock  func(m *mockUsecase)
        wantStatus int
        wantBody   string
    }{
        {
            name: "AC-05: returns 200 with current inventory",
            setupMock: func(m *mockUsecase) {
                m.On("GetCurrentInventory", mock.Anything).
                    Return(&domain.InventorySummary{Total: 25.5}, nil)
            },
            wantStatus: http.StatusOK,
        },
        {
            name: "returns 500 on usecase error",
            setupMock: func(m *mockUsecase) {
                m.On("GetCurrentInventory", mock.Anything).
                    Return(nil, errors.New("db down"))
            },
            wantStatus: http.StatusInternalServerError,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            uc := &mockUsecase{}
            tt.setupMock(uc)
            h := NewTelemetryHandler(uc)

            req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/current", nil)
            rr := httptest.NewRecorder()
            h.GetCurrent(rr, req)

            assert.Equal(t, tt.wantStatus, rr.Code)
        })
    }
}
```

### Layer 5 — Smoke Tests (Service startup verification)

Create / update `tests/smoke/smoke_test.go` for any new endpoint or service dependency:

```go
//go:build smoke

// TestSmoke_[Feature] — run after docker compose up
func TestSmoke_NewEndpoint(t *testing.T) {
    resp, err := http.Get(baseURL() + "/api/v1/your-new-endpoint")
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

```bash
# Run after make docker-up && make run (in background)
make test-smoke
```

### Layer 6 — E2E Tests (Full flow)

Create `tests/e2e/[feature]_flow_test.go` for each new complete ingestion/processing flow:

```go
//go:build e2e

// TestE2E_[FlowName] — requires: all docker services + app running
func TestE2E_TelemetryIngestionFlow(t *testing.T) {
    // 1. Publish MQTT uplink (simulate gateway)
    // 2. Poll DB with require.Eventually (max 5s)
    // 3. Verify stored data matches payload
    // 4. Verify API response
    // 5. Test idempotency (publish duplicate, count stays 1)
}
```

```bash
# Run with full environment
make docker-up
make run &        # or go run cmd/server/main.go &
make test-e2e
```

---

## Step 6 — Run All Quality Gates

**Run in this ORDER. Do not skip any.**

```bash
# 1. Unit tests
make test
# Expected: ok  inventory-manage/internal/...

# 2. Race detector
make test-race
# Expected: zero DATA RACE warnings

# 3. Integration tests (DB required)
make test-integration
# Expected: ok  inventory-manage/internal/repository/...

# 4. Coverage (must meet minimums per package)
make test-cover
# Check output — look for packages < 80%
# domain/ must be ≥ 90%, usecase/ ≥ 85%

# 5. Smoke tests (infra running)
make test-smoke
# Expected: all PASS, 0 FAIL

# 6. E2E tests (full env)
make test-e2e
# Expected: all PASS, especially idempotency

# 7. Lint
make lint
# Expected: zero warnings from go vet and staticcheck
```

---

## Step 7 — Submit PR: IN_PROGRESS → IN_REVIEW

### Pre-PR checklist:
```
── Implementation ─────────────────────────
[ ] All ACs ticked [x] in sprint file
[ ] No hardcoded secrets
[ ] All errors wrapped fmt.Errorf("ctx: %w", err)
[ ] All logs have device_id + trace_id
[ ] DB tables have up + down migration files

── Tests (FDA mandatory) ──────────────────
[ ] Unit tests: table-driven, traceability header, ≥ 90% for domain/
[ ] Integration tests: testcontainers, all repository methods
[ ] Smoke test: new endpoints covered
[ ] E2E test: new ingestion flows covered
[ ] Regression test: any bug fix has a test
[ ] make test → PASS
[ ] make test-race → PASS (zero races)
[ ] make test-integration → PASS
[ ] make test-smoke → PASS
[ ] make test-cover → all packages meet minimums
[ ] make lint → zero warnings
```

### Update sprint file:
```markdown
> **Status:** 👀 IN_REVIEW

| Date       | From        | To        | Performed by | Notes                                          |
|------------|-------------|-----------|--------------|------------------------------------------------|
| YYYY-MM-DD | IN_PROGRESS | IN_REVIEW | Developer    | PR #XX — all ACs + unit/integration/smoke/E2E |
```

### Commit convention:
```bash
git add .
git commit -m "feat(INV-SPR01-TASK-003): telemetry validator + parser

- domain/telemetry/validator.go: validate payload fields (AC-01, AC-02)
- domain/telemetry/parser.go: decode ChirpStack JSON uplink (AC-03)
- domain/telemetry/validator_test.go: 12 unit test cases, 94% coverage
- internal/repository/postgres/telemetry_repo_test.go: integration tests
- tests/smoke/smoke_test.go: health + DB + MQTT checks
- tests/e2e/ingestion_flow_test.go: full MQTT → DB flow + idempotency

Tests: unit ✓ integration ✓ smoke ✓ e2e ✓ race ✓"
```

---

## Step 8 — After Review Decision

### If REJECTED ❌
- Read ALL items in the rejection report — do not skip any
- Fix each failing AC and failing test
- Add Status History row: `REJECTED → IN_PROGRESS | Developer | Rework: <reason>`
- Re-run full test suite before resubmitting
- Change status → `👀 IN_REVIEW`, add history row

### If VERIFIED ✅ → CLOSED 🔒
- Lead closes the task (`CLOSED`)
- Start next `✅ APPROVED` task in registry order
- Check sprint dependency: never jump to Sprint N+1 without completing Sprint N

---

## Sprint 1 — Task Implementation Order

```
INV-SPR01-TASK-001  Setup Infrastructure          🏆 VERIFIED   ← DONE
INV-SPR01-TASK-002  Gateway Message Receiver       ✅ APPROVED   ← next
INV-SPR01-TASK-003  Telemetry Validator & Parser   ✅ APPROVED
INV-SPR01-TASK-004  Raw Storage                    ✅ APPROVED
```

**Tests to write per task:**

| Task | Unit | Integration | Smoke | E2E |
|------|------|-------------|-------|-----|
| TASK-002 | MQTT worker, message parser | MQTT subscriber | MQTT connectivity | Gateway → ingestion |
| TASK-003 | Validator (all fields), Parser (decoder) | — | — | Invalid packet rejected |
| TASK-004 | Repository methods | TimescaleDB save, duplicate f_cnt | DB table exists | Full ingestion + idempotency |

---

## Important File Paths

| File | When to use |
|------|-------------|
| `docs/sprints/task_registry.md` | Find task, update status |
| `docs/sprints/sprint_0N_*.md` | Read AC list, update history |
| `.agents/rules/golang-developer-rules.md` | Full coding + testing standards |
| `tests/smoke/` | Add smoke test for new endpoints |
| `tests/e2e/` | Add E2E test for new flows |
| `tests/testdata/` | Shared fixtures, JSON payloads, SQL seeds |
