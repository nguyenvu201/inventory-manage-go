package model

import "time"

// CalibrationConfig represents a calibration profile for an IoT scale.
// Multiple calibrations may exist for the same device; only one is active at a time.
type CalibrationConfig struct {
	ID             int                    `json:"id"`
	DeviceID       string                 `json:"device_id"`
	ZeroValue      float64                `json:"zero_value"`
	SpanValue      float64                `json:"span_value"`
	Unit           string                 `json:"unit"`
	CapacityMax    float64                `json:"capacity_max"`
	HardwareConfig map[string]interface{} `json:"hardware_config"`
	EffectiveFrom  time.Time              `json:"effective_from"`
	DeactivatedAt  *time.Time             `json:"deactivated_at,omitempty"`
	CreatedBy      string                 `json:"created_by"`
	CreatedAt      time.Time              `json:"created_at"`
}
