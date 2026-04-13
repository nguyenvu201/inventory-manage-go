package model

import "errors"

// WeightResult represents the result of converting raw ADC readings to physical weights
type WeightResult struct {
	GrossWeightKg float64 `json:"gross_weight_kg"`
	NetWeightKg   float64 `json:"net_weight_kg"`
}

// Domain errors
var (
	ErrNoActiveCalibration = errors.New("no active calibration found for device")
	ErrInvalidSpanValue    = errors.New("invalid calibration span value (cannot be zero)")
)
