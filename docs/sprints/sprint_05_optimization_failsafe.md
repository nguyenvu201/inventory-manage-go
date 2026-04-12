# Sprint 5: Optimization & Fail-safe

> **Goal:** Ensure the system operates reliably, stably, and is ready for MVP launch.

---

## Metadata

| Field           | Value                                                        |
|-----------------|--------------------------------------------------------------|
| Sprint          | 5 / 5                                                        |
| Status          | 🔲 Not Started                                               |
| Created date    | 2026-04-12                                                   |
| Owner           | —                                                            |
| Priority        | High (MVP Gate)                                              |
| Dependencies    | Sprints 1–4 complete (all Definitions of Done met ✅)        |

---

## [INV-SPR05-TASK-001] — Power Strategy Monitoring

> **Task ID:** `INV-SPR05-TASK-001`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 5  

**Description:** Monitor device sleep/wake cycles and battery levels to manage device longevity and proactively alert on low-battery conditions.

**Acceptance Criteria:**
- [ ] AC-01: Track `battery_level` from every telemetry packet and persist to the time-series store
- [ ] AC-02: Detect late wake-up patterns: device reporting less frequently than the expected interval
- [ ] AC-03: Implement `GET /api/v1/devices/:id/battery` — current battery level and 7-day trend
- [ ] AC-04: Trigger notification when battery < 20% (warning) and < 5% (critical)
- [ ] AC-05: Calculate `estimated_days_remaining` based on the average battery consumption rate over the last 7 days
- [ ] AC-06: Write unit tests for `estimated_days_remaining` logic including cases with < 7 days of data

**Related Technologies:**
- TimescaleDB: linear regression on battery_level time series
- 7-day rolling average to calculate discharge rate

**Notes / Dependencies:** Depends on `INV-SPR04-TASK-001` (requires notification service)

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR05-TASK-002] — Error Handling

> **Task ID:** `INV-SPR05-TASK-002`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 5  

**Description:** Implement comprehensive error handling for all IoT system failure scenarios: invalid values, gateway connectivity loss, and unknown devices.

### Sub-task 5.2.1 — Handling Devices Sending Invalid Values

**Acceptance Criteria:**
- [ ] AC-01: Detect outliers: `raw_weight` outside ±3σ of the 24-hour rolling mean (Z-score)
- [ ] AC-02: Mark suspicious records with `suspect = true` rather than storing them as normal data
- [ ] AC-03: Exclude `suspect` data from threshold evaluation and reorder workflows
- [ ] AC-04: Alert admin when a device continuously sends suspect values more than 5 times in 1 hour

### Sub-task 5.2.2 — Gateway Internet Disconnection (Buffer Mechanism)

**Acceptance Criteria:**
- [ ] AC-05: Process batch messages received after reconnection in timestamp order, not arrival order
- [ ] AC-06: Detect and log a "reconnect burst": more than 50 messages arriving within the same second
- [ ] AC-07: Idempotent ingestion: do not process duplicate messages if the same message is received twice (unique message ID)

### Sub-task 5.2.3 — Unrecognized Device

**Acceptance Criteria:**
- [ ] AC-08: Telemetry from an unregistered `device_id`: log the event and route to an `unknown_devices` queue
- [ ] AC-09: Implement `GET /api/v1/devices/unknown` — list of devices not yet registered
- [ ] AC-10: Service must not crash when receiving messages from unknown devices
- [ ] AC-11: Alert admin if the same unknown `device_id` appears more than 10 times

**Related Technologies:**
- Statistical outlier detection: Z-score on a rolling window
- Idempotency: unique constraint on `(device_id, message_id)`
- Dead-letter queue pattern

**Notes / Dependencies:** Depends on the full system from Sprint 1 → Sprint 4

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR05-TASK-003] — Data Aggregation Optimization

> **Task ID:** `INV-SPR05-TASK-003`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 5  

**Description:** Optimize large-data queries on TimescaleDB to reduce reporting load time and storage costs.

**Acceptance Criteria:**
- [ ] AC-01: Identify slow queries (> 1s) using `pg_stat_statements` and `EXPLAIN ANALYZE`
- [ ] AC-02: Create TimescaleDB Continuous Aggregates for commonly used time buckets: 1h, 1d
- [ ] AC-03: Enable Compression Policy on `raw_telemetry` data older than 7 days
- [ ] AC-04: Enable Retention Policy: auto-delete raw data older than 90 days (keep aggregates)
- [ ] AC-05: Benchmark before/after: 30-day query must complete in < 200ms after optimization
- [ ] AC-06: Write `docs/timescale-optimization.md` documenting all applied configurations
- [ ] AC-07: Verify that continuous aggregate refresh schedule does not impact write throughput

**Related Technologies:**
- TimescaleDB: `add_continuous_aggregate_policy()`, `add_compression_policy()`, `add_retention_policy()`
- Indexing: BRIN index on time column, GIN index on JSONB
- Connection pooling: `pgbouncer` if needed

**Notes / Dependencies:** Depends on `INV-SPR03-TASK-004` (requires baseline queries for benchmarking)

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR05-TASK-004] — Final MVP Integration (End-to-End Test)

> **Task ID:** `INV-SPR05-TASK-004`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 5  

**Description:** Comprehensive end-to-end testing of the entire system, from physical sensors (or simulator) to the final ERP export file.

**Acceptance Criteria:**

### E2E Scenarios — all must pass:
- [ ] AC-01: **Happy path** — Simulator sends telemetry → received → inventory calculated → snapshot saved → API returns correct data
- [ ] AC-02: **Low stock trigger** — Reduce weight → alert email sent → PR created → FTP file exported
- [ ] AC-03: **Device reconnect** — Shut down MQTT for 5 minutes → restart → system processes batch messages in correct timestamp order
- [ ] AC-04: **Unknown device** — Send message from unregistered device → logged → service does not crash
- [ ] AC-05: **Calibration update** — Update zero/span → inventory automatically recalculated with new values

### Mandatory quality checklist:
- [ ] AC-06: All unit + integration tests pass (`go test ./...`)
- [ ] AC-07: No race conditions (`go test -race ./...` passes completely)
- [ ] AC-08: `go vet ./...` and `staticcheck ./...` produce no warnings
- [ ] AC-09: Dockerfile builds successfully, `docker-compose up` starts all services
- [ ] AC-10: README is complete: quick start, architecture diagram, config reference
- [ ] AC-11: All secrets are injected via environment variables — none are hardcoded

**Related Technologies:**
- MQTT simulator: `mosquitto_pub` script or Go test helper
- E2E testing: `testcontainers-go` to spin up database in CI
- Race detector: `go test -race -count=1 ./...`

**Notes / Dependencies:** Depends on the entire system Sprint 1 → Sprint 5

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR05-TASK-005] — Analytics Hub UI

> **Task ID:** `INV-SPR05-TASK-005`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 5  

**Description:** Build the Analytics Hub management UI, providing historical consumption trend charts, Day Zero prediction widgets, anomaly investigation log, and signal quality monitoring. Served as static files by the Go backend.

**Acceptance Criteria:**
- [ ] AC-01: Implement **Consumption Trends screen** — interactive line chart of `net_weight_kg` and `qty` over time per SKU; date range picker (7d / 30d / 90d / custom); downloadable as PNG
- [ ] AC-02: Implement **SKU Comparison chart** — overlay multiple SKUs on the same axis for side-by-side consumption comparison
- [ ] AC-03: Implement **Day Zero Prediction widget** — per-SKU card showing: current %, days_remaining gauge, day_zero_date, confidence label; sorted by urgency
- [ ] AC-04: Implement **Anomaly Log screen** — table of all anomaly events with columns: type badge, SKU, magnitude, detected_at, resolved badge; expandable row for resolution note; filter by type/status/date
- [ ] AC-05: Implement **Signal Quality screen** — line chart of RSSI and SNR per device over time; helps identify poor coverage or antenna issues
- [ ] AC-06: Implement **Export Report button** on Consumption Trends — triggers backend `POST /api/v1/erp/export/:customer_id/trigger` and shows download link on completion
- [ ] AC-07: All charts call existing backend APIs (`/api/v1/reports/consumption`, `/api/v1/inventory/:sku_code/forecast`, `/api/v1/anomalies`)
- [ ] AC-08: Serve UI at `/ui/analytics` via `embed.FS`

**Related Technologies:**
- Chart library: Chart.js or Recharts (if Vue-based)
- Same stack as Sprint 4 UI tasks (Vue 3 or Alpine.js)
- `embed.FS` for static file serving in Go

**Notes / Dependencies:** Depends on `INV-SPR03-TASK-004`, `INV-SPR03-TASK-005`, `INV-SPR03-TASK-006`

**Status History:**
| Date       | From | To    | Performed by | Notes                              |
|------------|------|-------|--------------|-------------------------------------|
| 2026-04-12 | —    | DRAFT | BA           | New task — customer PDF requirement|

---

## Definition of Done — Sprint 5 (MVP Gate)

- [ ] All 5 E2E scenarios pass completely
- [ ] `go test -race ./...` passes — no race conditions
- [ ] 30-day query response time < 200ms (measured via benchmark test)
- [ ] Battery monitoring correctly alerts on all low-battery scenarios
- [ ] System runs continuously for 30 minutes in Docker Compose without any panic or memory leak
- [ ] Analytics Hub UI renders all charts correctly with real data
- [ ] README and technical documentation are complete
- [ ] All tasks (TASK-001 → TASK-005) reach status 🔒 CLOSED

---

*Managed by: `.agents/rules/golang-ba.md` | Workflow: `docs/workflows/ba_task_creation_workflow.md`*
