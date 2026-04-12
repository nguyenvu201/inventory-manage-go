// Package telemetry_test implements tests for INV-SPR01-TASK-002
// AC Coverage:
//   AC-08: TestProcessor_Process_MovingAverage
// IEC 62304 Classification: Software Safety Class B
package telemetry_test

import (
	"context"
	"testing"
	"time"

	"inventory-manage/internal/domain/telemetry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessor_Process_MovingAverage(t *testing.T) {
	tests := []struct {
		name          string
		deviceID      string
		sequence      []float64
		sampleCount   int
		expectedFinal float64
	}{
		{
			name:          "AC-08: sample_count > 1 skips averaging",
			deviceID:      "SCALE-001",
			sequence:      []float64{100, 200, 300},
			sampleCount:   3,
			expectedFinal: 300, // No moving average, uses exact last value
		},
		{
			name:          "AC-08: sample_count == 1 uses moving average of last 5",
			deviceID:      "SCALE-002",
			sequence:      []float64{10.0, 20.0, 30.0, 40.0, 50.0},
			sampleCount:   1,
			expectedFinal: 30.0, // (10+20+30+40+50)/5 = 30
		},
		{
			name:          "AC-08: moving average with sparse history (less than 5 readings)",
			deviceID:      "SCALE-003",
			sequence:      []float64{10.0, 20.0, 30.0},
			sampleCount:   1,
			expectedFinal: 20.0, // (10+20+30)/3 = 20
		},
		{
			name:          "AC-08: rolling buffer evicts oldest (reading 6)",
			deviceID:      "SCALE-004",
			sequence:      []float64{10.0, 20.0, 30.0, 40.0, 50.0, 60.0},
			sampleCount:   1,
			expectedFinal: 40.0, // (20+30+40+50+60)/5 = 40
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := telemetry.NewProcessor()
			ctx := context.Background()

			var finalPayload telemetry.TelemetryPayload
			for _, val := range tt.sequence {
				payload := telemetry.TelemetryPayload{
					DeviceID:    tt.deviceID,
					RawWeight:   val,
					SampleCount: tt.sampleCount,
					ReceivedAt:  time.Now(),
				}

				processed, err := proc.Process(ctx, payload)
				require.NoError(t, err)
				finalPayload = processed
			}

			assert.InDelta(t, tt.expectedFinal, finalPayload.RawWeight, 0.001)
		})
	}
}
