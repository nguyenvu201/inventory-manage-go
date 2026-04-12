-- Migration: 000001_create_raw_telemetry.up.sql
-- Creates the raw_telemetry hypertable and supporting indexes.
-- Standards: FDA 21 CFR Part 11 / IEC 62304

-- Enable TimescaleDB extension (idempotent)
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- ── raw_telemetry ────────────────────────────────────────────────────────────
-- Stores every validated telemetry packet received from IoT scale nodes.
-- This is the immutable raw data layer — records are NEVER updated or deleted.
CREATE TABLE IF NOT EXISTS raw_telemetry (
    id              BIGSERIAL       NOT NULL,
    device_id       TEXT            NOT NULL,
    raw_weight      DOUBLE PRECISION NOT NULL,
    battery_level   SMALLINT        NOT NULL CHECK (battery_level BETWEEN 0 AND 100),
    -- LoRaWAN metadata (ChirpStack uplink)
    rssi            SMALLINT,
    snr             REAL,
    f_cnt           BIGINT,                     -- frame counter — used for idempotency
    spreading_factor SMALLINT,
    sample_count    SMALLINT        DEFAULT 1,  -- number of HX711 readings averaged on-node
    -- Payload archive
    payload_json    JSONB,
    -- Timestamps
    received_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    device_time     TIMESTAMPTZ,               -- timestamp reported by the device, if available
    CONSTRAINT raw_telemetry_pkey PRIMARY KEY (id, received_at)
);

-- Convert to TimescaleDB hypertable partitioned by received_at
SELECT create_hypertable(
    'raw_telemetry',
    'received_at',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Unique constraint for idempotent ingestion:
-- Same device + same LoRaWAN frame counter = duplicate packet from multiple gateways
-- NOTE: TimescaleDB hypertable partitioned by received_at requires the
--       partition column in all unique indexes.
CREATE UNIQUE INDEX IF NOT EXISTS uq_raw_telemetry_device_fcnt
    ON raw_telemetry (device_id, f_cnt, received_at)
    WHERE f_cnt IS NOT NULL;

-- Query index: look up telemetry by device_id within a time range
CREATE INDEX IF NOT EXISTS idx_raw_telemetry_device_id
    ON raw_telemetry (device_id, received_at DESC);

-- Signal quality index: for RSSI/SNR trend analysis
CREATE INDEX IF NOT EXISTS idx_raw_telemetry_signal
    ON raw_telemetry (device_id, received_at DESC)
    WHERE rssi IS NOT NULL;

-- Comment for FDA traceability
COMMENT ON TABLE raw_telemetry IS
    'Immutable raw telemetry store. Source of truth for all weight measurements. '
    'Records must not be deleted. TimescaleDB hypertable partitioned by received_at.';
