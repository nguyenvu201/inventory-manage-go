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

| Sprint | Name                                         | Primary Goal                                                        | Task File                                                    |
|--------|----------------------------------------------|---------------------------------------------------------------------|--------------------------------------------------------------|
| 1      | Infrastructure & Data Ingestion              | Receive and store telemetry messages from Gateway                   | [sprint_01](./sprint_01_infrastructure_ingestion.md)         |
| 2      | Device Management & Calibration              | Manage device registry and calibration workflow                     | [sprint_02](./sprint_02_device_calibration.md)               |
| 3      | Inventory Rules Engine + Dashboard API       | Weight conversion, business rules, Day Zero, anomaly, dashboard API | [sprint_03](./sprint_03_inventory_rules_engine.md)           |
| 4      | Action, ERP Integration & Management UI      | Alerts, ERP export, FTP, reorder, Node UI, Alerts UI               | [sprint_04](./sprint_04_action_erp_integration.md)           |
| 5      | Optimization, Fail-safe & Analytics UI       | Fail-safe, optimization, E2E testing, Analytics Hub UI              | [sprint_05](./sprint_05_optimization_failsafe.md)            |
| **6**  | **Firebase & Cloud Deployment** 🆕           | **Deploy lên cloud — khách hàng preview sản phẩm qua URL công khai** | [sprint_06](./sprint_06_firebase_deployment.md)            |

---

## Overall Progress

| Sprint | Total Tasks | Completed | Status               | Notes                                                    |
|--------|-------------|-----------|----------------------|----------------------------------------------------------|
| 1      | 4           | 4         | ✅ Completed          | TASK-004 closed                                          |
| 2      | 4           | 4         | ✅ Completed          | All tasks (001 - 004) closed                             |
| 3      | 7           | 2         | 🔄 In Progress        | TASK-001 and TASK-002 closed; TASK-003 ready to start    |
| 4      | 6           | 0         | 🔲 Not Started        | +2 new UI tasks (PDF Rev 2)                              |
| 5      | 5           | 0         | 🔲 Not Started        | +1 new UI task (PDF Rev 2)                               |
| **6**  | **9**       | **0**     | 🔲 **Not Started** 🆕 | **CRITICAL — customer preview. Chạy song song Sprint 3–5** |

---

## Technology Stack

| Layer              | Local Dev                                          | Cloud (Sprint 6)                          |
|--------------------|----------------------------------------------------|--------------------------------------------||
| Language           | Golang 1.22+                                       | Golang 1.22+                              |
| Database           | PostgreSQL 15 + TimescaleDB 2.x (Docker)           | Google Cloud SQL for PostgreSQL 15        |
| Message Queue      | MQTT (Eclipse Mosquitto) / ChirpStack (LoRaWAN LNS) | HiveMQ Cloud (MQTT over TLS)             |
| Cache              | Redis 7 (Docker)                                   | Upstash Redis / Cloud Memorystore         |
| HTTP Router        | `chi` or `gin`                                     | Same                                      |
| ORM / DB           | `pgx/v5`, `squirrel`, `golang-migrate`             | Same + Cloud SQL Connector                |
| Scheduler          | `robfig/cron/v3`                                   | Same                                      |
| Notification       | SMTP (gomail), Twilio SMS, WebSocket/SSE           | Same                                      |
| FTP/SFTP           | `jlaffaye/ftp`, `pkg/sftp`                         | Same                                      |
| Testing            | `testify`, `testcontainers-go`                     | GitHub Actions CI                         |
| Infrastructure     | Docker, Docker Compose                             | Google Cloud Run + Firebase Hosting       |
| Container Registry | Local                                              | Google Artifact Registry                  |
| Secrets            | `.env` file                                        | Google Secret Manager                     |
| CI/CD              | Manual                                             | GitHub Actions                            |
| Management UI      | Vue 3 + Vite (or Alpine.js) + Chart.js; via `embed.FS` | Firebase Hosting (static CDN)        |
| Real-time          | SSE (`text/event-stream`) for live inventory push  | Same (via Cloud Run)                      |

---

*Last updated: 2026-04-19 — Sprint 6 (Firebase & Cloud Deployment) added*
