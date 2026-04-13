package inventory

import (
	"inventory-manage/internal/model"
	"math"
)

// WeightConverter handles the conversion of raw ADC readings to physical weights
type WeightConverter struct{}

// NewWeightConverter creates a new instance of WeightConverter
func NewWeightConverter() *WeightConverter {
	return &WeightConverter{}
}

// ConvertToNetWeight applies calibration formulas to convert a raw ADC weight to a net physical weight.
// gross_weight = (raw_weight - zero_value) * (capacity_max / span_value)
// net_weight = gross_weight - tare_weight
// Results are normalized to standard kg.
func (wc *WeightConverter) ConvertToNetWeight(rawWeight float64, config *model.CalibrationConfig, tareWeightKg float64, resolutionKg float64) (*model.WeightResult, error) {
	if config == nil {
		return nil, model.ErrNoActiveCalibration
	}

	if config.SpanValue == 0 {
		return nil, model.ErrInvalidSpanValue
	}

	// Calculate gross weight in the calibration's configured unit
	grossWeight := (rawWeight - config.ZeroValue) * (config.CapacityMax / config.SpanValue)

	// Normalize gross weight to kg
	grossWeightKg := normalizeToKg(grossWeight, config.Unit)

	// Round gross weight to 3 decimal places
	grossWeightKg = roundTo3Decimals(grossWeightKg)

	// Calculate net weight
	netWeightKg := grossWeightKg - tareWeightKg

	// Provide a floor of 0 if net weight goes slightly negative due to minor ADC fluctuation around tare
	// But according to AC-06, edge cases raw_weight=0 etc needs to be handled.
	// We'll let it be negative if it's actually negative, unless explicitly required to clamp to 0.

	// Apply SKU measurement resolution rounding (e.g., snap to nearest 0.1, 0.5)
	if resolutionKg > 0 {
		netWeightKg = math.Round(netWeightKg/resolutionKg) * resolutionKg
	}

	// Final round to 3 decimal places to remove floating-point artifacts after resolution multiplier
	netWeightKg = roundTo3Decimals(netWeightKg)

	return &model.WeightResult{
		GrossWeightKg: grossWeightKg,
		NetWeightKg:   netWeightKg,
	}, nil
}

// normalizeToKg converts a standardized weight reading into kg
func normalizeToKg(value float64, unit string) float64 {
	switch unit {
	case "g":
		return value * 0.001
	case "lb":
		return value * 0.45359237
	case "kg":
		return value
	default:
		// Default to 1:1 if unit is unknown
		return value
	}
}

// roundTo3Decimals rounds a float64 value to exactly 3 decimal places
func roundTo3Decimals(value float64) float64 {
	return math.Round(value*1000) / 1000
}
