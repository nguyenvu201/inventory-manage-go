#!/bin/bash
# mqtt_test.sh — Test MQTT broker và ingestion pipeline
# Usage:
#   chmod +x mqtt_test.sh
#   ./mqtt_test.sh              # local Mosquitto (localhost:1883)
#   ./mqtt_test.sh happy        # chỉ test happy path
#
# HiveMQ Cloud:
#   MQTT_BROKER=29ae9d12a8c54f1295a5d2e1e4391c6c.s1.eu.hivemq.cloud \
#   MQTT_PORT=8883 \
#   MQTT_TLS=true \
#   MQTT_USER=your_username \
#   MQTT_PASS=your_password \
#   ./mqtt_test.sh happy

BROKER=${MQTT_BROKER:-localhost}
PORT=${MQTT_PORT:-1883}
TOPIC="application/1/device/aabbccdd11223344/event/up"
DEVICE_EUI="aabbccdd11223344"
MQTT_USER=${MQTT_USER:-""}
MQTT_PASS=${MQTT_PASS:-""}
# TLS mode: true khi kết nối HiveMQ Cloud
MQTT_TLS=${MQTT_TLS:-false}

# ── Color output ──────────────────────────────────────────────────────────────
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${BLUE}[INFO]${NC}  $1"; }
log_ok()    { echo -e "${GREEN}[PASS]${NC}  $1"; }
log_fail()  { echo -e "${RED}[FAIL]${NC}  $1"; }
log_test()  { echo -e "${YELLOW}[TEST]${NC}  $1"; }

# ── Check mosquitto_pub available ─────────────────────────────────────────────
check_deps() {
  if ! command -v mosquitto_pub &> /dev/null; then
    log_fail "mosquitto_pub not found. Install with:"
    echo "  macOS:  brew install mosquitto"
    echo "  Ubuntu: sudo apt install mosquitto-clients"
    exit 1
  fi
  log_ok "mosquitto_pub found: $(mosquitto_pub --version 2>&1 | head -1)"
}

# ── Publish helper ────────────────────────────────────────────────────────────
publish() {
  local topic="$1"
  local payload="$2"
  local flags=""

  # Auth
  [ -n "$MQTT_USER" ] && flags="$flags -u $MQTT_USER -P $MQTT_PASS"

  # TLS — HiveMQ Cloud dùng Let's Encrypt (system CA)
  if [ "$MQTT_TLS" = "true" ]; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
      # macOS: brew install ca-certificates
      flags="$flags --capath /opt/homebrew/etc/ca-certificates"
    else
      # Linux
      flags="$flags --capath /etc/ssl/certs"
    fi
  fi

  mosquitto_pub -h "$BROKER" -p "$PORT" $flags -t "$topic" -m "$payload"
  if [ $? -eq 0 ]; then
    log_ok "Published to: $topic"
  else
    log_fail "Failed to publish to: $topic"
  fi
  sleep 0.5
}

# ── TEST CASES ────────────────────────────────────────────────────────────────

# TC-01: Happy path — dữ liệu bình thường đầy đủ
test_happy_path() {
  log_test "TC-01: Happy path — cân 45.2kg, battery 85%, sample_count 3"
  PAYLOAD=$(cat <<EOF
{
  "deviceInfo": {
    "tenantId": "tenant-001",
    "applicationId": "app-001",
    "deviceName": "Scale-A1",
    "devEui": "${DEVICE_EUI}"
  },
  "devAddr": "00112233",
  "adr": true,
  "dr": 5,
  "fCnt": 1001,
  "fPort": 1,
  "data": "",
  "object": {
    "raw_weight": 45.2,
    "battery_level": 85,
    "sample_count": 3
  },
  "rxInfo": [
    {
      "gatewayId": "gw-001",
      "rssi": -85,
      "snr": 7.5
    }
  ],
  "txInfo": {
    "frequency": 868100000,
    "modulation": {
      "lora": {
        "bandwidth": 125000,
        "spreadingFactor": 7,
        "codeRate": "CR_4_5"
      }
    }
  }
}
EOF
)
  publish "$TOPIC" "$PAYLOAD"
}

# TC-02: Duplicate packet — cùng fCnt → phải bị reject
test_duplicate() {
  log_test "TC-02: Duplicate packet (fCnt=1001 lần 2) — phải bị discard"
  PAYLOAD=$(cat <<EOF
{
  "deviceInfo": { "devEui": "${DEVICE_EUI}", "deviceName": "Scale-A1" },
  "fCnt": 1001,
  "object": { "raw_weight": 45.2, "battery_level": 85, "sample_count": 1 },
  "rxInfo": [{ "rssi": -85, "snr": 7.5 }],
  "txInfo": { "modulation": { "lora": { "spreadingFactor": 7 } } }
}
EOF
)
  publish "$TOPIC" "$PAYLOAD"
  log_info "→ Xem log app: phải thấy 'idempotency violation: duplicate telemetry packet'"
}

# TC-03: Battery = 0 → phải lưu được (edge case)
test_battery_zero() {
  log_test "TC-03: Battery = 0 (low battery) — phải LƯU được, không reject"
  PAYLOAD=$(cat <<EOF
{
  "deviceInfo": { "devEui": "${DEVICE_EUI}", "deviceName": "Scale-A1" },
  "fCnt": 1002,
  "object": { "raw_weight": 30.0, "battery_level": 0, "sample_count": 1 },
  "rxInfo": [{ "rssi": -90, "snr": 5.0 }],
  "txInfo": { "modulation": { "lora": { "spreadingFactor": 9 } } }
}
EOF
)
  publish "$TOPIC" "$PAYLOAD"
  log_info "→ Xem log app: phải thấy 'Received telemetry message' và lưu thành công"
}

# TC-04: Battery = 101 → PHẢI bị reject (validation error)
test_battery_invalid() {
  log_test "TC-04: Battery = 101 (invalid) — phải bị REJECT"
  PAYLOAD=$(cat <<EOF
{
  "deviceInfo": { "devEui": "${DEVICE_EUI}", "deviceName": "Scale-A1" },
  "fCnt": 1003,
  "object": { "raw_weight": 20.0, "battery_level": 101, "sample_count": 1 },
  "rxInfo": [{ "rssi": -75, "snr": 9.0 }],
  "txInfo": { "modulation": { "lora": { "spreadingFactor": 7 } } }
}
EOF
)
  publish "$TOPIC" "$PAYLOAD"
  log_info "→ Xem log app: phải thấy 'payload validation error'"
}

# TC-05: Empty device_id → PHẢI bị reject
test_missing_device_id() {
  log_test "TC-05: Missing devEui — phải bị REJECT"
  PAYLOAD=$(cat <<EOF
{
  "deviceInfo": { "devEui": "", "deviceName": "Unknown" },
  "fCnt": 1004,
  "object": { "raw_weight": 10.0, "battery_level": 50, "sample_count": 1 },
  "rxInfo": [{ "rssi": -80, "snr": 6.0 }],
  "txInfo": { "modulation": { "lora": { "spreadingFactor": 7 } } }
}
EOF
)
  publish "$TOPIC" "$PAYLOAD"
  log_info "→ Xem log app: phải thấy 'device_id missing from payload'"
}

# TC-06: Thiết bị chưa đăng ký (device_id khác hoàn toàn)
test_unknown_device() {
  log_test "TC-06: Unknown device (chưa register) — phải log warning, không crash"
  UNKNOWN_TOPIC="application/1/device/ffffffffffffffff/event/up"
  PAYLOAD=$(cat <<EOF
{
  "deviceInfo": { "devEui": "ffffffffffffffff", "deviceName": "Unknown Scale" },
  "fCnt": 1,
  "object": { "raw_weight": 5.0, "battery_level": 70, "sample_count": 1 },
  "rxInfo": [{ "rssi": -95, "snr": 3.0 }],
  "txInfo": { "modulation": { "lora": { "spreadingFactor": 12 } } }
}
EOF
)
  publish "$UNKNOWN_TOPIC" "$PAYLOAD"
  log_info "→ Xem log app: phải thấy warning về unknown device, app KHÔNG crash"
}

# TC-07: Raw weight = 0 → phải LƯU được (container rỗng)
test_zero_weight() {
  log_test "TC-07: Weight = 0kg (container rỗng) — phải LƯU được"
  PAYLOAD=$(cat <<EOF
{
  "deviceInfo": { "devEui": "${DEVICE_EUI}", "deviceName": "Scale-A1" },
  "fCnt": 1005,
  "object": { "raw_weight": 0.0, "battery_level": 90, "sample_count": 1 },
  "rxInfo": [{ "rssi": -82, "snr": 8.0 }],
  "txInfo": { "modulation": { "lora": { "spreadingFactor": 7 } } }
}
EOF
)
  publish "$TOPIC" "$PAYLOAD"
  log_info "→ Xem log app: raw_weight=0 phải được lưu bình thường"
}

# TC-08: Malformed JSON → phải reject gracefully (không crash)
test_malformed_json() {
  log_test "TC-08: Malformed JSON — phải reject, app KHÔNG crash"
  publish "$TOPIC" "this is not json at all {{{"
  log_info "→ Xem log app: phải thấy 'json unmarshal failed'"
}

# TC-09: Nhiều thiết bị đồng thời (stress nhẹ)
test_multi_device() {
  log_test "TC-09: 5 thiết bị publish đồng thời"
  for i in 1 2 3 4 5; do
    DEV_EUI=$(printf "aabbccdd%08x" $i)
    DEV_TOPIC="application/1/device/${DEV_EUI}/event/up"
    PAYLOAD=$(cat <<EOF
{
  "deviceInfo": { "devEui": "${DEV_EUI}", "deviceName": "Scale-${i}" },
  "fCnt": $((2000 + i)),
  "object": { "raw_weight": $((40 + i)).5, "battery_level": $((60 + i * 5)), "sample_count": 1 },
  "rxInfo": [{ "rssi": $((- 80 - i)), "snr": 7.0 }],
  "txInfo": { "modulation": { "lora": { "spreadingFactor": 7 } } }
}
EOF
)
    mosquitto_pub -h "$BROKER" -p "$PORT" -t "$DEV_TOPIC" -m "$PAYLOAD" &
  done
  wait
  log_ok "5 messages published concurrently"
}

# ── RUNNER ────────────────────────────────────────────────────────────────────
run_all() {
  echo ""
  echo "════════════════════════════════════════════════════════"
  echo "  MQTT Ingestion Test Suite — Inventory Management"
  echo "  Broker: $BROKER:$PORT"
  echo "  Topic:  $TOPIC"
  echo "════════════════════════════════════════════════════════"
  echo ""

  log_info "Xem log app song song:"
  log_info "  docker logs inventory_app -f --tail 20"
  echo ""

  test_happy_path
  sleep 1
  test_duplicate
  sleep 1
  test_battery_zero
  sleep 1
  test_battery_invalid
  sleep 1
  test_missing_device_id
  sleep 1
  test_unknown_device
  sleep 1
  test_zero_weight
  sleep 1
  test_malformed_json
  sleep 1
  test_multi_device

  echo ""
  echo "════════════════════════════════════════════════════════"
  log_ok "Tất cả test cases đã publish xong"
  echo ""
  log_info "Kiểm tra kết quả trong log của app:"
  echo "  docker logs inventory_app -f --tail 50"
  echo ""
  log_info "Kiểm tra data trong DB:"
  echo "  docker exec inventory_db psql -U inventory_user -d inventory_db -c 'SELECT device_id, raw_weight, battery_level, received_at FROM raw_telemetry ORDER BY received_at DESC LIMIT 10;'"
  echo "════════════════════════════════════════════════════════"
}

# ── ENTRYPOINT ────────────────────────────────────────────────────────────────
check_deps

case "${1:-all}" in
  all)        run_all ;;
  happy)      test_happy_path ;;
  duplicate)  test_duplicate ;;
  battery0)   test_battery_zero ;;
  invalid)    test_battery_invalid ;;
  noid)       test_missing_device_id ;;
  unknown)    test_unknown_device ;;
  zero)       test_zero_weight ;;
  malformed)  test_malformed_json ;;
  stress)     test_multi_device ;;
  *)
    echo "Usage: $0 [all|happy|duplicate|battery0|invalid|noid|unknown|zero|malformed|stress]"
    exit 1
    ;;
esac
