DROP TRIGGER IF EXISTS trg_prevent_audit_tampering ON calibration_audit_logs;
DROP FUNCTION IF EXISTS prevent_audit_tampering();

DROP TRIGGER IF EXISTS trg_calibration_audit ON calibration_configs;
DROP FUNCTION IF EXISTS log_calibration_changes();

DROP TABLE IF EXISTS calibration_audit_logs;
