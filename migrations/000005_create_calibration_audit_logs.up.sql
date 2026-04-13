CREATE TABLE IF NOT EXISTS calibration_audit_logs (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(50) NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    action VARCHAR(20) NOT NULL CHECK (action IN ('INSERT', 'UPDATE')),
    old_values JSONB,
    new_values JSONB,
    performed_by VARCHAR(100) NOT NULL,
    performed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT
);

CREATE INDEX idx_calibration_audit_logs_device_id ON calibration_audit_logs(device_id);
CREATE INDEX idx_calibration_audit_logs_performed_at ON calibration_audit_logs(performed_at);

-- Trigger 1: Auto Log INSERT / UPDATE
CREATE OR REPLACE FUNCTION log_calibration_changes()
RETURNS TRIGGER AS $$
DECLARE
    audit_action VARCHAR(20);
    old_data JSONB;
    new_data JSONB;
    user_id VARCHAR(100);
BEGIN
    IF TG_OP = 'INSERT' THEN
        audit_action := 'INSERT';
        old_data := NULL;
        new_data := to_jsonb(NEW);
        user_id := NEW.created_by;
    ELSIF TG_OP = 'UPDATE' THEN
        audit_action := 'UPDATE';
        old_data := to_jsonb(OLD);
        new_data := to_jsonb(NEW);
        user_id := COALESCE(NEW.created_by, OLD.created_by);
    END IF;

    INSERT INTO calibration_audit_logs (device_id, action, old_values, new_values, performed_by, reason)
    VALUES (NEW.device_id, audit_action, old_data, new_data, user_id, 'System auto-audit trigger');

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_calibration_audit
AFTER INSERT OR UPDATE ON calibration_configs
FOR EACH ROW EXECUTE FUNCTION log_calibration_changes();

-- Trigger 2: Append Only Table (AC-03)
CREATE OR REPLACE FUNCTION prevent_audit_tampering()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'FDA Compliance Violation: Audit logs cannot be modified or deleted.';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_audit_tampering
BEFORE UPDATE OR DELETE ON calibration_audit_logs
FOR EACH ROW EXECUTE FUNCTION prevent_audit_tampering();
