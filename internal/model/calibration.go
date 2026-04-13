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
	CalibrationType string                `json:"calibration_type"`
	EffectiveFrom  time.Time              `json:"effective_from"`
	DeactivatedAt  *time.Time             `json:"deactivated_at,omitempty"`
	CreatedBy      string                 `json:"created_by"`
	CreatedAt      time.Time              `json:"created_at"`
}

// Calibration Types
const (
	CalibrationTypeInitial         = "initial"
	CalibrationTypePeriodic        = "periodic"
	CalibrationTypeDriftCorrection = "drift_correction"
)

// UpdateCalibrationParams represents the incoming request to update calibration
type UpdateCalibrationParams struct {
	ZeroValue      float64                `json:"zero_value" binding:"required"`
	SpanValue      float64                `json:"span_value" binding:"required"`
	Unit           string                 `json:"unit" binding:"required"`
	CapacityMax    float64                `json:"capacity_max" binding:"required"`
	HardwareConfig map[string]interface{} `json:"hardware_config"`
	CalibrationType string                `json:"calibration_type" binding:"required"`
	CreatedBy      string                 `json:"created_by" binding:"required"`
}

