CREATE TABLE IF NOT EXISTS sku_configs (
    sku_code VARCHAR(100) PRIMARY KEY,
    unit_weight_kg NUMERIC(10,3) NOT NULL,
    full_capacity_kg NUMERIC(10,3) NOT NULL,
    tare_weight_kg NUMERIC(10,3) NOT NULL,
    resolution_kg NUMERIC(10,3) NOT NULL DEFAULT 0,
    reorder_point_qty INT NOT NULL,
    unit_label VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inventory_snapshots (
    device_id VARCHAR(100) PRIMARY KEY REFERENCES devices(device_id) ON DELETE CASCADE,
    sku_code VARCHAR(100) NOT NULL REFERENCES sku_configs(sku_code) ON DELETE RESTRICT,
    net_weight_kg NUMERIC(10,3) NOT NULL,
    qty INT NOT NULL,
    percentage NUMERIC(5,2) NOT NULL,
    snapshot_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inventory_snapshots_sku ON inventory_snapshots(sku_code);
