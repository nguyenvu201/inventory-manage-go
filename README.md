# Inventory Management System

> IoT-based real-time inventory monitoring using LoRaWAN weight scales.  
> Standards: **FDA 21 CFR Part 11 / IEC 62304**

---

## Architecture Overview

```
[Scale Node (STM32WL + HX711)]
         │  LoRaWAN
         ▼
[LoRaWAN Gateway] → [ChirpStack LNS] → [MQTT: application/+/device/+/event/up]
                                                    │
                                         [Golang Backend Service]
                                         ┌────────────────────────┐
                                         │  Ingestion Layer        │ ← MQTT Worker
                                         │  Rules Engine           │ ← Inventory Calc
                                         │  Action Layer           │ ← Alerts / ERP
                                         │  Management UI          │ ← Vue 3 / Alpine
                                         └────────────────────────┘
                                                    │
                                         [PostgreSQL + TimescaleDB]
```

---

## Prerequisites

| Tool              | Version      | Install |
|-------------------|--------------|---------|
| Go                | 1.22+        | [go.dev/dl](https://go.dev/dl/) |
| Docker            | 24+          | [docker.com](https://docs.docker.com/get-docker/) |
| Docker Compose    | v2           | Included with Docker Desktop |
| golang-migrate    | latest       | `go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |
| staticcheck       | latest       | `go install honnef.co/go/tools/cmd/staticcheck@latest` |

---

## Quick Start

### 1 — Clone and configure

```bash
git clone <repo-url>
cd project_inventory_manage

# Copy the example environment file
cp .env.example .env

# Edit .env with your values (see Configuration Reference below)
nano .env
```

### 2 — Start infrastructure services

```bash
make docker-up

# Wait for services to be healthy (usually ~15 seconds)
docker-compose ps
```

### 3 — Apply database migrations

```bash
make migrate
```

### 4 — Run the service

```bash
# Development (hot-reload via .env)
make run

# Or build and run the binary
make build
./bin/inventory-manage
```

### 5 — Verify health

```bash
curl http://localhost:8080/health
# Expected: {"status":"ok"}
```

---

## Project Structure

```
project_inventory_manage/
├── cmd/
│   └── server/
│       └── main.go             ← Entry point
├── internal/
│   ├── config/                 ← Env-based config loader
│   ├── domain/                 ← Business entities and interfaces
│   │   ├── telemetry/          ← TelemetryPayload, Repository interface
│   │   ├── device/             ← Device, CalibrationConfig
│   │   ├── inventory/          ← InventorySnapshot, SKUConfig
│   │   └── notification/       ← AlertMessage, NotificationSender interface
│   ├── usecase/                ← Business logic (use cases)
│   ├── repository/
│   │   └── postgres/           ← pgx/v5 repository implementations
│   ├── handler/                ← HTTP handlers (chi router)
│   ├── middleware/             ← Auth, logging, trace ID, recovery
│   ├── worker/                 ← MQTT subscriber, cron jobs
│   └── platform/              ← Adapters: SMTP, FTP/SFTP, MQTT
├── pkg/                        ← Shared utilities
├── migrations/                 ← golang-migrate SQL files
├── config/
│   └── mosquitto/              ← Mosquitto MQTT broker config
├── ui/                         ← Management UI static files (embed.FS)
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── .env.example
└── README.md
```

---

## Make Commands

| Command          | Description                                       |
|------------------|---------------------------------------------------|
| `make run`       | Run service locally                               |
| `make build`     | Build binary to `./bin/`                          |
| `make test`      | Run all tests                                     |
| `make test-race` | Run tests with race detector (required before PR) |
| `make test-cover`| Run tests with HTML coverage report               |
| `make lint`      | Run `go vet` + `staticcheck`                      |
| `make migrate`   | Apply all pending migrations                      |
| `make migrate-down` | Roll back last migration                       |
| `make migrate-create NAME=<name>` | Create new migration pair        |
| `make docker-up` | Start all Docker services (detached)              |
| `make docker-down` | Stop all Docker services                        |
| `make docker-logs` | Tail container logs                             |
| `make clean`     | Remove build artifacts                            |

---

## Configuration Reference

All configuration is injected via environment variables. Copy `.env.example` to `.env` and fill in the values.

| Variable           | Required | Default        | Description                     |
|--------------------|----------|----------------|---------------------------------|
| `SERVICE_ENV`      | No       | `development`  | `development` or `production`   |
| `LISTEN_ADDR`      | No       | `:8080`        | HTTP server bind address        |
| `LOG_LEVEL`        | No       | `info`         | `debug`, `info`, `warn`, `error`|
| `DB_HOST`          | **Yes**  | —              | PostgreSQL host                 |
| `DB_PORT`          | No       | `5432`         | PostgreSQL port                 |
| `DB_NAME`          | **Yes**  | —              | Database name                   |
| `DB_USER`          | **Yes**  | —              | Database user                   |
| `DB_PASSWORD`      | **Yes**  | —              | Database password               |
| `DB_SSL_MODE`      | No       | `disable`      | `disable`, `require`, `verify-full` |
| `MQTT_BROKER`      | **Yes**  | —              | MQTT broker hostname            |
| `MQTT_PORT`        | No       | `1883`         | MQTT broker port                |
| `MQTT_CLIENT_ID`   | No       | `inventory-manage` | MQTT client identifier     |
| `MQTT_USERNAME`    | No       | —              | MQTT username (if auth enabled) |
| `MQTT_PASSWORD`    | No       | —              | MQTT password (if auth enabled) |
| `SMTP_HOST`        | No       | —              | SMTP server for email alerts    |
| `SMTP_PORT`        | No       | `587`          | SMTP port (TLS)                 |
| `TWILIO_ACCOUNT_SID` | No   | —              | Twilio SID for SMS alerts       |

---

## Database

- **PostgreSQL 15 + TimescaleDB 2.x**
- Partitioned hypertable: `raw_telemetry` — partitioned by `received_at` (1-day chunks)
- Idempotent ingestion via unique constraint on `(device_id, f_cnt)`
- Migrations managed by `golang-migrate`

---

## FDA Compliance Notes

- All task changes recorded in `docs/sprints/` Status History tables (append-only)
- No raw data is deleted from `raw_telemetry` (immutable audit record)
- Calibration history is append-only (`deactivated_at` soft-deactivation)
- All configuration injected via env vars — no secrets in source control
- Every log entry includes `device_id` and `trace_id` for full traceability

---

## Development Standards

- Error handling: `fmt.Errorf("package.Function: %w", err)` — never swallow errors
- Logging: `zerolog` — mandatory fields: `device_id`, `trace_id`
- Interfaces defined in `internal/domain/` (consumer side)
- Race detector: `go test -race ./...` must pass before any PR
- Test coverage: ≥ 80% for all business logic

---

*For sprint tasks and FDA audit trail, see `docs/sprints/`*
