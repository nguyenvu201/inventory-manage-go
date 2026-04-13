# Golang BA — Sprint Overview

> Overall plan for 5 sprints of the Inventory Management System based on IoT scales.

---

## System Architecture

```
[Scale Sensor] → [Gateway / ChirpStack] → [MQTT Broker]
                                                ↓
                                    [Golang Backend Service]
                                    ┌──────────────────────────┐
                                    │  Ingestion Layer         │ ← Sprint 1
                                    │  Device & Calibration    │ ← Sprint 2
                                    │  Inventory Rules Engine  │ ← Sprint 3
                                    │  Action / ERP Layer      │ ← Sprint 4
                                    │  Fail-safe & Optimize    │ ← Sprint 5
                                    └──────────────────────────┘
                                                ↓
                               [PostgreSQL + TimescaleDB]
                                                ↓
                               [Dashboard] ←→ [ERP via FTP]
```

---

## Sprint List

| Sprint | Name                                         | Primary Goal                                           | Task File                                              |
|--------|----------------------------------------------|--------------------------------------------------------|--------------------------------------------------------|
| 1      | Infrastructure & Data Ingestion              | Receive and store telemetry messages from Gateway      | [sprint_01](./sprint_01_infrastructure_ingestion.md)   |
| 2      | Device Management & Calibration              | Manage device registry and calibration workflow        | [sprint_02](./sprint_02_device_calibration.md)         |
| 3      | Inventory Rules Engine + Dashboard API       | Weight conversion, business rules, Day Zero, anomaly, dashboard API | [sprint_03](./sprint_03_inventory_rules_engine.md) |
| 4      | Action, ERP Integration & Management UI      | Alerts, ERP export, FTP, reorder, Node UI, Alerts UI  | [sprint_04](./sprint_04_action_erp_integration.md)     |
| 5      | Optimization, Fail-safe & Analytics UI       | Fail-safe, optimization, E2E testing, Analytics Hub UI | [sprint_05](./sprint_05_optimization_failsafe.md)     |

---

## Overall Progress

| Sprint | Total Tasks | Completed | Status            | Notes                          |
|--------|-------------|-----------|-------------------|--------------------------------|
| 1      | 4           | 4         | ✅ Completed       | TASK-004 closed                |
| 2      | 4           | 4         | ✅ Completed       | All tasks (001 - 004) closed   |
| 3      | 7           | 1         | 🔄 In Progress     | TASK-001 closed; TASK-002 submitted for review |
| 4      | 6           | 0         | 🔲 Not Started     | +2 new UI tasks (PDF Rev 2)    |
| 5      | 5           | 0         | 🔲 Not Started     | +1 new UI task (PDF Rev 2)     |

---

## Technology Stack

| Layer         | Technology                                         |
|---------------|----------------------------------------------------|
| Language      | Golang 1.22+                                       |
| Database      | PostgreSQL 15 + TimescaleDB 2.x                    |
| Message Queue | MQTT (Eclipse Mosquitto) / ChirpStack (LoRaWAN LNS)|
| HTTP Router   | `chi` or `gin`                                     |
| ORM / DB      | `pgx/v5`, `squirrel`, `golang-migrate`             |
| Scheduler     | `robfig/cron/v3`                                   |
| Notification  | SMTP (gomail), Twilio SMS, WebSocket/SSE           |
| FTP/SFTP      | `jlaffaye/ftp`, `pkg/sftp`                         |
| Testing       | `testify`, `testcontainers-go`                     |
| Infrastructure| Docker, Docker Compose                             |
| Management UI | Vue 3 + Vite (or Alpine.js) + Chart.js; served via Go `embed.FS` |
| Real-time     | SSE (`text/event-stream`) for live inventory push  |

---

*Last updated: 2026-04-12*
