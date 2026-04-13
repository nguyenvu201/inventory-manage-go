ALTER TABLE calibration_configs 
ADD COLUMN IF NOT EXISTS calibration_type VARCHAR(20) NOT NULL DEFAULT 'initial';

-- Add check constraint for allowed calibration types
ALTER TABLE calibration_configs
ADD CONSTRAINT chk_calibration_type 
CHECK (calibration_type IN ('initial', 'periodic', 'drift_correction'));

-- AC-04: Prohibit deletion of calibration history records — only deactivation is allowed
CREATE OR REPLACE FUNCTION prevent_calibration_delete()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'FDA Compliance Violation: Deletion of calibration history is prohibited. Use deactivation (deactivated_at) instead.';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_calibration_delete
BEFORE DELETE ON calibration_configs
FOR EACH ROW
EXECUTE FUNCTION prevent_calibration_delete();
