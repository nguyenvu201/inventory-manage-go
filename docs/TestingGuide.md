# Testing Guide — Inventory Management System

This document outlines the testing strategy, tools, and best practices for the Inventory Management System, following **FDA 21 CFR Part 11** and **IEC 62304 Software Safety Class B** requirements.

With our transition to a new architecture (`Gin`, `Viper`, `Zap`, `Redis`, `pgx/v5` pool), testing must reflect the updated decoupling of components and explicit dependency injection.

---

## 1. Test Organization & Commands

We have structured tests into four tiers. All tests must be deterministic and CI/CD ready.

| Type | Location | Build Tag | Command |
|------|----------|-----------|---------|
| Unit | `internal/**/*_test.go` | _(none)_ | `make test` |
| Integration | `internal/repository/postgres/*_test.go` | `integration` | `make test-integration` |
| Smoke | `tests/smoke/` | `smoke` | `make test-smoke` |
| E2E | `tests/e2e/` | `e2e` | `make test-e2e` |
| Race | all | _(none)_ | `make test-race` |
| Coverage | all | _(none)_ | `make test-cover` |

---

## 2. Test Requirements (FDA Compliance)

### 2.1 Traceability Header
Every test file MUST begin with a header mapping tests to Jira/Sprint Task IDs and Acceptance Criteria. This provides direct traceability for audits.

```go
// Package model_test implements tests for INV-SPR01-TASK-003
// AC Coverage:
//   AC-01: TestTelemetry_Validate_ValidPayload
//   AC-02: TestTelemetry_Validate_MissingFields
// IEC 62304 Classification: Software Safety Class B
package model_test
```

### 2.2 Coverage Minimums
*   `internal/model/` - **≥ 90%**
*   `internal/service/impl/` - **≥ 85%**
*   `internal/repository/postgres/` - **≥ 80%** (Integration tests apply)
*   `internal/controller/` - **≥ 80%**
*   `tests/smoke/` - **100%** (All must pass)

---

## 3. Unit Tests

Unit tests are used for services, models, and middleware.
*   **Do not mock the database in unit tests**. We use integration tests for DB interactions.
*   **Table-Driven Design is MANDATORY** for all business logic validations.
*   **Dependency Injection:** Inject mocked interfaces (like `mockDeviceRepo`) into services.

**Example: Table-Driven Service Logic:**

```go
func TestDeviceService_RegisterDevice(t *testing.T) {
	tests := []struct {
		name    string
		input   *model.Device
		repoErr error
		wantErr bool
		errMsg  string
	}{
		{
			name:  "AC-01: valid device",
			input: &model.Device{DeviceID: "SCALE-01", Status: model.StatusActive},
		},
		{
			name:    "AC-02: missing ID is rejected",
			input:   &model.Device{DeviceID: "", Status: model.StatusActive},
			wantErr: true, errMsg: "device_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockDeviceRepo{errResponse: tt.repoErr}
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

---

## 4. Gin Controller Tests (httptest)

Mock HTTP requests via `httptest.NewRecorder()`. We verify the HTTP Status Code as well as our standardized `response.ErrCodeXxx` JSON envelope.

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
			name:       "AC-03: returns device when found",
			deviceID:   "SCALE-01",
			svcReturn:  &model.Device{DeviceID: "SCALE-01"},
			wantStatus: http.StatusOK,
			wantCode:   response.ErrCodeSuccess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockDeviceService{result: tt.svcReturn, err: tt.svcErr}
			ctrl := controller.NewDeviceController(mockSvc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: tt.deviceID}}
			
			// Mocking request context IDs
			c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
			c.Set("trace_id", "trace-123")

			ctrl.GetDevice(c)

			assert.Equal(t, tt.wantStatus, w.Code)
			var body map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &body)
			assert.Equal(t, float64(tt.wantCode), body["code"])
		})
	}
}
```

---

## 5. Integration Tests (Testcontainers)

Repository files MUST be tested against a real PostgreSQL container. **We do not use go-mock for Repos.**

```go
//go:build integration
package postgres_test

func TestDeviceRepository_Save(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	ctx := context.Background()

	// Spin up TimescaleDB container
	pgContainer, err := pgmodule.Run(ctx,
		"timescale/timescaledb:latest-pg15",
		pgmodule.WithDatabase("test_inventory"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { pgContainer.Terminate(ctx) })

	connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")
	runMigrations(t, connStr) // Run golang-migrate
	
	pool := connectPool(t, connStr)
	repo := postgres.NewDeviceRepository(pool)

	t.Run("AC-01: duplicate throws unique constraint error", func(t *testing.T) {
		d := &model.Device{DeviceID: "SCALE-DUP"}
		require.NoError(t, repo.Save(ctx, d))
		err := repo.Save(ctx, d)
		require.ErrorIs(t, err, model.ErrDuplicateDevice)
	})
}
```

---

## 6. Safe Logging in Tests

Because background workers and some controllers depend on `global.Logger` (Zap), tests where `initialize.Run()` is NOT executed will have a `nil` pointer. 
Always use a safe logging helper inside components that log:

```go
// log returns the global logger or a no-op logger for tests.
func log() *zap.Logger {
	if global.Logger != nil {
		return global.Logger.Logger
	}
	return zap.NewNop()
}
```
Then use `log().Info(...)` instead of `global.Logger.Info(...)` everywhere in that package.

---

## 7. Edge Case Definitions

To pass QA Review, tests must check:
- **Zero Values:** `RawWeight: 0` is valid, `DeviceID: ""` is invalid.
- **Null Safety:** Payloads with missing LoRaWAN fields (e.g., no `fCnt`) failing gracefully.
- **State Changes:** Devices updating status correctly (e.g., `StatusActive` -> `StatusMaintenance`).
- **Concurrent Maps/States:** Always test workers and data structure states using `go test -race`.
