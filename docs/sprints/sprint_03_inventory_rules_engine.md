# Sprint 3: Inventory Rules Engine

> **Goal:** Convert raw weight readings into actionable business information.

---

## Metadata

| Field           | Value                                                              |
|-----------------|--------------------------------------------------------------------|
| Sprint          | 3 / 5                                                              |
| Status          | 🔲 Not Started                                                     |
| Created date    | 2026-04-12                                                         |
| Owner           | —                                                                  |
| Priority        | High                                                               |
| Dependencies    | Sprint 1 & Sprint 2 complete (Definition of Done met ✅)            |

---

## [INV-SPR03-TASK-001] — Weight Conversion Logic

> **Task ID:** `INV-SPR03-TASK-001`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 3  

**Description:** Apply the conversion formula to transform an ADC raw value into a net weight using the active calibration parameters.

**Formula:**
```
Gross Weight = (raw_weight - zero_value) × (capacity_max ÷ span_value)
Net Weight   = Gross Weight - tare_weight
```

**Acceptance Criteria:**
- [ ] AC-01: Implement `WeightConverter` service that accepts `raw_weight` and an active `CalibrationConfig`
- [ ] AC-02: Calculate `gross_weight` from the ADC raw value using `zero_value` and `span_value`
- [ ] AC-03: Calculate `net_weight = gross_weight - tare_weight` (tare sourced from SKU config)
- [ ] AC-04: Support units kg, g, lb — automatically normalize to kg
- [ ] AC-05: Round results to 3 decimal places
- [ ] AC-06: Write unit tests covering edge cases: raw_weight = 0, raw_weight = max, no active calibration
- [ ] AC-07: Return a clear error if no active calibration exists for the device
- [ ] AC-08: Apply configurable measurement resolution rounding defined in `sku_configs.resolution_kg` (e.g., 0.1 for small bin, 0.5 for mid-size)

**Related Technologies:**
- Pure function — no side effects to simplify testing
- Floating point precision: consider `decimal` library if high precision is required

**Notes / Dependencies:** Depends on `INV-SPR02-TASK-002` (requires CalibrationConfig)

**Status History:**
| Date       | From | To    | Performed by | Notes                                               |
|------------|------|-------|--------------|-----------------------------------------------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created                                        |
| 2026-04-12 | DRAFT| DRAFT | BA           | AC-08 added — per-SKU resolution from customer PDF  |

---

## [INV-SPR03-TASK-002] — Inventory Calculation

> **Task ID:** `INV-SPR03-TASK-002`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 3  

**Description:** Convert net weight into a quantity (Qty) of stock-keeping units and a percentage of remaining inventory.

**Formula:**
```
Qty        = floor(net_weight ÷ unit_weight_kg)
Percentage = clamp((net_weight ÷ full_capacity_kg) × 100, 0, 100)
```

**Acceptance Criteria:**
- [ ] AC-01: Create schema migration for `sku_configs` table with: `sku_code` (PK), `unit_weight_kg`, `full_capacity_kg`, `tare_weight_kg`, `reorder_point_qty`, `unit_label`
- [ ] AC-02: Implement `InventoryCalculator` service computing `qty` and `percentage`
- [ ] AC-03: Persist results to `inventory_snapshots` table (device_id, sku_code, net_weight_kg, qty, percentage, snapshot_at)
- [ ] AC-04: Implement `GET /api/v1/inventory/current` — current inventory status for all SKUs
- [ ] AC-05: Implement `GET /api/v1/inventory/:sku_code/current` — current inventory for a single SKU
- [ ] AC-06: Use UPSERT pattern to update the latest snapshot rather than continuously inserting

**Related Technologies:**
- `math.Floor`, clamping logic
- TimescaleDB continuous aggregates for snapshots
- UPSERT: PostgreSQL `ON CONFLICT DO UPDATE`

**Notes / Dependencies:** Depends on `INV-SPR03-TASK-001`

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR03-TASK-003] — Threshold Rules

> **Task ID:** `INV-SPR03-TASK-003`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 3  

**Description:** Define and evaluate low-stock threshold rules per SKU, emitting events when breached while using a cooldown mechanism to prevent alert spam.

**Acceptance Criteria:**
- [ ] AC-01: Create `threshold_rules` table with: `sku_code`, `rule_type` (low_stock/critical/overstock), `trigger_percentage`, `trigger_qty`, `cooldown_minutes`, `is_active`
- [ ] AC-02: Implement `ThresholdEvaluator` service that evaluates rules after each inventory calculation
- [ ] AC-03: Emit a `ThresholdBreachedEvent` when a rule violation is detected
- [ ] AC-04: Implement cooldown mechanism — suppress duplicate events while inventory oscillates near the threshold
- [ ] AC-05: Implement CRUD API: `POST/GET/PUT/DELETE /api/v1/rules/thresholds`
- [ ] AC-06: Write unit tests covering: threshold breach, within cooldown window, rule is disabled

**Related Technologies:**
- Event-driven: publish `ThresholdBreachedEvent` to an internal event bus
- Cooldown state: `sync.Map` or Redis with TTL

**Notes / Dependencies:** Depends on `INV-SPR03-TASK-002`

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR03-TASK-004] — Historical Reporting API

> **Task ID:** `INV-SPR03-TASK-004`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 3  

**Description:** Provide an API for the Dashboard to display consumption trends over time, with aggregation and response caching.

**Acceptance Criteria:**
- [ ] AC-01: Implement `GET /api/v1/reports/consumption` with params: `sku_code`, `from`, `to`, `interval` (1h/1d/1w)
- [ ] AC-02: Return an array of `{timestamp, net_weight_kg, qty, percentage}` grouped by the requested interval
- [ ] AC-03: Use TimescaleDB `time_bucket()` for aggregation
- [ ] AC-04: Implement `GET /api/v1/reports/consumption/summary` — total consumption per SKU over the period
- [ ] AC-05: Support cursor-based pagination in response
- [ ] AC-06: Cache responses for 5 minutes for queries spanning more than 7 days
- [ ] AC-07: Response time < 500ms for a 30-day query (benchmark test required before merge)

**Related Technologies:**
- TimescaleDB: `time_bucket()`, continuous aggregates, compression policy
- Cache: Redis or in-memory with TTL
- Query optimization: run `EXPLAIN ANALYZE` before merging

**Notes / Dependencies:** Depends on `INV-SPR03-TASK-002`, `INV-SPR03-TASK-003`

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR03-TASK-005] — Stock-out / Day Zero Forecasting

> **Task ID:** `INV-SPR03-TASK-005`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 3  

**Description:** Predict when each SKU will run out of stock based on the 7-day rolling average consumption rate, and proactively alert when the predicted stock-out date is within a configurable warning horizon.

**Acceptance Criteria:**
- [ ] AC-01: Compute 7-day rolling average daily consumption rate per SKU from `inventory_snapshots` time series
- [ ] AC-02: Calculate `day_zero_date = today + (current_net_weight_kg ÷ avg_daily_consumption_kg)` — handle edge case of zero / negative consumption rate
- [ ] AC-03: Add `day_zero_date` and `days_remaining` (integer) fields to the inventory snapshot response
- [ ] AC-04: Implement `GET /api/v1/inventory/:sku_code/forecast` returning: `avg_daily_consumption_kg`, `day_zero_date`, `days_remaining`, `confidence` (based on data completeness)
- [ ] AC-05: Emit a `StockoutForecastAlert` when `days_remaining` falls below configurable threshold (default: 7 days)
- [ ] AC-06: Add `forecast_horizon_days` config field to `threshold_rules` per SKU
- [ ] AC-07: Unit test all edge cases: new SKU with < 7 days data, zero consumption, consumption increasing

**Related Technologies:**
- TimescaleDB `time_bucket()` + `lag()` window function for daily delta
- Pure calculation service — no side effects

**Notes / Dependencies:** Depends on `INV-SPR03-TASK-002` (requires inventory_snapshots data)

**Status History:**
| Date       | From | To    | Performed by | Notes                              |
|------------|------|-------|--------------|------------------------------------|
| 2026-04-12 | —    | DRAFT | BA           | New task — customer PDF requirement|

---

## [INV-SPR03-TASK-006] — Consumption Anomaly Detection

> **Task ID:** `INV-SPR03-TASK-006`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 3  

**Description:** Detect abnormal consumption patterns — sudden weight drops (possible leak or theft) or unexpected weight increases (calibration drift or mis-reading) — and emit structured anomaly events.

**Acceptance Criteria:**
- [ ] AC-01: Define anomaly types: `sudden_drop | unexpected_increase | measurement_error`
- [ ] AC-02: Detect `sudden_drop`: net_weight decreases by more than `anomaly_threshold_pct` (configurable, default 20%) between two consecutive readings
- [ ] AC-03: Detect `unexpected_increase`: net_weight increases by more than `overstock_threshold_pct` relative to `full_capacity_kg`
- [ ] AC-04: Detect `measurement_error`: reading outside physically possible range (negative weight or > 110% of capacity)
- [ ] AC-05: Persist anomalies to `anomaly_events` table (id, device_id, sku_code, anomaly_type, magnitude_kg, detected_at, resolved_at, resolution_note)
- [ ] AC-06: Emit `AnomalyEvent` to internal event bus — notification service subscribes
- [ ] AC-07: Implement `GET /api/v1/anomalies` with filters: `sku_code`, `type`, `from`, `to`, `resolved`; paginated
- [ ] AC-08: Implement `PUT /api/v1/anomalies/:id/resolve` with `resolution_note` — marks anomaly as investigated
- [ ] AC-09: Unit test all 3 anomaly types with boundary values

**Related Technologies:**
- Event bus: internal channel pattern (same as `ThresholdBreachedEvent`)
- Append-only `anomaly_events` table with soft resolution

**Notes / Dependencies:** Depends on `INV-SPR03-TASK-002`, `INV-SPR04-TASK-001` (notification service)

**Status History:**
| Date       | From | To    | Performed by | Notes                              |
|------------|------|-------|--------------|------------------------------------|
| 2026-04-12 | —    | DRAFT | BA           | New task — customer PDF requirement|

---

## [INV-SPR03-TASK-007] — Inventory Dashboard API

> **Task ID:** `INV-SPR03-TASK-007`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 3  

**Description:** Provide optimized backend API endpoints to power the real-time Inventory Dashboard UI, including a live-push mechanism via WebSocket/SSE.

**Acceptance Criteria:**
- [ ] AC-01: Implement `GET /api/v1/dashboard/summary` — aggregated stats: total_skus, low_stock_count, critical_count, offline_node_count, active_alert_count
- [ ] AC-02: Implement `GET /api/v1/dashboard/inventory` — real-time table per SKU: `sku_code`, `net_weight_kg`, `qty`, `percentage`, `day_zero_date`, `days_remaining`, `device_status`, `last_seen_at`
- [ ] AC-03: Add computed `status` field to each row: `normal | low_stock | critical | out_of_stock | offline` — derived from threshold rules + connection status
- [ ] AC-04: Support filter `?status=low_stock,critical` and sort by `?sort=days_remaining:asc`
- [ ] AC-05: Implement `GET /api/v1/dashboard/stream` SSE endpoint — pushes `InventoryUpdatedEvent` whenever a new telemetry snapshot is written
- [ ] AC-06: Dashboard summary API response time < 100ms (cached with 10-second TTL)
- [ ] AC-07: Integration test: simulate 5 devices reporting → verify dashboard reflects all changes within 1 cycle

**Related Technologies:**
- SSE: `net/http` with `text/event-stream` content type
- Dashboard cache: in-memory with `sync.RWMutex` or Redis
- Derived `status` field: service layer computation, not DB column

**Notes / Dependencies:** Depends on `INV-SPR03-TASK-005` (day_zero_date), `INV-SPR03-TASK-003` (threshold rules)

**Status History:**
| Date       | From | To    | Performed by | Notes                              |
|------------|------|-------|--------------|------------------------------------|
| 2026-04-12 | —    | DRAFT | BA           | New task — customer PDF requirement|

---

## Definition of Done — Sprint 3

- [ ] Weight and inventory calculation formulas are fully covered by passing unit tests
- [ ] Threshold rules fire accurately with no duplicate events
- [ ] Historical API returns correct data with response time < 500ms for 30-day queries
- [ ] Snapshots are saved after every new telemetry ingestion
- [ ] Day Zero forecasting returns valid predictions for all SKUs with sufficient data
- [ ] Anomaly detection fires correctly for all 3 anomaly types
- [ ] Dashboard API responds < 100ms with SSE stream verified in integration test
- [ ] Overall Sprint 3 test coverage ≥ 80%
- [ ] All tasks (TASK-001 → TASK-007) reach status 🔒 CLOSED

---

*Managed by: `.agents/rules/golang-ba.md` | Workflow: `docs/workflows/ba_task_creation_workflow.md`*
