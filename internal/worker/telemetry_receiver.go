package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"inventory-manage/global"
	"inventory-manage/internal/domain/telemetry"
	"inventory-manage/internal/model"
	inventorymqtt "inventory-manage/internal/platform/mqtt"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// log returns the global logger if initialised, or a no-op logger for tests.
func log() *zap.Logger {
	if global.Logger != nil {
		return global.Logger.Logger
	}
	return zap.NewNop()
}


// TelemetryReceiver sits at the boundary, consuming MQTT messages, verifying their structure,
// passing them to the business rules processor, and feeding them to the downstream channel.
type TelemetryReceiver struct {
	client    *inventorymqtt.Client
	processor *telemetry.Processor
	validator *telemetry.Validator
	outChan   chan<- model.TelemetryPayload
}

// NewTelemetryReceiver constructs the gateway message receiver.
func NewTelemetryReceiver(
	client *inventorymqtt.Client,
	processor *telemetry.Processor,
	validator *telemetry.Validator,
	outChan chan<- model.TelemetryPayload,
) *TelemetryReceiver {
	return &TelemetryReceiver{
		client:    client,
		processor: processor,
		validator: validator,
		outChan:   outChan,
	}
}

// Start begins subscribing to the MQTT topic.
func (r *TelemetryReceiver) Start() error {
	topic := "application/+/device/+/event/up"
	log().Info("Starting telemetry receiver subscription", zap.String("topic", topic))

	if err := r.client.Subscribe(topic, 1, r.handleMessage); err != nil {
		return fmt.Errorf("telemetryReceiver.Start: %w", err)
	}
	return nil
}

// handleMessage is the callback from the Paho MQTT client.
func (r *TelemetryReceiver) handleMessage(client paho.Client, msg paho.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.ProcessPayload(ctx, msg.Payload()); err != nil {
		log().Error("failed to process telemetry MQTT message", zap.Error(err))
	}
}

// ProcessPayload handles the raw byte payload parsing and routing. Safe for unit testing.
func (r *TelemetryReceiver) ProcessPayload(ctx context.Context, raw []byte) error {
	traceID := uuid.New().String()

	var uplink model.ChirpStackUplink
	if err := json.Unmarshal(raw, &uplink); err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}

	deviceID := uplink.DeviceInfo.DevEui
	if deviceID == "" {
		return fmt.Errorf("device_id missing from payload")
	}

	log().Info("Received telemetry message from gateway",
		zap.String("device_id", deviceID),
		zap.String("trace_id", traceID),
	)

	if uplink.FCnt == nil {
		return fmt.Errorf("f_cnt is missing in payload")
	}

	var rssi int
	var snr float32
	if len(uplink.RxInfo) > 0 {
		rssi = uplink.RxInfo[0].Rssi
		snr = uplink.RxInfo[0].Snr
	}

	rawWeight := uplink.Object.RawWeight
	battery := uplink.Object.BatteryLevel
	sampleCount := uplink.Object.SampleCount

	// Fallback: Base64 decoder if Object is unparsed
	if sampleCount == 0 && uplink.Data != "" {
		w, b, s, decErr := telemetry.DecodeBase64Payload(uplink.Data)
		if decErr == nil {
			rawWeight = w
			battery = b
			sampleCount = s
		} else {
			log().Warn("failed to decode fallback base64 payload",
				zap.String("device_id", deviceID),
				zap.String("trace_id", traceID),
				zap.Error(decErr),
			)
		}
	}

	// Build the domain TelemetryPayload (still using domain validator/processor)
	domainPayload := telemetry.TelemetryPayload{
		DeviceID:        deviceID,
		RawWeight:       rawWeight,
		BatteryLevel:    battery,
		SampleCount:     sampleCount,
		RSSI:            rssi,
		SNR:             snr,
		FCnt:            *uplink.FCnt,
		SpreadingFactor: uplink.TxInfo.Modulation.Lora.SpreadingFactor,
		ReceivedAt:      time.Now(),
	}

	if err := r.validator.Validate(domainPayload); err != nil {
		log().Error("telemetry payload validation failed",
			zap.String("device_id", deviceID),
			zap.String("trace_id", traceID),
			zap.Error(err),
		)
		return fmt.Errorf("payload validation error: %w", err)
	}

	processedDomain, err := r.processor.Process(ctx, domainPayload)
	if err != nil {
		return fmt.Errorf("processor.Process: %w", err)
	}

	// Convert domain payload → model.TelemetryPayload for the model-based pipeline
	payload := model.TelemetryPayload{
		DeviceID:        processedDomain.DeviceID,
		RawWeight:       processedDomain.RawWeight,
		BatteryLevel:    processedDomain.BatteryLevel,
		RSSI:            processedDomain.RSSI,
		SNR:             processedDomain.SNR,
		FCnt:            processedDomain.FCnt,
		SpreadingFactor: processedDomain.SpreadingFactor,
		SampleCount:     processedDomain.SampleCount,
		ReceivedAt:      processedDomain.ReceivedAt,
	}

	select {
	case r.outChan <- payload:
		log().Debug("Payload pushed to downstream processing channel",
			zap.String("device_id", deviceID),
			zap.String("trace_id", traceID),
		)
	case <-ctx.Done():
		return fmt.Errorf("timeout pushing to downstream channel: %w", ctx.Err())
	}

	return nil
}
