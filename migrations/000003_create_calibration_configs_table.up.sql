CREATE TABLE IF NOT EXISTS calibration_configs (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(50) NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    zero_value FLOAT NOT NULL,
    span_value FLOAT NOT NULL,
    unit VARCHAR(10) NOT NULL,
    capacity_max FLOAT NOT NULL,
    hardware_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deactivated_at TIMESTAMPTZ,
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Partial unique index to enforce only one active record per device
CREATE UNIQUE INDEX IF NOT EXISTS unique_active_calibration ON calibration_configs (device_id) WHERE deactivated_at IS NULL;
