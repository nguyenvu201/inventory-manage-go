package telemetry

import "time"

// ChirpStackUplink represents the JSON payload pushed by ChirpStack v4
// (application/+/device/+/event/up).
type ChirpStackUplink struct {
	DeviceInfo struct {
		TenantID        string `json:"tenantId"`
		TenantName      string `json:"tenantName"`
		ApplicationID   string `json:"applicationId"`
		ApplicationName string `json:"applicationName"`
		DeviceProfileID string `json:"deviceProfileId"`
		DeviceProfileName string `json:"deviceProfileName"`
		DeviceName      string `json:"deviceName"`
		DevEui          string `json:"devEui"`
	} `json:"deviceInfo"`
	DevAddr string `json:"devAddr"`
	Adr     bool   `json:"adr"`
	Dr      int     `json:"dr"`
	FCnt    *uint32 `json:"fCnt"`
	FPort   int     `json:"fPort"`
	Data    string `json:"data"`
	Object  struct {
		RawWeight    float64 `json:"raw_weight"`
		BatteryLevel int8    `json:"battery_level"`
		SampleCount  int     `json:"sample_count"`
	} `json:"object"`
	RxInfo []struct {
		GatewayID string  `json:"gatewayId"`
		Rssi      int     `json:"rssi"`
		Snr       float32 `json:"snr"`
	} `json:"rxInfo"`
	TxInfo struct {
		Frequency int `json:"frequency"`
		Modulation struct {
			Lora struct {
				Bandwidth       int `json:"bandwidth"`
				SpreadingFactor int `json:"spreadingFactor"`
				CodeRate        string `json:"codeRate"`
			} `json:"lora"`
		} `json:"modulation"`
	} `json:"txInfo"`
}

// TelemetryPayload represents the normalized payload entering the processing pipeline.
type TelemetryPayload struct {
	DeviceID        string  `validate:"required"`
	RawWeight       float64 `validate:"gte=0"`
	BatteryLevel    int8    `validate:"gte=0,lte=100"`
	RSSI            int
	SNR             float32
	FCnt            uint32
	SpreadingFactor int
	SampleCount     int
	ReceivedAt      time.Time
}
