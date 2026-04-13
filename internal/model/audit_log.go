package model

import (
	"encoding/json"
	"time"
)

// CalibrationAuditLog represents the audit trail entity triggered on calibration updates.
type CalibrationAuditLog struct {
	ID          int64           `json:"id" db:"id"`
	DeviceID    string          `json:"device_id" db:"device_id"`
	Action      string          `json:"action" db:"action"`
	OldValues   json.RawMessage `json:"old_values" db:"old_values"`
	NewValues   json.RawMessage `json:"new_values" db:"new_values"`
	PerformedBy string          `json:"performed_by" db:"performed_by"`
	PerformedAt time.Time       `json:"performed_at" db:"performed_at"`
	Reason      string          `json:"reason" db:"reason"`
}
