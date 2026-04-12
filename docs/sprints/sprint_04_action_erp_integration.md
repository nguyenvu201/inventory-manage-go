# Sprint 4: Action & ERP Integration (Action Layer)

> **Goal:** Automate the reorder process and alert notifications when inventory reaches threshold.

---

## Metadata

| Field           | Value                                              |
|-----------------|----------------------------------------------------|
| Sprint          | 4 / 5                                              |
| Status          | 🔲 Not Started                                     |
| Created date    | 2026-04-12                                         |
| Owner           | —                                                  |
| Priority        | High                                               |
| Dependencies    | Sprint 3 complete (Definition of Done met ✅)       |

---

## [INV-SPR04-TASK-001] — Alert & Notification Service

> **Task ID:** `INV-SPR04-TASK-001`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 4  

**Description:** Develop a multi-channel alert service (Email/SMS/Web Push) that fires when inventory drops below threshold or a device encounters an error.

**Acceptance Criteria:**
- [ ] AC-01: Define `NotificationSender` interface with method `Send(ctx, AlertMessage) error`
- [ ] AC-02: Implement Email sender: SMTP + `gomail.v2`, support HTML templates
- [ ] AC-03: Implement SMS sender: Twilio integration, fully configured via environment variables
- [ ] AC-04: Implement Web Push sender: deliver notifications via WebSocket or SSE endpoint
- [ ] AC-05: Implement retry logic: up to 3 retries with exponential backoff on send failure
- [ ] AC-06: Persist delivery history to `notification_logs` table (channel, status, error_reason, sent_at)
- [ ] AC-07: Implement `GET /api/v1/notifications/history` — paginated notification history
- [ ] AC-08: Implement alert type `node_connection_loss` — fires when a device has not reported within `2 × measurement_interval_minutes`; include `device_id`, `last_seen_at`, `expected_at` in the alert payload
- [ ] AC-09: Add `maintenance_mode` boolean flag per device — when `true`, suppress all alerts for that device (no notification sent, but log is still written)

**Related Technologies:**
- Notification pattern: Observer / Event Handler
- Template: Go `html/template`
- Concurrent sending: goroutine + `sync.WaitGroup`

**Notes / Dependencies:** Listens to `ThresholdBreachedEvent` from `INV-SPR03-TASK-003`, `AnomalyEvent` from `INV-SPR03-TASK-006`

**Status History:**
| Date       | From | To    | Performed by | Notes                                               |
|------------|------|-------|--------------|-----------------------------------------------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created                                        |
| 2026-04-12 | DRAFT| DRAFT | BA           | AC-08/09 added — customer PDF requirement update    |

---

## [INV-SPR04-TASK-002] — ERP Integration Service

> **Task ID:** `INV-SPR04-TASK-002`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 4  

**Description:** Build a module to export inventory data in customer-specific formats (CSV/TXT Fixed-Width).

**Acceptance Criteria:**
- [ ] AC-01: Design `ExportConfig` per customer: delimiter, encoding, field mapping, date format
- [ ] AC-02: Implement `DataExporter` interface with method `Export(ctx, ExportConfig, DateRange) ([]byte, error)`
- [ ] AC-03: Implement CSV exporter: RFC 4180 compliant, header row is configurable (on/off)
- [ ] AC-04: Implement TXT Fixed-Width exporter: columns aligned to fixed widths
- [ ] AC-05: Exported data includes: SKU, qty, unit, timestamp, device_id, location
- [ ] AC-06: Log every export event to `export_logs` table
- [ ] AC-07: Write unit tests with fixture data (golden file tests) to ensure format stability
- [ ] AC-08: Implement XML exporter — produce well-formed XML with configurable root element name and per-field element name mapping; validate output with `encoding/xml`
- [ ] AC-09: Support `file_naming_template` field in `ExportConfig` accepting tokens: `{YYYYMMDD}`, `{TIMESTAMP}`, `{CUSTOMER_ID}`, `{SKU_CODE}`; default template: `inventory_export_{YYYYMMDD}.{ext}`

**Related Technologies:**
- `encoding/csv`, `encoding/xml`, `golang.org/x/text/encoding` (Windows-1252 support)
- Strategy pattern for multiple formats
- Export uses a data snapshot, not a live query

**Notes / Dependencies:** Depends on `INV-SPR03-TASK-002` (requires inventory snapshot data)

**Status History:**
| Date       | From | To    | Performed by | Notes                                               |
|------------|------|-------|--------------|-----------------------------------------------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created                                        |
| 2026-04-12 | DRAFT| DRAFT | BA           | AC-08/09 added — XML export & file naming from PDF  |

---

## [INV-SPR04-TASK-003] — Scheduled FTP Upload

> **Task ID:** `INV-SPR04-TASK-003`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 4  

**Description:** Automatically push data files to customer FTP servers on a cron schedule, with retry logic and failure alerts.

**Acceptance Criteria:**
- [ ] AC-01: Implement `FTPUploader` service supporting both FTP (`jlaffaye/ftp`) and SFTP (`pkg/sftp`)
- [ ] AC-02: Configure per-customer: `host`, `port`, `user`, `password`, `remote_path` via env/DB
- [ ] AC-03: Scheduler: configure per-customer cron expression using `robfig/cron/v3`
- [ ] AC-04: Automated workflow: generate file → upload → log result
- [ ] AC-05: Retry up to 3 times on failure; if still failing → send email alert to admin
- [ ] AC-06: Implement `POST /api/v1/erp/export/:customer_id/trigger` — manual upload trigger
- [ ] AC-07: Store upload results in `ftp_upload_logs` (status, file_size, duration_ms, error)

**Related Technologies:**
- FTP: `jlaffaye/ftp`, SFTP: `pkg/sftp`
- Scheduler: `robfig/cron/v3`
- Graceful shutdown: drain scheduler before service stops

**Notes / Dependencies:** Depends on `INV-SPR04-TASK-002` (requires generated file)

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR04-TASK-004] — Reorder Workflow

> **Task ID:** `INV-SPR04-TASK-004`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 4  

**Description:** Trigger an automatic Purchase Requisition (PR) workflow when a low-stock threshold event is received.

**Acceptance Criteria:**
- [ ] AC-01: Create `purchase_requisitions` table with: `id`, `sku_code`, `requested_qty`, `triggered_by`, `status` (pending/sent/acknowledged), `created_at`
- [ ] AC-02: On receiving a `ThresholdBreachedEvent` of type `low_stock`: check for no pending PR → create PR → send email
- [ ] AC-03: Calculate `suggested_reorder_qty = reorder_point_qty × 2` or use a configurable multiplier
- [ ] AC-04: Prevent duplicate PRs for the same SKU within `cooldown_hours` (idempotent behavior)
- [ ] AC-05: Implement `GET /api/v1/requisitions` — list PRs by status with pagination
- [ ] AC-06: Implement `PUT /api/v1/requisitions/:id/acknowledge` — mark PR as acknowledged
- [ ] AC-07: Write unit tests to verify idempotency when the workflow is triggered multiple times with the same event

**Related Technologies:**
- State machine for PR lifecycle
- Idempotency key: unique constraint on `(sku_code, cooldown_window)`

**Notes / Dependencies:** Depends on `INV-SPR04-TASK-001` (requires notification service to send PR email)

**Status History:**
| Date       | From | To    | Performed by | Notes        |
|------------|------|-------|--------------|--------------|
| 2026-04-12 | —    | DRAFT | BA           | Task created |

---

## [INV-SPR04-TASK-005] — Node Management & Calibration UI

> **Task ID:** `INV-SPR04-TASK-005`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 4  

**Description:** Build the management web UI for device registry and calibration workflow. Served as static files by the Go backend. Covers screens: Node List, Node Detail/Edit, Calibration form, and Add Node wizard.

**Acceptance Criteria:**
- [ ] AC-01: Implement **Node List screen** — table with columns: device_id, name, location, SKU, status badge, battery_level progress bar, last_seen_at; sortable and filterable
- [ ] AC-02: Implement **Node Detail / Edit screen** — edit name, location, SKU assignment; toggle `maintenance_mode` with confirmation dialog
- [ ] AC-03: Implement **Calibration screen** — display current active calibration values (zero, span, tare, unit); form to submit new calibration; collapsible calibration history log
- [ ] AC-04: Implement **Add Node wizard** — 3-step: (1) enter device_id + metadata, (2) run calibration, (3) confirm and save
- [ ] AC-05: All screens call existing backend REST APIs (`/api/v1/devices`, `/api/v1/devices/:id/calibration`)
- [ ] AC-06: Serve the UI as static files from Go using `embed.FS` at route `/ui/nodes`
- [ ] AC-07: UI displays real-time connection status (online/offline badge) by polling `GET /api/v1/dashboard/summary` every 30 seconds

**Related Technologies:**
- Stack: Vue 3 (Composition API) + Vite build, OR plain HTML + Alpine.js (lightweight)
- `embed.FS` for static file serving in Go
- Styling: Tailwind CSS or minimal custom CSS

**Notes / Dependencies:** Depends on `INV-SPR02-TASK-001`, `INV-SPR02-TASK-002`, `INV-SPR02-TASK-003`

**Status History:**
| Date       | From | To    | Performed by | Notes                              |
|------------|------|-------|--------------|------------------------------------|
| 2026-04-12 | —    | DRAFT | BA           | New task — customer PDF requirement|

---

## [INV-SPR04-TASK-006] — Alerts Center UI

> **Task ID:** `INV-SPR04-TASK-006`  
> **Status:** 📝 DRAFT  
> **Created by:** BA  
> **Created date:** 2026-04-12  
> **Assignee:** —  
> **Sprint:** 4  

**Description:** Build the management web UI for threshold rule management, active alert monitoring, and notification history log. Served as static files by the Go backend.

**Acceptance Criteria:**
- [ ] AC-01: Implement **Threshold Rules screen** — CRUD table for all threshold rules; columns: SKU, rule_type badge, trigger %, cooldown (minutes), enabled toggle; inline edit
- [ ] AC-02: Implement **Active Alerts screen** — list of all unresolved alert events (type, SKU/device, triggered_at, severity badge); with acknowledge button
- [ ] AC-03: Implement **Notification History screen** — log of all sent notifications; columns: channel icon, recipient, subject, delivery_status badge, sent_at; filterable by channel and date
- [ ] AC-04: Implement **Alert Settings screen** — configure per-SKU recipient email list; configure global silencing schedule
- [ ] AC-05: All screens call existing backend APIs (`/api/v1/rules/thresholds`, `/api/v1/notifications/history`)
- [ ] AC-06: Serve UI at `/ui/alerts` via `embed.FS`
- [ ] AC-07: Active Alerts badge count visible in global navigation bar, updated via polling every 30 seconds

**Related Technologies:**
- Same stack as TASK-005 (Vue 3 or Alpine.js)
- Badge/count polling: `setInterval` on dashboard summary endpoint

**Notes / Dependencies:** Depends on `INV-SPR04-TASK-001`, `INV-SPR03-TASK-003`, `INV-SPR03-TASK-006`

**Status History:**
| Date       | From | To    | Performed by | Notes                              |
|------------|------|-------|--------------|------------------------------------|
| 2026-04-12 | —    | DRAFT | BA           | New task — customer PDF requirement|

---

## Definition of Done — Sprint 4

- [ ] Alerts sent successfully via at least 2 channels (Email + Web Push) on threshold breach
- [ ] `node_connection_loss` alert fires correctly when device is silent beyond expected interval
- [ ] ERP CSV, TXT, and XML file exports work correctly end-to-end with correct file naming
- [ ] FTP upload works with the scheduler, retry, and admin alert on failure
- [ ] Reorder workflow does not create duplicate PRs
- [ ] Node Management UI renders correctly and all CRUD operations work via backend API
- [ ] Alerts Center UI shows real-time alert count and history
- [ ] All integration tests pass in the Docker Compose environment
- [ ] All tasks (TASK-001 → TASK-006) reach status 🔒 CLOSED

---

*Managed by: `.agents/rules/golang-ba.md` | Workflow: `docs/workflows/ba_task_creation_workflow.md`*
