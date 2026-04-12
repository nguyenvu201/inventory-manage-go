# Manual Testing Guide — INV-SPR01-TASK-001

> Infrastructure setup verification for the Inventory Management System.  
> Run these commands in sequence from your **Terminal** inside the project directory.

---

## 0 · Prerequisites

Open Terminal và chạy lệnh kiểm tra:

```bash
# Chuyển vào thư mục project
cd "/Volumes/SSD_MAC_EXTEND/Inventory Manage/project_inventory_manage"

# Kiểm tra Docker Desktop đang chạy
docker info | grep "Server Version"
# Expected: Server Version: xx.x.x

# Kiểm tra Go (nếu cần chạy service)
go version
# Expected: go version go1.22.x darwin/arm64

# Kiểm tra golang-migrate (cần để chạy migrate)
migrate -version
# Expected: v4.x.x
# Nếu chưa có: go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

> **Lưu ý:** Nếu Docker chưa chạy, mở **Docker Desktop** trước rồi đợi icon ở menu bar ổn định.

---

## 1 · Setup Environment

```bash
# Copy file .env từ example
cp .env.example .env

# Kiểm tra nội dung .env (không cần sửa gì để test local)
cat .env
```

Verify các giá trị default trong `.env`:
```
DB_HOST=localhost
DB_PORT=5432
DB_NAME=inventory_db
DB_USER=inventory_user
DB_PASSWORD=inventory_secret
MQTT_BROKER=localhost
MQTT_PORT=1883
```

---

## 2 · Start Docker Services

```bash
# Khởi động tất cả services (TimescaleDB + Mosquitto)
docker-compose up -d

# Xem trạng thái các containers
docker-compose ps
```

**Expected output:**

```
NAME                IMAGE                           STATUS
inventory_db        timescale/timescaledb:latest-pg15   Up (healthy)
inventory_mqtt      eclipse-mosquitto:2.0               Up (healthy)
inventory_app       project_inventory_manage-app        Up
```

> ⏳ Lần đầu chạy cần 1–2 phút để pull Docker images (~500MB).  
> Các lần sau chạy ngay trong vài giây.

### Kiểm tra health của từng service:

```bash
# PostgreSQL health
docker exec inventory_db pg_isready -U inventory_user -d inventory_db
# Expected: /var/run/postgresql:5432 - accepting connections

# MQTT Mosquitto logs
docker logs inventory_mqtt --tail 10
# Expected: ...mosquitto version 2.0.xx starting, ...Listening on port 1883
```

---

## 3 · Kiểm tra TimescaleDB & Extension

```bash
# Kết nối vào PostgreSQL
docker exec -it inventory_db psql -U inventory_user -d inventory_db

# Trong psql shell, chạy các lệnh sau:
```

```sql
-- 1. Kiểm tra TimescaleDB extension đã cài chưa
\dx timescaledb
-- Expected: timescaledb | x.x.x | public | Enables scalable inserts and complex queries...

-- 2. Kiểm tra danh sách tables (lúc này chưa có, sẽ tạo sau khi migrate)
\dt
-- Expected: Did not find any relations (chưa migrate)

-- 3. Thoát psql
\q
```

---

## 4 · Run Database Migrations

```bash
# Chạy migration (tạo raw_telemetry hypertable)
export DB_URL="postgres://inventory_user:inventory_secret@localhost:5432/inventory_db?sslmode=disable"
migrate -path migrations -database "$DB_URL" up

# Expected output:
# 1/u create_raw_telemetry (Xms)
```

### Verify migration đã tạo đúng table:

```bash
docker exec -it inventory_db psql -U inventory_user -d inventory_db
```

```sql
-- 1. Kiểm tra table raw_telemetry tồn tại
\dt
-- Expected:
--  Schema |     Name      | Type  |     Owner
-- --------+---------------+-------+----------------
--  public | raw_telemetry | table | inventory_user

-- 2. Kiểm tra cấu trúc table
\d raw_telemetry
-- Expected: columns: id, device_id, raw_weight, battery_level, rssi, snr,
--           f_cnt, spreading_factor, sample_count, payload_json,
--           received_at, device_time

-- 3. Kiểm tra hypertable (TimescaleDB)
SELECT hypertable_name, num_chunks
FROM timescaledb_information.hypertables;
-- Expected:
-- hypertable_name | num_chunks
-- ----------------+-----------
-- raw_telemetry   |          0

-- 4. Kiểm tra indexes (bao gồm unique index cho idempotency)
\di raw_telemetry*
-- Expected: uq_raw_telemetry_device_fcnt (unique)
--           idx_raw_telemetry_device_id
--           idx_raw_telemetry_signal

-- 5. Kiểm tra constraint battery_level
SELECT constraint_name, check_clause
FROM information_schema.check_constraints
WHERE constraint_name LIKE '%battery%';
-- Expected: raw_telemetry_battery_level_check | battery_level BETWEEN 0 AND 100

\q
```

---

## 5 · Test MQTT Broker

```bash
# Test publish/subscribe cơ bản với mosquitto_pub/sub
# (Cần mosquitto client: brew install mosquitto)

# Terminal 1 — Subscribe
mosquitto_sub -h localhost -p 1883 -t "test/inventory" -v

# Terminal 2 — Publish
mosquitto_pub -h localhost -p 1883 -t "test/inventory" -m "hello from LoRaWAN"

# Expected in Terminal 1:
# test/inventory hello from LoRaWAN
```

**Nếu không có mosquitto client**, dùng Docker để test:

```bash
# Subscribe trong background
docker exec inventory_mqtt mosquitto_sub -h localhost -p 1883 -t "test/#" &

# Publish
docker exec inventory_mqtt mosquitto_pub -h localhost -p 1883 -t "test/scale" -m '{"device_id":"SCALE-001","raw_weight":5000}'

# Expected: test/scale {"device_id":"SCALE-001","raw_weight":5000}
```

---

## 6 · Run the Go Service

```bash
# Chạy service (cần Go 1.22+ installed)
go run cmd/server/main.go

# Expected console output:
# INF inventory-manage service starting env=development addr=:8080
# INF HTTP server listening addr=:8080
```

### Test health endpoint:

```bash
# Mở terminal mới, test health check
curl -s http://localhost:8080/health
# Expected: {"status":"ok"}

# Verbose mode để xem HTTP headers
curl -v http://localhost:8080/health
# Expected: HTTP/1.1 200 OK
```

---

## 7 · Test Makefile Targets

```bash
# Test từng Makefile target

# Build binary
make build
# Expected: ✓ Build complete: bin/inventory-manage
ls -la bin/
# Expected: bin/inventory-manage (executable file)

# Run tests (hiện tại chưa có test files — sẽ pass với 0 tests)
make test
# Expected: ok inventory-manage/internal/... (hoặc [no test files])

# Lint check
make lint
# Expected: go vet — no output (clean)

# Migrate status
make migrate-status
# Expected: 1 (version 000001 applied)

# Rollback migration (để test down.sql)
make migrate-down
# Expected: 1/d create_raw_telemetry (Xms)

# Apply lại
make migrate
# Expected: 1/u create_raw_telemetry (Xms)
```

---

## 8 · Insert Data Test (SQL)

```bash
docker exec -it inventory_db psql -U inventory_user -d inventory_db
```

```sql
-- Insert một record telemetry hợp lệ
INSERT INTO raw_telemetry (
    device_id, raw_weight, battery_level,
    rssi, snr, f_cnt, spreading_factor, sample_count
) VALUES (
    'SCALE-001', 5000.0, 85,
    -80, 7.5, 1234, 7, 3
);
-- Expected: INSERT 0 1

-- Verify record
SELECT id, device_id, raw_weight, battery_level, f_cnt, received_at
FROM raw_telemetry;

-- Test idempotency: insert record trùng f_cnt → phải bị reject
INSERT INTO raw_telemetry (device_id, raw_weight, battery_level, f_cnt)
VALUES ('SCALE-001', 6000.0, 80, 1234);
-- Expected: ERROR duplicate key value violates unique constraint "uq_raw_telemetry_device_fcnt"

-- Test battery validation: battery_level = 101 → phải bị reject
INSERT INTO raw_telemetry (device_id, raw_weight, battery_level)
VALUES ('SCALE-001', 5000.0, 101);
-- Expected: ERROR new row violates check constraint "raw_telemetry_battery_level_check"

-- Test battery = 0 → phải pass (dead battery, still valid)
INSERT INTO raw_telemetry (device_id, raw_weight, battery_level)
VALUES ('SCALE-001', 5000.0, 0);
-- Expected: INSERT 0 1

\q
```

---

## 9 · Rollback & Cleanup

```bash
# Rollback migration (nếu cần reset)
migrate -path migrations -database "$DB_URL" down 1
# Expected: 1/d create_raw_telemetry

# Stop tất cả containers nhưng giữ data volumes
docker-compose stop

# Stop và XÓA containers + volumes (full reset)
docker-compose down -v
# Expected: Removing containers and volumes

# Remove compiled binary
make clean
# Expected: ✓ Clean complete
```

---

## 10 · Checklist Summary

```
[ ] Step 0: Docker running, Go installed
[ ] Step 1: .env file created from .env.example
[ ] Step 2: docker-compose up -d → all containers healthy
[ ] Step 3: TimescaleDB extension present in psql
[ ] Step 4: migrate up → raw_telemetry hypertable created
            → uq_raw_telemetry_device_fcnt unique index exists
            → battery_level CHECK constraint enforced
[ ] Step 5: MQTT broker accepts publish/subscribe on port 1883
[ ] Step 6: go run → service starts, /health returns {"status":"ok"}
[ ] Step 7: make build, make test, make lint all pass
[ ] Step 8: Duplicate f_cnt insert → REJECTED by DB ✓
            battery_level=101 → REJECTED by CHECK constraint ✓
            battery_level=0 → ACCEPTED ✓
[ ] Step 9: Cleanup works (docker-compose down, make clean)
```

---

## Troubleshooting

| Problem | Solution |
|---------|---------|
| `docker-compose: command not found` | Dùng `docker compose` (không có gạch ngang) |
| `inventory_db` container không healthy | `docker logs inventory_db` để xem lỗi |
| `migrate: command not found` | `go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |
| Port 5432 already in use | `lsof -i :5432` → stop local postgres trước |
| Port 1883 already in use | `lsof -i :1883` → stop local mosquitto trước |
| `go: command not found` | Cài Go từ [go.dev/dl](https://go.dev/dl/) hoặc `brew install go` |
| Service crash ngay sau khi start | Kiểm tra `.env` có đủ required fields: `DB_HOST`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `MQTT_BROKER` |
