DROP TRIGGER IF EXISTS trg_prevent_calibration_delete ON calibration_configs;

DROP FUNCTION IF EXISTS prevent_calibration_delete();

ALTER TABLE calibration_configs 
DROP CONSTRAINT IF EXISTS chk_calibration_type;

ALTER TABLE calibration_configs 
DROP COLUMN IF EXISTS calibration_type;
