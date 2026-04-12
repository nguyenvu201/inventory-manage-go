// Package telemetry_test implements tests for INV-SPR01-TASK-003
// AC Coverage:
//   AC-03: TestDecoder_DecodeBase64Payload (extract Base64)
//   AC-06: TestDecoder_DecodeBase64Payload (handle partial/short payloads gracefully)
package telemetry_test

import (
	"encoding/base64"
	"testing"

	"inventory-manage/internal/domain/telemetry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecoder_DecodeBase64Payload(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		setupBase64   []byte
		wantWeight    float64
		wantBattery   int8
		wantSample    int
		wantErr       bool
		errContains   string
	}{
		{
			name:        "AC-03: successfully parses 4-byte payload (Weight: 5000g, Battery: 85, Sample: 3)",
			setupBase64: []byte{0x13, 0x88, 85, 3}, // 0x1388 = 5000 (BigEndian)
			wantWeight:  5000.0,
			wantBattery: 85,
			wantSample:  3,
			wantErr:     false,
		},
		{
			name:        "successfully parses payload with weight 0",
			setupBase64: []byte{0x00, 0x00, 100, 1},
			wantWeight:  0.0,
			wantBattery: 100,
			wantSample:  1,
			wantErr:     false,
		},
		{
			name:        "empty string yields error",
			input:       "",
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "invalid base64 yields error",
			input:       "invalid_base64_!@#",
			wantErr:     true,
			errContains: "failed to decode Base64",
		},
		{
			name:        "AC-06: payload too short (3 bytes instead of 4) fails gracefully",
			setupBase64: []byte{0x13, 0x88, 85}, // Missing sample count
			wantErr:     true,
			errContains: "payload too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			if tt.setupBase64 != nil {
				input = base64.StdEncoding.EncodeToString(tt.setupBase64)
			}

			w, b, s, err := telemetry.DecodeBase64Payload(input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantWeight, w)
			assert.Equal(t, tt.wantBattery, b)
			assert.Equal(t, tt.wantSample, s)
		})
	}
}
