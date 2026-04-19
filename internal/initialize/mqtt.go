package initialize

import (
	"context"
	"fmt"
	"time"

	"inventory-manage/global"
	inventorymqtt "inventory-manage/internal/platform/mqtt"

	"go.uber.org/zap"
)

// MQTTClient is exported so the MQTT worker can be started after DI is complete.
var MQTTClient *inventorymqtt.Client

// InitMQTT creates and connects the Paho MQTT client.
// Connection errors at startup are logged as warnings (not fatal) because
// the Paho client has built-in auto-reconnect logic.
func InitMQTT() {
	m := global.Config.MQTT

	client, err := inventorymqtt.NewClient(
		m.Broker, m.Port, m.ClientID, m.Username, m.Password,
		m.TLSEnabled,
	)
	if err != nil {
		loggerOrPanic("InitMQTT: failed to create MQTT client", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		// Not fatal — Paho will auto-reconnect
		global.Logger.Warn("InitMQTT: broker unavailable at startup, auto-reconnect enabled",
			zap.String("broker", fmt.Sprintf("%s:%d", m.Broker, m.Port)),
			zap.Error(err),
		)
	} else {
		global.Logger.Info("MQTT broker connected",
			zap.String("broker", fmt.Sprintf("%s:%d", m.Broker, m.Port)),
			zap.String("client_id", m.ClientID),
		)
	}

	MQTTClient = client
}
