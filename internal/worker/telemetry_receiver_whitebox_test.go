// Package worker implements tests for internal worker package behavior.
// AC Coverage:
//   telemetry_receiver_whitebox_test.go covers integration between handleMessage and ProcessPayload
// IEC 62304 Classification: Software Safety Class B
package worker

import (
	"context"
	"testing"
	"time"

	"inventory-manage/internal/domain/telemetry"
	"inventory-manage/internal/model"

	"github.com/stretchr/testify/require"
)

// mockMessage implements paho.Message
type mockMessage struct {
	payload []byte
}

func (m mockMessage) Duplicate() bool  { return false }
func (m mockMessage) Qos() byte        { return 1 }
func (m mockMessage) Retained() bool   { return false }
func (m mockMessage) Topic() string    { return "test/topic" }
func (m mockMessage) MessageID() uint16 { return 1 }
func (m mockMessage) Payload() []byte  { return m.payload }
func (m mockMessage) Ack()             {}

func TestReceiver_HandleMessage(t *testing.T) {
	processor := telemetry.NewProcessor()
	validator := telemetry.NewValidator()
	outChan := make(chan model.TelemetryPayload, 1)
	receiver := NewTelemetryReceiver(nil, processor, validator, outChan)

	validJSON := []byte(`{
		"deviceInfo": {"devEui": "SCALE-VALID-01"},
		"fCnt": 42,
		"object": {"raw_weight": 5200.5, "battery_level": 85, "sample_count": 1}
	}`)

	msg := mockMessage{payload: validJSON}

	// Should not panic, should push to channel
	receiver.handleMessage(nil, msg)

	select {
	case <-outChan:
		// success
	case <-time.After(1 * time.Second):
		t.Fatal("failed to receive processed message")
	}
}

func TestReceiver_HandleMessage_ErrorLogs(t *testing.T) {
	processor := telemetry.NewProcessor()
	validator := telemetry.NewValidator()
	outChan := make(chan model.TelemetryPayload, 1)
	receiver := NewTelemetryReceiver(nil, processor, validator, outChan)

	invalidJSON := []byte(`{invalid}`)
	msg := mockMessage{payload: invalidJSON}

	// Should log error but not panic
	receiver.handleMessage(nil, msg)
}

func TestReceiver_ProcessorError(t *testing.T) {
	// Push a valid payload but timeout downstream to force an error being returned by ProcessPayload.
	processor := telemetry.NewProcessor()
	validator := telemetry.NewValidator()
	outChan := make(chan model.TelemetryPayload)
	receiver := NewTelemetryReceiver(nil, processor, validator, outChan)

	validJSON := []byte(`{
		"deviceInfo": {"devEui": "SCALE-BLOCK"},
		"fCnt": 42,
		"object": {"raw_weight": 1111, "battery_level": 85, "sample_count": 1}
	}`)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := receiver.ProcessPayload(ctx, validJSON)
	require.Error(t, err)
}
