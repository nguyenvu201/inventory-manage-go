CREATE TABLE IF NOT EXISTS threshold_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku_code VARCHAR(100) NOT NULL,
    rule_type VARCHAR(50) NOT NULL,
    trigger_percentage NUMERIC(5,2),
    trigger_qty INT,
    cooldown_minutes INT NOT NULL DEFAULT 60,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_threshold_sku FOREIGN KEY (sku_code) REFERENCES sku_configs(sku_code) ON DELETE CASCADE,
    CONSTRAINT threshold_rules_unique_sku_type UNIQUE (sku_code, rule_type)
);

CREATE INDEX idx_threshold_rules_sku ON threshold_rules(sku_code);
