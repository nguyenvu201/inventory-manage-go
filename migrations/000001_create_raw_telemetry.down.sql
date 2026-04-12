-- Migration: 000001_create_raw_telemetry.down.sql
-- Reverses the raw_telemetry table creation.
-- WARNING: This will permanently delete all raw telemetry data.

DROP INDEX IF EXISTS idx_raw_telemetry_signal;
DROP INDEX IF EXISTS idx_raw_telemetry_device_id;
DROP INDEX IF EXISTS uq_raw_telemetry_device_fcnt;
DROP TABLE IF EXISTS raw_telemetry;
