package telemetry

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

// DecodeBase64Payload extracts payload data from a raw Base64 string for devices
// that do not have a specialized payload parser defined in ChirpStack.
//
// Expects an exact 4-byte structure:
// Byte 0-1: uint16 BigEndian (RawWeight in grams)
// Byte 2:   uint8  (BatteryLevel 0-100)
// Byte 3:   uint8  (SampleCount)
func DecodeBase64Payload(data string) (rawWeight float64, battery int8, sampleCount int, err error) {
	if data == "" {
		return 0, 0, 0, fmt.Errorf("base64 payload data is empty")
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to decode Base64: %w", err)
	}

	// AC-06: handle partial payloads gracefully without panicking
	if len(decoded) < 4 {
		return 0, 0, 0, fmt.Errorf("payload too short: expected 4 bytes, got %d", len(decoded))
	}

	weightUint16 := binary.BigEndian.Uint16(decoded[0:2])
	rawWeight = float64(weightUint16)

	battery = int8(decoded[2])
	sampleCount = int(decoded[3])

	return rawWeight, battery, sampleCount, nil
}
