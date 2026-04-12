DROP TRIGGER IF EXISTS set_devices_timestamp ON devices;
DROP FUNCTION IF EXISTS trigger_set_devices_timestamp();

DROP TABLE IF EXISTS devices CASCADE;
