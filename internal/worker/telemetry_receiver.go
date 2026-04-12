package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"inventory-manage/internal/domain/telemetry"
	inventorymqtt "inventory-manage/internal/platform/mqtt"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// TelemetryReceiver sits at the boundary, consuming MQTT messages, verifying their structure,
// passing them to the business rules processor, and feeding them to the downstream channel.
type TelemetryReceiver struct {
	client    *inventorymqtt.Client
	processor *telemetry.Processor
	outChan   chan<- telemetry.TelemetryPayload
}

// NewTelemetryReceiver constructs the gateway message receiver.
func NewTelemetryReceiver(client *inventorymqtt.Client, processor *telemetry.Processor, outChan chan<- telemetry.TelemetryPayload) *TelemetryReceiver {
	return &TelemetryReceiver{
		client:    client,
		processor: processor,
		outChan:   outChan,
	}
}

// Start begins subscribing to the Mosquitto broker (AC-02).
func (r *TelemetryReceiver) Start() error {
	topic := "application/+/device/+/event/up"
	log.Info().Str("topic", topic).Msg("Starting telemetry receiver subscription")

	err := r.client.Subscribe(topic, 1, r.handleMessage)
	if err != nil {
		return fmt.Errorf("telemetryReceiver.Start: %w", err)
	}

	return nil
}

// handleMessage is the callback from the Paho MQTT client.
func (r *TelemetryReceiver) handleMessage(client paho.Client, msg paho.Message) {
	// Create context with a timeout for internal processing
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.ProcessPayload(ctx, msg.Payload()); err != nil {
		// Just log the error, don't crash the worker
		log.Error().Err(err).Msg("failed to process telemetry MQTT message")
	}
}

// ProcessPayload handles the raw byte payload parsing and routing. Safe for unit testing.
func (r *TelemetryReceiver) ProcessPayload(ctx context.Context, raw []byte) error {
	traceID := uuid.New().String()

	var uplink telemetry.ChirpStackUplink
	if err := json.Unmarshal(raw, &uplink); err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}

	deviceID := uplink.DeviceInfo.DevEui
	if deviceID == "" {
		return fmt.Errorf("device_id missing from payload")
	}

	// AC-06: Log every received message including device_id and trace_id
	logger := log.With().Str("device_id", deviceID).Str("trace_id", traceID).Logger()
	logger.Info().Msg("Received telemetry message from gateway")

	// AC-09: Validate f_cnt field
	if uplink.FCnt == nil {
		return fmt.Errorf("f_cnt is missing in payload")
	}

	// Calculate Gateway metrics
	var rssi int
	var snr float32
	if len(uplink.RxInfo) > 0 {
		rssi = uplink.RxInfo[0].Rssi
		snr = uplink.RxInfo[0].Snr
	}

	// AC-07: Construct TelemetryPayload
	payload := telemetry.TelemetryPayload{
		DeviceID:        deviceID,
		RawWeight:       uplink.Object.RawWeight,
		BatteryLevel:    uplink.Object.BatteryLevel,
		SampleCount:     uplink.Object.SampleCount,
		RSSI:            rssi,
		SNR:             snr,
		FCnt:            *uplink.FCnt,
		SpreadingFactor: uplink.TxInfo.Modulation.Lora.SpreadingFactor,
		ReceivedAt:      time.Now(),
	}

	// AC-08: Moving average processing
	processedPayload, err := r.processor.Process(ctx, payload)
	if err != nil {
		return fmt.Errorf("processor.Process: %w", err)
	}

	// AC-05: Push valid messages into a buffered channel
	select {
	case r.outChan <- processedPayload:
		logger.Debug().Msg("Payload pushed to downstream processing channel")
	case <-ctx.Done():
		return fmt.Errorf("timeout pushing to downstream channel: %w", ctx.Err())
	}

	return nil
}
