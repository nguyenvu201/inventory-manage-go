package telemetry

import "time"

// RawTelemetry represents a single validated telemetry packet from a scale node.
// This is the immutable raw data entity — never updated after storage.
type RawTelemetry struct {
	ID             int64
	DeviceID       string
	RawWeight      float64
	BatteryLevel   int8
	// LoRaWAN metadata from ChirpStack uplink
	RSSI           *int16
	SNR            *float32
	FCnt           *uint32  // frame counter — used for idempotent ingestion
	SpreadingFactor *int8
	SampleCount    int      // number of HX711 readings averaged on the node
	// Payload archive
	PayloadJSON    []byte
	// Timestamps
	ReceivedAt     time.Time
	DeviceTime     *time.Time // timestamp reported by the device, if available
}

// TelemetryQuery defines filter parameters for querying telemetry records.
type TelemetryQuery struct {
	DeviceID string
	From     time.Time
	To       time.Time
	Limit    int
	Offset   int
}
