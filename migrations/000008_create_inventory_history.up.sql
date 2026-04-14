CREATE TABLE IF NOT EXISTS inventory_history (
    device_id VARCHAR(100) NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    sku_code VARCHAR(100) NOT NULL REFERENCES sku_configs(sku_code) ON DELETE RESTRICT,
    net_weight_kg NUMERIC(10,3) NOT NULL,
    qty INT NOT NULL,
    percentage NUMERIC(5,2) NOT NULL,
    snapshot_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

SELECT create_hypertable('inventory_history', 'snapshot_at', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_inventory_history_sku_code_time ON inventory_history (sku_code, snapshot_at DESC);
