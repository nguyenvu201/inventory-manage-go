# Sprint 2: Device Management & Calibration System

> **Goal:** Manage the device catalog and ensure scale accuracy.

---

## Metadata

| Field           | Value                                              |
|-----------------|----------------------------------------------------|
| Sprint          | 2 / 5                                              |
| Status          | 🔄 In Progress                                     |
| Created date    | 2026-04-12                                         |
| Owner           | BA                                                 |
| Priority        | High                                               |
| Dependencies    | Sprint 1 complete (Definition of Done met ✅)       |

---

## [INV-SPR02-TASK-001] — Device Registry

> **Task ID:** `INV-SPR02-TASK-001`  
> **Status:** 🔒 CLOSED  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** Developer  
> **Sprint:** 2  

**Description:** Develop a CRUD API to manage scale node information and associate devices with SKU codes.

**Acceptance Criteria:**
- [x] AC-01: Create schema migration for `devices` table with fields: `device_id` (PK), `name`, `sku_code`, `location`, `status`, `created_at`, `updated_at`
- [x] AC-02: Implement `POST /api/v1/devices` — register a new device, return 201 Created
- [x] AC-03: Implement `GET /api/v1/devices` — list devices with filter support by `status` and `sku_code`
- [x] AC-04: Implement `GET /api/v1/devices/:id` — device detail, return 404 if not found
- [x] AC-05: Implement `PUT /api/v1/devices/:id` and `DELETE /api/v1/devices/:id`
- [x] AC-06: Return HTTP 422 with a detailed error list when `device_id` is duplicate or `sku_code` is invalid
- [x] AC-07: Write integration tests covering all 5 endpoints

**Related Technologies:**
- HTTP Router: `chi` or `gin`
- Response format: JSON standard `{data, message, status}`
- Middleware: request logging, error recovery

**Notes / Dependencies:** Depends on Sprint 1 being complete.

**Status History:**
| Date       | From      | To             | Performed by | Notes                        |
|------------|-----------|----------------|--------------|------------------------------|
| 2026-04-12 | —         | DRAFT          | BA           | Task created                 |
| 2026-04-13 | DRAFT     | PENDING_REVIEW | BA           | Submitted for Lead approval  |
| 2026-04-13 | PENDING_REVIEW | APPROVED  | Lead         | Implementation plan approved |
| 2026-04-13 | APPROVED  | IN_PROGRESS    | Developer    | Started implementation       |
| 2026-04-13 | IN_PROGRESS | IN_REVIEW | Developer    | PR + Tests passing, ready for QA |
| 2026-04-13 | IN_REVIEW | REJECTED  | QA           | Tỷ lệ Test Coverage thấp hơn quy định (<80%) |
| 2026-04-13 | REJECTED  | IN_PROGRESS | Developer  | Fixing test coverage     |
| 2026-04-13 | IN_PROGRESS | IN_REVIEW | Developer  | Bổ sung tests, coverage đạt >89% |
| 2026-04-13 | IN_REVIEW | VERIFIED  | QA         | All ACs verified. Coverage >= 80%. All gates pass. |
| 2026-04-13 | VERIFIED  | CLOSED    | Lead       | Phê duyệt đóng task hoàn thiện |

---

## QA Rejection Report — INV-SPR02-TASK-001

**Verified ACs:**
- [x] AC-01: ✅ Create schema migration
- [x] AC-02: ✅ POST /api/v1/devices
- [x] AC-03: ✅ GET /api/v1/devices
- [x] AC-04: ✅ GET /api/v1/devices/:id
- [x] AC-05: ✅ PUT /api/v1/devices/:id and DELETE
- [x] AC-06: ✅ HTTP 422 on duplicate/invalid constraints
- [x] AC-07: ✅ Integration tests written

**Quality Gate Results:**
- Build: ✅ pass
- go vet: ✅ pass
- Tests: ✅ pass
- Race detector: ✅ pass
- Coverage: ❌ FAIL — `internal/handler` (33.8%), `internal/usecase` (71.0%). Minimum requirement for both is >= 80%.

**Required fixes before re-review:**
1. Developer cần viết thêm Unit Test cho các func còn thiếu trong `internal/usecase/device_usecase_test.go` (đặc biệt là GetDevice, ListDevices, RemoveDevice)
2. Bổ sung Unit Test cho `internal/handler/device_handler_test.go` (List, Update, Delete) để đẩy coverage lên >= 80%.

## [INV-SPR02-TASK-002] — Calibration Manager

> **Task ID:** `INV-SPR02-TASK-002`  
> **Status:** 🔍 PENDING_REVIEW  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 2  

**Description:** Build logic to store and retrieve calibration parameters (Zero/Span) and hardware configuration for each scale type.

**Acceptance Criteria:**
- [ ] AC-01: Create schema migration for `calibration_configs` table with: `id`, `device_id` (FK), `zero_value`, `span_value`, `unit`, `capacity_max`, `hardware_config (jsonb)`, `effective_from`, `created_by`
- [ ] AC-02: Implement `POST /api/v1/devices/:id/calibration` — create a new calibration config
- [ ] AC-03: Implement `GET /api/v1/devices/:id/calibration/active` — return the currently active configuration
- [ ] AC-04: Store `hardware_config` as flexible JSONB (ADC bits, sampling rate, filter type)
- [ ] AC-05: Automatically deactivate the old record (set `deactivated_at`) when a new record is created
- [ ] AC-06: Enforce only one active calibration record per device at any time using a partial unique index

**Related Technologies:**
- PostgreSQL JSONB, partial unique index
- Repository pattern: `CalibrationRepository`

**Notes / Dependencies:** Depends on `INV-SPR02-TASK-001` (requires `devices` table)

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created |
| 2026-04-13 | DRAFT | PENDING_REVIEW | BA | Trình QA/Lead review thiết kế schema DB |

---

## [INV-SPR02-TASK-003] — Calibration Workflow

> **Task ID:** `INV-SPR02-TASK-003`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 2  

**Description:** Implement the calibration update process for field installation and periodic checks, using database transactions to guarantee consistency.

**Acceptance Criteria:**
- [ ] AC-01: Implement use case `UpdateCalibration(deviceID, params)` with 4 sequential steps inside a single transaction
- [ ] AC-02: Validate all input parameters (zero_value < span_value, unit is a valid enum) before executing
- [ ] AC-03: Support `calibration_type`: `initial` | `periodic` | `drift_correction`
- [ ] AC-04: Prohibit deletion of calibration history records — only deactivation is allowed
- [ ] AC-05: Roll back the entire transaction if any step fails
- [ ] AC-06: Write unit tests for all business logic including the rollback scenario

**Related Technologies:**
- Clean Architecture: Use Case layer
- `pgx/v5` transaction with `BeginTx` / `Rollback`

**Notes / Dependencies:** Depends on `INV-SPR02-TASK-002`

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR02-TASK-004] — Audit Trail

> **Task ID:** `INV-SPR02-TASK-004`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 2  

**Description:** Record a full append-only history of all calibration changes and detect drift to ensure transparency and traceability.

**Acceptance Criteria:**
- [ ] AC-01: Create `calibration_audit_log` table with: `id`, `device_id`, `action`, `old_values (jsonb)`, `new_values (jsonb)`, `performed_by`, `performed_at`, `reason`
- [ ] AC-02: Every change to `calibration_configs` automatically generates an audit log entry
- [ ] AC-03: Audit log is append-only: no UPDATE or DELETE operations on this table are permitted
- [ ] AC-04: Implement `GET /api/v1/devices/:id/calibration/history` with time-based pagination
- [ ] AC-05: Drift detection: raise an alert if `zero_value` deviates by more than a configured threshold from the previous calibration
- [ ] AC-06: Write unit tests for the drift detection logic

**Related Technologies:**
- PostgreSQL trigger or application-level audit hook
- Drift detection: comparison logic in the service layer

**Notes / Dependencies:** Depends on `INV-SPR02-TASK-003`

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created |

---

## Definition of Done — Sprint 2

- [ ] All API endpoints work correctly verified via Postman/curl
- [ ] Calibration workflow leaves no inconsistent state on mid-process failure
- [ ] Audit log records every change with no data loss
- [ ] Unit + integration tests pass with coverage ≥ 80%
- [ ] Schema migrations are idempotent (`migrate up` can be re-run without errors)
- [ ] All tasks (TASK-001 → TASK-004) reach status 🔒 CLOSED

---

*Managed by: `.agents/rules/golang-ba.md` | Workflow: `docs/workflows/ba_task_creation_workflow.md`*
