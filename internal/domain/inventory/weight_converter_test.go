// Package inventory_test implements tests for INV-SPR03-TASK-001
// AC Coverage:
//   AC-01: TestWeightConverter_ConvertToNetWeight
//   AC-02: TestWeightConverter_ConvertToNetWeight
//   AC-03: TestWeightConverter_ConvertToNetWeight
//   AC-04: TestWeightConverter_Units
//   AC-05: TestWeightConverter_Rounding
//   AC-06: TestWeightConverter_EdgeCases
//   AC-07: TestWeightConverter_Errors
//   AC-08: TestWeightConverter_Resolution
// IEC 62304 Classification: Software Safety Class B
package inventory_test

import (
	"inventory-manage/internal/domain/inventory"
	"inventory-manage/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeightConverter_ConvertToNetWeight(t *testing.T) {
	converter := inventory.NewWeightConverter()

	tests := []struct {
		name         string
		rawWeight    float64
		config       *model.CalibrationConfig
		tareWeightKg float64
		resolution   float64
		wantGrossKg  float64
		wantNetKg    float64
		wantErr      error
	}{
		{
			name:      "AC-02/03: Normal calculation in kg",
			rawWeight: 5000,
			config: &model.CalibrationConfig{
				ZeroValue:   1000,
				SpanValue:   8000,
				CapacityMax: 40,
				Unit:        "kg",
			},
			tareWeightKg: 2.0,
			resolution:   0,
			wantGrossKg:  20.0, // (5000-1000) * (40/8000) = 4000 * 0.005 = 20.0
			wantNetKg:    18.0,
		},
		{
			name:      "AC-04: Support unit lb - normalize to kg",
			rawWeight: 5000,
			config: &model.CalibrationConfig{
				ZeroValue:   1000,
				SpanValue:   8000,
				CapacityMax: 40, // 40 lbs
				Unit:        "lb",
			},
			tareWeightKg: 2.0,
			resolution:   0,
			// gross is 20 lbs -> 20 * 0.45359237 = 9.0718474 kg -> round to 9.072
			wantGrossKg: 9.072,
			// net -> 9.072 - 2.0 = 7.072
			wantNetKg: 7.072,
		},
		{
			name:      "AC-04: Support unit g - normalize to kg",
			rawWeight: 5000,
			config: &model.CalibrationConfig{
				ZeroValue:   1000,
				SpanValue:   8000,
				CapacityMax: 40000, // 40000 g
				Unit:        "g",
			},
			tareWeightKg: 2.0,
			resolution:   0,
			// gross is 20000 g -> 20.0 kg
			wantGrossKg: 20.0,
			wantNetKg:   18.0,
		},
		{
			name:      "AC-05: Round results to 3 decimal places",
			rawWeight: 3333,
			config: &model.CalibrationConfig{
				ZeroValue:   1000,
				SpanValue:   8000,
				CapacityMax: 40,
				Unit:        "kg",
			},
			tareWeightKg: 2.111,
			resolution:   0,
			// (3333 - 1000) * (40/8000) = 2333 * 0.005 = 11.665
			// net = 11.665 - 2.111 = 9.554
			wantGrossKg: 11.665,
			wantNetKg:   9.554,
		},
		{
			name:      "AC-08: Apply configurable measurement resolution (0.5kg)",
			rawWeight: 3100, // gross = 2100 * 0.005 = 10.5 kg
			config: &model.CalibrationConfig{
				ZeroValue:   1000,
				SpanValue:   8000,
				CapacityMax: 40,
				Unit:        "kg",
			},
			tareWeightKg: 2.1, // net before res = 10.5 - 2.1 = 8.4 kg
			resolution:   0.5,
			wantGrossKg:  10.5,
			wantNetKg:    8.5, // 8.4 rounded to nearest 0.5 is 8.5
		},
		{
			name:      "AC-08: Apply configurable measurement resolution (0.1kg)",
			rawWeight: 3100,
			config: &model.CalibrationConfig{
				ZeroValue:   1000,
				SpanValue:   8000,
				CapacityMax: 40,
				Unit:        "kg",
			},
			tareWeightKg: 2.12, // net before res = 10.5 - 2.12 = 8.38 kg
			resolution:   0.1,
			wantGrossKg:  10.5,
			wantNetKg:    8.4, // 8.38 rounded to nearest 0.1 is 8.4
		},
		{
			name:         "AC-06: Edge case - raw_weight = 0",
			rawWeight:    0,
			config:       &model.CalibrationConfig{ZeroValue: 1000, SpanValue: 8000, CapacityMax: 40, Unit: "kg"},
			tareWeightKg: 2.0,
			resolution:   0,
			wantGrossKg:  -5.0, // (0 - 1000) * 0.005 = -5.0
			wantNetKg:    -7.0, // -5.0 - 2.0 = -7.0
		},
		{
			name:         "AC-06: Edge case - raw_weight = max ADC (65535)",
			rawWeight:    65535,
			config:       &model.CalibrationConfig{ZeroValue: 1000, SpanValue: 8000, CapacityMax: 40, Unit: "kg"},
			tareWeightKg: 2.0,
			resolution:   0,
			wantGrossKg:  322.675, // (65535-1000)*0.005 = 64535 * 0.005 = 322.675
			wantNetKg:    320.675, // 322.675 - 2.0 = 320.675
		},
		{
			name:      "AC-07: Error - no active calibration",
			rawWeight: 5000,
			config:    nil,
			wantErr:   model.ErrNoActiveCalibration,
		},
		{
			name:      "AC-07: Error - span value is zero (divide by zero protection)",
			rawWeight: 5000,
			config: &model.CalibrationConfig{
				ZeroValue:   1000,
				SpanValue:   0,
				CapacityMax: 40,
				Unit:        "kg",
			},
			wantErr: model.ErrInvalidSpanValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := converter.ConvertToNetWeight(tt.rawWeight, tt.config, tt.tareWeightKg, tt.resolution)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
			assert.InDelta(t, tt.wantGrossKg, res.GrossWeightKg, 0.001)
			assert.InDelta(t, tt.wantNetKg, res.NetWeightKg, 0.001)
		})
	}
}
