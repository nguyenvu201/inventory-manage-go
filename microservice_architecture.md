# Kiến trúc Microservice — Inventory Management System
> Tài liệu giải thích trực quan từng thành phần  
> Project: IoT Scale Inventory | Golang | FDA 21 CFR Part 11

---

## 1. Bức tranh tổng thể

```mermaid
graph TB
    subgraph Physical["🏭 Tầng Vật Lý (Hardware)"]
        SCALE["⚖️ IoT Scale<br/>(Cân điện tử)"]
        GW["📡 LoRaWAN Gateway<br/>(ChirpStack LNS)"]
    end

    subgraph Edge["🌐 Tầng Mạng & Gateway"]
        MQTT["🟢 MQTT Broker<br/>Eclipse Mosquitto :1883"]
        TRAEFIK["🔀 Traefik API Gateway<br/>:80 (HTTP) | :8081 (Dashboard)"]
    end

    subgraph Services["⚙️ Tầng Microservices"]
        MS1["📥 ms-ingestion<br/>:8081<br/>Nhận & lưu dữ liệu thô"]
        MS2["🧮 ms-inventory-core<br/>:8082<br/>Business Rules Engine"]
        MS3["🔔 ms-action<br/>:8083<br/>Alert & ERP Integration"]
        MSUI["🖥️ ms-ui<br/>:80<br/>Management Dashboard"]
    end

    subgraph Storage["💾 Tầng Lưu Trữ"]
        DB[("🐘 PostgreSQL 15<br/>+ TimescaleDB")]
        REDIS[("⚡ Redis 7<br/>Cache & Pub/Sub")]
    end

    subgraph External["📤 Hệ thống bên ngoài"]
        ERP["🏢 ERP System<br/>(FTP/SFTP)"]
        EMAIL["📧 Email SMTP<br/>(gomail)"]
        SMS["📱 Twilio SMS"]
        ADMIN["👤 Admin / Operator"]
    end

    SCALE -->|"LoRaWAN RF"| GW
    GW -->|"MQTT publish<br/>topic: lorawan/+/up"| MQTT
    MQTT -->|"Subscribe"| MS1
    
    ADMIN -->|"HTTP Request"| TRAEFIK
    TRAEFIK -->|"/api/v1/devices/*<br/>/api/v1/calibration/*"| MS1
    TRAEFIK -->|"/api/v1/inventory/*<br/>/api/v1/rules/*<br/>/api/v1/reports/*"| MS2
    TRAEFIK -->|"/api/v1/notifications/*<br/>/api/v1/erp/*<br/>/api/v1/requisitions/*"| MS3
    TRAEFIK -->|"/ui/*"| MSUI

    MS1 -->|"ThresholdBreachedEvent<br/>InventoryUpdatedEvent"| REDIS
    MS2 -->|"Subscribe events"| REDIS
    MS3 -->|"Subscribe events"| REDIS

    MS1 --- DB
    MS2 --- DB
    MS3 --- DB

    MS2 -->|"Cache inventory<br/>snapshots"| REDIS
    
    MS3 -->|"Upload file CSV/XML"| ERP
    MS3 -->|"Send alert"| EMAIL
    MS3 -->|"Send SMS"| SMS

    MSUI -->|"REST API calls"| TRAEFIK
```

---

## 2. Giải thích từng thành phần

---

### 🔀 Traefik — API Gateway (Cổng vào duy nhất)

```mermaid
graph LR
    CLIENT["👤 Client<br/>(Browser / App)"]
    
    subgraph TRAEFIK["🔀 Traefik API Gateway :80"]
        ROUTER["Router Rules<br/>(path prefix matching)"]
        MW["Middlewares<br/>• Rate Limiting<br/>• CORS<br/>• Auth Header Check<br/>• Request ID inject"]
        LB["Load Balancer<br/>(round-robin)"]
    end

    MS1["ms-ingestion :8081"]
    MS2["ms-inventory-core :8082"]
    MS3["ms-action :8083"]
    UI["ms-ui :80"]

    CLIENT -->|"HTTP :80"| ROUTER
    ROUTER -->|"PathPrefix: /api/v1/devices<br/>/api/v1/calibration"| MW
    ROUTER -->|"PathPrefix: /api/v1/inventory<br/>/api/v1/rules<br/>/api/v1/reports"| MW
    ROUTER -->|"PathPrefix: /api/v1/erp<br/>/api/v1/notifications"| MW
    ROUTER -->|"PathPrefix: /ui"| UI
    MW --> LB
    LB --> MS1
    LB --> MS2
    LB --> MS3
```

**Traefik là gì?**  
Giống như **lễ tân của một tòa nhà** — tất cả khách (request) đều vào qua 1 cửa duy nhất. Traefik sẽ:
1. Kiểm tra request (authentication, rate limiting)
2. Inject thêm thông tin (`X-Request-ID`, `X-Trace-ID`)
3. Chuyển đến đúng service phía sau dựa trên URL path

**Config hiện tại** (bạn đã có `traefik/traefik.yml` và `traefik/dynamic.yml`):
- `:80` → HTTP entrypoint (route requests)
- `:8081` → Traefik dashboard (monitoring)

---

### 📥 ms-ingestion — Tầng Nhận Dữ Liệu

```mermaid
graph TB
    subgraph MS1["📥 ms-ingestion (Sprint 1 + 2)"]
        direction LR
        subgraph WORKERS["Background Workers"]
            MQTTW["🔌 MQTT Receiver<br/>telemetry_receiver.go<br/>Subscribe lorawan/#"]
            STORW["💾 Storage Worker<br/>storage_worker.go<br/>Batch insert raw_telemetry"]
        end

        subgraph DOMAIN["Domain Logic"]
            VALID["✅ Telemetry Validator<br/>- Kiểm tra battery 0-100<br/>- device_id không rỗng<br/>- raw_weight ≥ 0<br/>- Dedup (device_id, f_cnt)"]
            PARSER["🔤 Payload Parser<br/>- Decode LoRaWAN bytes<br/>- Map to RawTelemetry struct"]
        end

        subgraph API["REST API (Gin)"]
            DEVAPI["Device Registry API<br/>POST /api/v1/devices<br/>GET /api/v1/devices/:id<br/>PUT /api/v1/devices/:id"]
            CALAPI["Calibration API<br/>POST /api/v1/calibration<br/>GET /api/v1/calibration/:device_id<br/>PUT /api/v1/calibration/:device_id"]
        end
    end

    MQTT["🟢 MQTT Broker"] -->|"JSON payload"| MQTTW
    MQTTW --> PARSER
    PARSER --> VALID
    VALID -->|"Valid"| STORW
    VALID -->|"Invalid → log"| LOG["📋 Log warning<br/>(zap logger)"]
    STORW -->|"INSERT raw_telemetry"| DB[("🐘 TimescaleDB")]
    STORW -->|"Publish InventoryUpdatedEvent"| REDIS[("⚡ Redis")]

    TRAEFIK["🔀 Traefik"] -->|"HTTP"| DEVAPI
    TRAEFIK -->|"HTTP"| CALAPI
    DEVAPI --- DB
    CALAPI --- DB
```

**ms-ingestion làm gì?**

| Trách nhiệm | Chi tiết |
|-------------|---------|
| **Nhận dữ liệu từ cân** | Subscribe MQTT topic `lorawan/+/up`, decode JSON payload |
| **Validate** | Battery 0-100, device_id không rỗng, raw_weight ≥ 0, loại bỏ duplicate |
| **Lưu trữ** | Batch insert vào TimescaleDB hypertable `raw_telemetry` |
| **Quản lý thiết bị** | CRUD devices, CRUD calibration config, audit trail |
| **Publish event** | Sau mỗi telemetry hợp lệ → publish `InventoryUpdatedEvent` |

**Tables owned:**
- `raw_telemetry` (hypertable — time-series)
- `devices`
- `calibration_configs`
- `calibration_audit_logs`

---

### 🧮 ms-inventory-core — Business Rules Engine

```mermaid
graph TB
    subgraph MS2["🧮 ms-inventory-core (Sprint 3)"]
        subgraph DOMAIN["Domain Logic"]
            CONV["⚖️ Weight Conversion<br/>internal/domain/inventory/<br/>net_weight = (raw - zero - tare) × span<br/>qty = net_weight / unit_weight"]
            THRESH["📏 Threshold Evaluator<br/>So sánh qty với threshold rules<br/>Phát ThresholdBreachedEvent"]
            FORE["🔮 Forecasting<br/>Day Zero prediction<br/>Dựa trên consumption rate"]
            ANOM["🚨 Anomaly Detector<br/>Z-score outlier detection<br/>±3σ rolling window"]
        end

        subgraph API["REST API (Gin)"]
            INVAPI["Inventory API<br/>GET /api/v1/inventory/snapshots<br/>GET /api/v1/inventory/:sku/forecast"]
            RULAPI["Threshold Rules API<br/>CRUD /api/v1/rules/thresholds"]
            REPAPI["Report API<br/>GET /api/v1/reports/consumption<br/>GET /api/v1/reports/summary"]
        end
    end

    REDIS_IN["⚡ Redis<br/>InventoryUpdatedEvent"] -->|"Subscribe"| CONV
    CONV -->|"Upsert snapshot"| DB[("🐘 TimescaleDB")]
    CONV --> THRESH
    THRESH -->|"Breach detected"| PUB["📤 Publish<br/>ThresholdBreachedEvent"]
    PUB --> REDIS_OUT["⚡ Redis Pub/Sub"]
    
    REDIS_CACHE["⚡ Redis Cache"] <-->|"Cache snapshots<br/>(TTL 60s)"| INVAPI
    
    INVAPI --- DB
    RULAPI --- DB
    REPAPI --- DB

    TRAEFIK["🔀 Traefik"] --> INVAPI
    TRAEFIK --> RULAPI
    TRAEFIK --> REPAPI
```

**ms-inventory-core làm gì?**

| Trách nhiệm | Chi tiết |
|-------------|---------|
| **Weight Conversion** | `net_weight = (raw_weight - zero_offset - tare) × span_factor` |
| **Inventory Calculation** | `qty = net_weight_kg / unit_weight_kg` → upsert `inventory_snapshots` |
| **Threshold Evaluation** | Kiểm tra qty với rules → publish event nếu breach |
| **Forecasting (Sprint 3)** | Tính `days_remaining` và `day_zero_date` dựa trên consumption trend |
| **Anomaly Detection (Sprint 3)** | Z-score cho từng device, đánh dấu `suspect=true` |
| **Reporting API** | Query consumption trend từ `inventory_history` (TimescaleDB) |

**Tables owned:**
- `sku_configs`
- `inventory_snapshots`
- `inventory_history`
- `threshold_rules`

---

### 🔔 ms-action — Alert & ERP Integration Layer

```mermaid
graph TB
    subgraph MS3["🔔 ms-action (Sprint 4)"]
        subgraph LISTENERS["Event Listeners"]
            L1["👂 ThresholdBreachedEvent listener"]
            L2["👂 AnomalyEvent listener"]
            L3["👂 NodeConnectionLossEvent listener"]
        end

        subgraph ALERT["Alert Service"]
            EMAIL["📧 Email Sender<br/>SMTP + gomail.v2<br/>HTML templates"]
            SMS["📱 SMS Sender<br/>Twilio API"]
            PUSH["💻 Web Push<br/>SSE / WebSocket"]
            RETRY["🔁 Retry Logic<br/>3 retries + exponential backoff"]
        end

        subgraph ERP["ERP Integration"]
            CSV["📄 CSV Exporter<br/>RFC 4180 compliant"]
            XML["📋 XML Exporter<br/>encoding/xml"]
            TXT["📝 TXT Fixed-Width<br/>Column-aligned"]
            FTP["📤 FTP/SFTP Uploader<br/>jlaffaye/ftp + pkg/sftp"]
            CRON["⏰ Cron Scheduler<br/>robfig/cron/v3<br/>Per-customer schedule"]
        end

        subgraph REORDER["Reorder Workflow"]
            PR["📋 Purchase Requisition<br/>Idempotent creation<br/>Cooldown window check"]
        end

        subgraph RAPI["REST API"]
            NAPI["GET /api/v1/notifications/history"]
            EAPI["POST /api/v1/erp/export/:customer_id/trigger"]
            RQAPI["GET /api/v1/requisitions"]
        end
    end

    REDIS["⚡ Redis Pub/Sub"] -->|"ThresholdBreachedEvent"| L1
    REDIS -->|"AnomalyEvent"| L2
    REDIS -->|"NodeConnectionLossEvent"| L3

    L1 --> EMAIL
    L1 --> PR
    L2 --> EMAIL
    L3 --> EMAIL

    EMAIL --> RETRY
    SMS --> RETRY
    PUSH --> RETRY
    RETRY -->|"Log result"| DB[("🐘 notification_logs")]

    CSV --> FTP
    XML --> FTP
    TXT --> FTP
    CRON -->|"Trigger export"| CSV
    FTP -->|"Log result"| DB2[("🐘 ftp_upload_logs")]
    FTP -->|"Upload"| EXT["🏢 ERP FTP Server"]

    PR --- DB3[("🐘 purchase_requisitions")]
    TRAEFIK["🔀 Traefik"] --> NAPI
    TRAEFIK --> EAPI
    TRAEFIK --> RQAPI
```

**ms-action làm gì?**

| Trách nhiệm | Chi tiết |
|-------------|---------|
| **Alert Service** | Nhận events → gửi Email/SMS/Push; retry 3 lần; log kết quả |
| **ERP Export** | Tạo file CSV/TXT/XML theo config từng khách hàng |
| **FTP Upload** | Schedule upload tự động theo cron; manual trigger qua API |
| **Reorder Workflow** | Tạo Purchase Requisition khi low_stock; idempotent (cooldown) |

**Đặc điểm quan trọng:**  
`ms-action` **không pull dữ liệu trực tiếp** từ ms-inventory-core. Nó **lắng nghe events** từ Redis Pub/Sub. Đây là **loose coupling** — ms-action không biết ms-inventory-core tồn tại.

**Tables owned:**
- `notification_logs`
- `export_logs`
- `ftp_upload_logs`
- `purchase_requisitions`

---

### 🖥️ ms-ui — Management Dashboard

```mermaid
graph TB
    subgraph MSUI["🖥️ ms-ui — Vue 3 + Vite"]
        subgraph PAGES["Pages (Vue Router)"]
            N1["/ui/nodes → Node List"]
            N2["/ui/nodes/:id → Node Detail"]
            N3["/ui/nodes/:id/calibration → Calibration"]
            N4["/ui/alerts → Alerts Center"]
            N5["/ui/analytics → Analytics Hub"]
        end

        subgraph COMPONENTS["Key Components"]
            CHART["📊 Chart.js<br/>Consumption trend<br/>Battery gauge<br/>SKU comparison"]
            TABLE["📋 DataTable<br/>Sortable, filterable<br/>Inline edit"]
            BADGE["🏷️ Badge Components<br/>Status, severity, delivery"]
            POLL["⏱️ Polling Service<br/>setInterval 30s<br/>Active alerts count"]
        end

        subgraph BUILD["Serve Strategy"]
            NGINX["🌐 Nginx<br/>(Standalone container)<br/>vs.<br/>embed.FS<br/>(Go binary)"]
        end
    end

    subgraph API_CALLS["API Calls (qua Traefik)"]
        D_API["GET /api/v1/devices"]
        I_API["GET /api/v1/inventory/snapshots"]
        T_API["GET /api/v1/rules/thresholds"]
        N_API["GET /api/v1/notifications/history"]
        R_API["GET /api/v1/reports/consumption"]
    end

    N1 --> D_API
    N2 --> D_API
    N4 --> T_API
    N4 --> N_API
    N5 --> I_API
    N5 --> R_API
    POLL --> D_API

    D_API -->|"qua Traefik"| MS1["ms-ingestion"]
    I_API -->|"qua Traefik"| MS2["ms-inventory-core"]
    T_API -->|"qua Traefik"| MS2
    N_API -->|"qua Traefik"| MS3["ms-action"]
    R_API -->|"qua Traefik"| MS2
```

**ms-ui làm gì?**

| Screen | Nội dung |
|--------|---------|
| **Node List** | Bảng thiết bị: status badge, battery bar, last_seen; sort/filter |
| **Node Detail** | Edit metadata, toggle `maintenance_mode`, lịch sử calibration |
| **Calibration Screen** | Xem/cập nhật zero, span, tare, unit; form submit new calibration |
| **Alerts Center** | Threshold rules CRUD; active alerts; notification history log |
| **Analytics Hub** | Consumption trend chart; Day Zero prediction; Anomaly log; RSSI/SNR |

---

### 💾 Shared Infrastructure

```mermaid
graph LR
    subgraph INFRA["💾 Shared Infrastructure"]
        subgraph PG["🐘 PostgreSQL 15 + TimescaleDB"]
            direction TB
            S1["Schema: ingestion<br/>raw_telemetry (hypertable)<br/>devices<br/>calibration_configs<br/>calibration_audit_logs"]
            S2["Schema: inventory<br/>sku_configs<br/>inventory_snapshots<br/>inventory_history<br/>threshold_rules"]
            S3["Schema: action<br/>notification_logs<br/>export_logs<br/>ftp_upload_logs<br/>purchase_requisitions"]
        end

        subgraph REDIS["⚡ Redis 7"]
            direction TB
            CACHE["Cache Layer<br/>inventory:snapshot:{sku}<br/>TTL 60s"]
            PUBSUB["Pub/Sub Channel<br/>events:inventory:updated<br/>events:threshold:breached<br/>events:anomaly:detected<br/>events:node:lost"]
        end

        subgraph MQTT["🟢 MQTT Mosquitto"]
            direction TB
            TOPICS["Topics<br/>lorawan/{device_id}/up<br/>lorawan/{device_id}/down"]
        end
    end

    MS1["ms-ingestion"] -->|"owns"| S1
    MS2["ms-inventory-core"] -->|"owns"| S2
    MS3["ms-action"] -->|"owns"| S3

    MS2 <-->|"read/write"| CACHE
    MS1 -->|"publish"| PUBSUB
    MS2 -->|"publish"| PUBSUB
    MS3 -->|"subscribe"| PUBSUB

    MS1 -->|"subscribe"| TOPICS
```

---

## 3. Event Flow — Luồng dữ liệu end-to-end

```mermaid
sequenceDiagram
    participant SCALE as ⚖️ IoT Scale
    participant GW as 📡 LoRaWAN Gateway
    participant MQTT as 🟢 MQTT Broker
    participant ING as 📥 ms-ingestion
    participant REDIS as ⚡ Redis
    participant CORE as 🧮 ms-inventory-core
    participant ACTION as 🔔 ms-action
    participant ERP as 🏢 ERP System

    Note over SCALE,ERP: 🟢 Happy Path — Dữ liệu bình thường
    SCALE->>GW: RF signal (raw weight: 45.2kg)
    GW->>MQTT: Publish lorawan/SCALE-001/up (JSON payload)
    MQTT->>ING: Deliver message
    ING->>ING: Validate (battery OK, weight ≥ 0, not duplicate)
    ING->>ING: Parse payload → RawTelemetry struct
    ING->>ING: Save to raw_telemetry (TimescaleDB)
    ING->>REDIS: PUBLISH events:inventory:updated {device_id, raw_weight}

    REDIS->>CORE: Deliver InventoryUpdatedEvent
    CORE->>CORE: Load calibration config (zero=0.3, span=1.02, tare=2.1)
    CORE->>CORE: net_weight = (45.2 - 0.3 - 2.1) × 1.02 = 43.67 kg
    CORE->>CORE: qty = 43.67 / 25 = 1.747 bags
    CORE->>CORE: Upsert inventory_snapshots
    CORE->>CORE: Evaluate threshold rules → qty < 2.0 (low_stock)!
    CORE->>REDIS: PUBLISH events:threshold:breached {sku, qty, rule_type: "low_stock"}

    REDIS->>ACTION: Deliver ThresholdBreachedEvent
    ACTION->>ACTION: Check: no pending PR for SKU-A in last 24h
    ACTION->>ACTION: Create purchase_requisition (status: pending)
    ACTION->>ACTION: Send email alert to procurement@company.com
    ACTION->>ACTION: Generate CSV export file
    ACTION->>ERP: SFTP upload inventory_export_20260415.csv

    Note over SCALE,ERP: 🔴 Error Path — Thiết bị không xác định
    SCALE->>MQTT: Publish lorawan/UNKNOWN-999/up
    MQTT->>ING: Deliver message
    ING->>ING: device_id not in devices table
    ING->>ING: Log to unknown_devices queue
    ING->>ING: Increment unknown_device_count
    Note right of ING: Nếu > 10 lần → publish alert event
```

---

## 4. Docker Compose Topology

```mermaid
graph TB
    subgraph DOCKER["🐳 Docker Compose — inventory_net"]
        subgraph INFRA_LAYER["Infrastructure Layer (luôn chạy)"]
            DB["🐘 db<br/>timescale/timescaledb:latest-pg15<br/>:5432 → :5432"]
            RD["⚡ redis<br/>redis:7-alpine<br/>:6379 → :6379"]
            MQ["🟢 mosquitto<br/>eclipse-mosquitto:2.0<br/>:1883 → :1883"]
            TR["🔀 traefik<br/>traefik:v2.11<br/>:80 → :80 | :8081 → :8081"]
        end

        subgraph APP_LAYER["Application Layer (profile: app)"]
            A1["📥 ms-ingestion<br/>Build: ./services/ms-ingestion<br/>expose: 8081 (không expose ra host)"]
            A2["🧮 ms-inventory-core<br/>Build: ./services/ms-inventory-core<br/>expose: 8082"]
            A3["🔔 ms-action<br/>Build: ./services/ms-action<br/>expose: 8083"]
            UI["🖥️ ms-ui<br/>nginx:alpine<br/>serve static dist/"]
        end
    end

    DB -->|"healthcheck: pg_isready"| A1
    DB -->|"healthcheck"| A2
    DB -->|"healthcheck"| A3
    RD -->|"healthcheck: redis-cli ping"| A1
    RD --> A2
    RD --> A3
    MQ -->|"healthcheck"| A1
    TR -->|"healthcheck: traefik healthcheck --ping"| A1

    TR -.->|"Docker label routing"| A1
    TR -.->|"Docker label routing"| A2
    TR -.->|"Docker label routing"| A3
    TR -.->|"Docker label routing"| UI
```

**Cách dùng:**
```bash
# Chỉ infra (DB + MQTT + Redis + Traefik)
docker compose up -d

# Tất cả services
docker compose --profile app up

# Xem dashboard Traefik
open http://localhost:8081
```

---

## 5. Ranh giới trách nhiệm (Ownership Boundary)

```mermaid
graph LR
    subgraph BOUNDARY["Domain Ownership"]
        subgraph ING_DOMAIN["📥 ms-ingestion domain"]
            D1["devices"]
            D2["calibration_configs"]
            D3["calibration_audit_logs"]
            D4["raw_telemetry"]
        end

        subgraph CORE_DOMAIN["🧮 ms-inventory-core domain"]
            D5["sku_configs"]
            D6["inventory_snapshots"]
            D7["inventory_history"]
            D8["threshold_rules"]
        end

        subgraph ACTION_DOMAIN["🔔 ms-action domain"]
            D9["notification_logs"]
            D10["export_logs"]
            D11["ftp_upload_logs"]
            D12["purchase_requisitions"]
        end
    end

    RULE1["❌ ms-action KHÔNG truy cập<br/>trực tiếp inventory_snapshots<br/>→ chỉ qua event"]
    RULE2["❌ ms-inventory-core KHÔNG biết<br/>đến notification_logs<br/>→ chỉ publish event"]
    RULE3["✅ Mỗi service chỉ<br/>write vào tables của mình"]
```

---

## 6. Tóm tắt — Mỗi service một câu

| Service | Câu mô tả ngắn gọn |
|---------|-------------------|
| **Traefik** | "Người bảo vệ cửa" — nhận tất cả request từ bên ngoài, kiểm tra, rồi chuyển đến đúng service |
| **ms-ingestion** | "Người nhận hàng" — nhận tín hiệu từ cân, validate, lưu vào database thô, quản lý thiết bị |
| **ms-inventory-core** | "Kế toán trưởng" — tính toán tồn kho thực tế từ số liệu thô, áp dụng business rules, dự báo |
| **ms-action** | "Người phản ứng" — nhận tín hiệu cảnh báo, gửi email/SMS, xuất file ERP, tạo đơn mua hàng |
| **ms-ui** | "Mặt tiền cửa hàng" — dashboard web cho operator nhìn vào hệ thống và điều chỉnh |
| **PostgreSQL + TimescaleDB** | "Kho lưu trữ" — lưu tất cả dữ liệu, time-series cho raw_telemetry |
| **Redis** | "Bảng thông báo + bộ nhớ nhanh" — truyền events giữa services, cache snapshot tồn kho |
| **MQTT Broker** | "Trạm phát thanh" — nhận tín hiệu từ cân IoT qua mạng LoRaWAN, broadcast cho subscriber |

---

*Tài liệu này mang tính educational — không thay đổi code. Sprint tasks sẽ được tạo sau khi kiến trúc được approve.*
