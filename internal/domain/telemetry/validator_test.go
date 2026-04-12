// Package telemetry_test implements tests for INV-SPR01-TASK-003
// AC Coverage:
//   AC-02: TestTelemetryValidator_Validate
//   AC-04: TestTelemetryValidator_Validate (testing ValidationError interface)
//   AC-05: TestTelemetryValidator_Validate (table-driven logic)
// IEC 62304 Classification: Software Safety Class B
package telemetry_test

import (
	"testing"
	"time"

	"inventory-manage/internal/domain/telemetry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetryValidator_Validate(t *testing.T) {
	v := telemetry.NewValidator()

	tests := []struct {
		name    string
		input   telemetry.TelemetryPayload
		wantErr bool
		errMsg  string
	}{
		{
			name: "AC-02: valid payload — all fields present and in range",
			input: telemetry.TelemetryPayload{
				DeviceID:     "SCALE-001",
				RawWeight:    5000,
				BatteryLevel: 85,
				FCnt:         1234,
				ReceivedAt:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "AC-02: battery_level=101 must be rejected",
			input: telemetry.TelemetryPayload{
				DeviceID:     "SCALE-001",
				RawWeight:    5000,
				BatteryLevel: 101,
			},
			wantErr: true,
			errMsg:  "BatteryLevel",
		},
		{
			name: "AC-02: battery_level=-1 must be rejected",
			input: telemetry.TelemetryPayload{
				DeviceID:     "SCALE-001",
				RawWeight:    5000,
				BatteryLevel: -1,
			},
			wantErr: true,
			errMsg:  "BatteryLevel",
		},
		{
			name: "AC-02: battery_level=0 (dead battery) must be accepted",
			input: telemetry.TelemetryPayload{
				DeviceID:     "SCALE-001",
				RawWeight:    5000,
				BatteryLevel: 0,
			},
			wantErr: false,
		},
		{
			name: "empty device_id must be rejected",
			input: telemetry.TelemetryPayload{
				DeviceID:     "",
				RawWeight:    5000,
				BatteryLevel: 50,
			},
			wantErr: true,
			errMsg:  "DeviceID",
		},
		{
			name: "negative raw_weight must be rejected",
			input: telemetry.TelemetryPayload{
				DeviceID:     "SCALE-001",
				RawWeight:    -100,
				BatteryLevel: 50,
			},
			wantErr: true,
			errMsg:  "RawWeight",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				
				// Assert AC-04: Returns structured validation error
				var valErrs *telemetry.ValidationErrors
				require.ErrorAs(t, err, &valErrs)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}
