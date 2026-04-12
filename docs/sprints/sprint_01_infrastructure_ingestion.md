# Sprint 1: Infrastructure & Data Ingestion (Ingestion Layer)

> **Goal:** Establish the environment and the ability to receive messages from the Gateway.

---

## Metadata

| Field           | Value                                              |
|-----------------|----------------------------------------------------|
| Sprint          | 1 / 5                                              |
| Status          | 🔄 In Progress                                     |
| Created date    | 2026-04-12                                         |
| Owner           | —                                                  |
| Priority        | High                                               |
| Dependencies    | — (first sprint, no prerequisites)                 |

---

## [INV-SPR01-TASK-001] — Setup Infrastructure

> **Task ID:** `INV-SPR01-TASK-001`  
> **Status:** ❌ REJECTED  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** Developer  
> **Sprint:** 1  

**Description:** Initialize the full technical foundation for the Golang project, including project structure, Docker services, and environment configuration.

**Acceptance Criteria:**
- [x] AC-01: Initialize Golang module (`go mod init`) following standard project layout (`cmd/`, `internal/`, `pkg/`, `config/`)
- [x] AC-02: Write `docker-compose.yml` including services: `postgresql`, `timescaledb`, `mosquitto` (MQTT broker)
- [x] AC-03: Configure TimescaleDB extension and create hypertable for the `telemetry_records` table
- [x] AC-04: Create `config/config.go` that reads all configuration from environment variables (`.env`)
- [x] AC-05: Write a Makefile with commands: `make run`, `make build`, `make migrate`, `make test`
- [x] AC-06: Update `README.md` with complete environment setup and startup instructions

**Related Technologies:**
- Golang 1.22+, Docker Compose v3.8
- PostgreSQL 15 + TimescaleDB 2.x
- Libraries: `pgx/v5`, `godotenv`, `zerolog`

**Notes / Dependencies:** No dependencies — this is the first task.

**Status History:**
| Date       | From           | To             | Performed by | Notes                                                            |
|------------|----------------|----------------|--------------|------------------------------------------------------------------|
| 2026-04-12 | —              | DRAFT          | BA           | Task created                                                     |
| 2026-04-12 | DRAFT          | PENDING_REVIEW | BA           | Submitted for Lead review                                        |
| 2026-04-12 | PENDING_REVIEW | APPROVED       | Lead         | Approved — ready for sprint                                      |
| 2026-04-12 | APPROVED       | IN_PROGRESS    | Developer    | Developer started work                                           |
| 2026-04-12 | IN_PROGRESS    | IN_REVIEW      | Developer    | All ACs implemented: go.mod, docker-compose, migration, config, Makefile, README |
| 2026-04-12 | IN_REVIEW      | REJECTED       | QA           | AC-01 FAIL: `pkg/` directory missing from project layout         |

### QA Rejection Report — INV-SPR01-TASK-001

**Verified ACs:**
- [x] AC-02: ✅ docker-compose.yml correct — 3 services, all healthchecks present
- [x] AC-03: ✅ Migration correct — hypertable, f_cnt unique index, LoRaWAN fields
- [x] AC-04: ✅ config.go correct — env tags, required fields, no hardcoded secrets
- [x] AC-05: ✅ Makefile complete — run, build, test, test-race, migrate, lint all present
- [x] AC-06: ✅ README.md complete — arch diagram, quick start, config table

**Failed ACs:**
- [ ] AC-01: ❌ `pkg/` directory does not exist. AC-01 explicitly requires project layout: `cmd/`, `internal/`, **`pkg/`**, `config/`. All other directories are present.

**Quality Gate Results (static review — Go binary not in system PATH):**
- Hardcoded secrets: ✅ None found
- Error handling: ✅ All errors wrapped with `fmt.Errorf("context: %w", err)`
- Domain interfaces in `internal/domain/`: ✅
- `.env` in `.gitignore`: ✅
- Migration up + down files: ✅
- Note: `_, _ = w.Write(...)` in health handler (main.go:41) — acceptable pattern for health endpoint

**Required fix before re-review:**
1. Create `pkg/` directory with a placeholder file (e.g., `pkg/README.md` or `pkg/.gitkeep`)

**Estimated fix time:** < 5 minutes

---

## [INV-SPR01-TASK-002] — Gateway Message Receiver

> **Task ID:** `INV-SPR01-TASK-002`  
> **Status:** ✅ APPROVED  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 1  

**Description:** Develop a service to receive telemetry messages from the Gateway via MQTT/ChirpStack, handle reconnection, and feed messages into the processing pipeline.

**Acceptance Criteria:**
- [ ] AC-01: Connect to MQTT broker using `paho.mqtt.golang` with config injected via environment variables
- [ ] AC-02: Subscribe to topic pattern `application/+/device/+/event/up`
- [ ] AC-03: Implement automatic reconnect logic with exponential backoff on broker disconnection
- [ ] AC-04: Parse JSON payload from ChirpStack uplink frame into an internal struct
- [ ] AC-05: Push valid messages into a buffered internal channel for downstream processing
- [ ] AC-06: Log every received message including `device_id` and `trace_id`
- [ ] AC-07: Extend `TelemetryPayload` with LoRaWAN metadata from ChirpStack: `rssi`, `snr`, `f_cnt` (frame counter), `spreading_factor`, `sample_count`
- [ ] AC-08: If `sample_count > 1` in payload, treat `raw_weight` as pre-averaged by the node; otherwise apply a server-side 5-reading moving average before storing
- [ ] AC-09: Validate `f_cnt` field is present and is an unsigned integer; forward to storage layer for duplicate detection

**Related Technologies:**
- `eclipse/paho.mqtt.golang`
- ChirpStack v4 Application Server API / Payload format
- Pattern: pub/sub with buffered channel

**Notes / Dependencies:** Depends on `INV-SPR01-TASK-001` (requires Docker + MQTT broker) — ⛔ awaiting TASK-001 completion

**Status History:**
| Date       | From           | To             | Performed by | Notes                          |
|------------|----------------|----------------|--------------|--------------------------------|
| 2026-04-12 | —              | DRAFT          | BA           | Task created                   |
| 2026-04-12 | DRAFT          | DRAFT          | BA           | AC-07/08/09 added — customer PDF requirement update |
| 2026-04-12 | DRAFT          | PENDING_REVIEW | BA           | Submitted for Lead review      |
| 2026-04-12 | PENDING_REVIEW | APPROVED       | Lead         | Approved — awaiting TASK-001   |

---

## [INV-SPR01-TASK-003] — Telemetry Validator & Data Parser

> **Task ID:** `INV-SPR01-TASK-003`  
> **Status:** ✅ APPROVED  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 1  

**Description:** Build a validation and decoding layer for scale sensor payloads, ensuring only valid data enters the storage pipeline.

**Acceptance Criteria:**
- [ ] AC-01: Define `TelemetryPayload` struct with required fields: `device_id`, `raw_weight`, `battery_level`
- [ ] AC-02: Validate that `device_id` is non-empty, `raw_weight` is within valid range, `battery_level` is 0–100
- [ ] AC-03: Decode Base64-encoded payloads received from ChirpStack
- [ ] AC-04: Return a structured `ValidationError` when data fails validation
- [ ] AC-05: Unit test coverage ≥ 80% for all validation logic (table-driven tests)
- [ ] AC-06: Handle partial payloads (missing fields) gracefully without panicking
- [ ] AC-07: Extend `TelemetryPayload` with LoRaWAN metadata from ChirpStack: `rssi`, `snr`, `f_cnt` (frame counter), `spreading_factor`, `sample_count`
- [ ] AC-08: If `sample_count > 1` in payload, treat `raw_weight` as pre-averaged by the node; otherwise apply a server-side 5-reading moving average before storing
- [ ] AC-09: Validate `f_cnt` field is present and is an unsigned integer; forward to storage layer for duplicate detection

**Related Technologies:**
- `encoding/json`, `encoding/base64`
- `go-playground/validator/v10`
- Testing: `testify/assert`, table-driven tests

**Notes / Dependencies:** Depends on `INV-SPR01-TASK-002` (requires payload format from Receiver) — ⛔ awaiting TASK-002 completion

**Status History:**
| Date       | From           | To             | Performed by | Notes                                               |
|------------|----------------|----------------|--------------|-----------------------------------------------------|
| 2026-04-12 | —              | DRAFT          | BA           | Task created                                        |
| 2026-04-12 | DRAFT          | DRAFT          | BA           | AC-07/08/09 added — customer PDF requirement update |
| 2026-04-12 | DRAFT          | PENDING_REVIEW | BA           | Submitted for Lead review                           |
| 2026-04-12 | PENDING_REVIEW | APPROVED       | Lead         | Approved — awaiting TASK-002                        |

---

## [INV-SPR01-TASK-004] — Raw Storage

> **Task ID:** `INV-SPR01-TASK-004`  
> **Status:** ✅ APPROVED  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 1  

**Description:** Persist raw telemetry data to the TimescaleDB database for historical retrieval and audit purposes.

**Acceptance Criteria:**
- [ ] AC-01: Create schema migration for `raw_telemetry` table (device_id, raw_weight, battery_level, received_at, payload_json)
- [ ] AC-02: Set `received_at` as the TimescaleDB hypertable time-based partition key
- [ ] AC-03: Implement repository pattern: `TelemetryRepository` interface + PostgreSQL implementation
- [ ] AC-04: Use batch insert when more than 10 records arrive simultaneously
- [ ] AC-05: Add an index on `device_id` to optimize historical queries
- [ ] AC-06: Write an integration test: insert → query to verify data is stored correctly
- [ ] AC-07: Add unique constraint on `(device_id, f_cnt)` — duplicate LoRaWAN packets received from multiple gateways are silently discarded (idempotent ingestion)
- [ ] AC-08: Store `rssi` and `snr` fields in `raw_telemetry` for downstream signal quality reporting

**Related Technologies:**
- `pgx/v5` connection pool, `squirrel` query builder
- TimescaleDB `create_hypertable()`
- Migration tool: `golang-migrate/migrate`

**Notes / Dependencies:** Depends on `INV-SPR01-TASK-003` (only validated payloads are stored) — ⛔ awaiting TASK-003 completion

**Status History:**
| Date       | From           | To             | Performed by | Notes                                               |
|------------|----------------|----------------|--------------|-----------------------------------------------------|
| 2026-04-12 | —              | DRAFT          | BA           | Task created                                        |
| 2026-04-12 | DRAFT          | DRAFT          | BA           | AC-07/08 added — customer PDF requirement update    |
| 2026-04-12 | DRAFT          | PENDING_REVIEW | BA           | Submitted for Lead review                           |
| 2026-04-12 | PENDING_REVIEW | APPROVED       | Lead         | Approved — awaiting TASK-003                        |

---

## Definition of Done — Sprint 1

- [ ] Docker Compose starts successfully with all services healthy
- [ ] Service receives real MQTT messages from a gateway or emulator
- [ ] Raw data is written to TimescaleDB and queryable via SQL
- [ ] Unit tests pass with ≥ 80% coverage for the Validator layer
- [ ] No errors from `go vet` or `staticcheck` across the entire codebase
- [ ] All tasks (TASK-001 → TASK-004) reach status 🔒 CLOSED

---

*Managed by: `.agents/rules/golang-ba.md` | Workflow: `docs/workflows/ba_task_creation_workflow.md`*
