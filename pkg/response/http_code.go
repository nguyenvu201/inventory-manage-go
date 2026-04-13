package response

// Inventory Management System — API Response Error Codes
//
// Code format:
//   20xxx — Success
//   40xxx — Client error (bad request, not found, conflict)
//   50xxx — Server error (internal, dependency failure)

const (
	// ── General ─────────────────────────────────────────────
	ErrCodeSuccess      = 20001
	ErrCodeParamInvalid = 20003

	// ── Device ──────────────────────────────────────────────
	ErrCodeDeviceNotFound  = 40001
	ErrCodeDeviceDuplicate = 40002
	ErrCodeDeviceInvalid   = 40003

	// ── Calibration ─────────────────────────────────────────
	ErrCodeCalibrationNotFound = 40011
	ErrCodeCalibrationInvalid  = 40012

	// ── Telemetry ────────────────────────────────────────────
	ErrCodeTelemetryDuplicate = 40021
	ErrCodeTelemetryInvalid   = 40022

	// ── Server ───────────────────────────────────────────────
	ErrCodeInternalServer = 50001
	ErrCodeDatabaseError  = 50002
	ErrCodeRedisError     = 50003
)

var msg = map[int]string{
	ErrCodeSuccess:      "success",
	ErrCodeParamInvalid: "invalid parameters",

	ErrCodeDeviceNotFound:  "device not found",
	ErrCodeDeviceDuplicate: "device already exists",
	ErrCodeDeviceInvalid:   "invalid device data",

	ErrCodeCalibrationNotFound: "calibration config not found",
	ErrCodeCalibrationInvalid:  "invalid calibration data",

	ErrCodeTelemetryDuplicate: "duplicate telemetry packet",
	ErrCodeTelemetryInvalid:   "invalid telemetry payload",

	ErrCodeInternalServer: "internal server error",
	ErrCodeDatabaseError:  "database error",
	ErrCodeRedisError:     "cache error",
}

// GetMsg returns the human-readable message for a given code.
// Returns a generic message if the code is not registered.
func GetMsg(code int) string {
	if m, ok := msg[code]; ok {
		return m
	}
	return "unknown error"
}
