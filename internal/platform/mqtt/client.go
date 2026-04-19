package mqtt

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Client wraps the Paho MQTT client to manage lifecycle and connections.
type Client struct {
	client paho.Client
}

// NewClientOptions configures the MQTT client options with auto-reconnect and exponential backoff (AC-03).
// tlsEnabled=true → dùng ssl:// scheme + TLS config (bắt buộc với HiveMQ Cloud port 8883).
func NewClientOptions(host string, port int, clientID, username, password string, tlsEnabled bool) *paho.ClientOptions {
	opts := paho.NewClientOptions()

	// Chọn scheme: ssl:// cho TLS (HiveMQ Cloud), tcp:// cho plain MQTT (local Mosquitto)
	scheme := "tcp"
	if tlsEnabled {
		scheme = "ssl"
		opts.SetTLSConfig(&tls.Config{
			// HiveMQ Cloud dùng certificate hợp lệ từ Let's Encrypt
			// → InsecureSkipVerify=false (mặc định) là đúng, không cần override
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		})
	}
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", scheme, host, port))
	opts.SetClientID(clientID)
	
	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}
	// Will message: broker publish khi client mất kết nối đột ngột
	// Hữu ích để các service khác biết ingestion service bị down
	opts.SetWill(
		"inventory/status/"+clientID,
		`{"status":"offline"}`,
		1,    // QoS 1
		true, // retain
	)

	opts.SetAutoReconnect(true)
	// Exponential backoff: bắt đầu từ 1s, tăng dần tới 30s
	// Tránh reconnect storm khi broker restart
	opts.SetConnectRetryInterval(1 * time.Second)
	opts.SetMaxReconnectInterval(30 * time.Second)
	opts.SetKeepAlive(30 * time.Second)  // phải nhỏ hơn broker's max_keepalive
	opts.SetPingTimeout(10 * time.Second)
	// CleanSession=false: giữ lại subscriptions sau khi reconnect
	// Broker nhớ session → không cần re-subscribe thủ công
	opts.SetCleanSession(false)
	// Thêm suffix ngẫu nhiên vào clientID để tránh duplicate client conflict
	// (2 instance cùng clientID → broker kick cái cũ → reconnect loop)
	opts.SetClientID(clientID + "-" + uuid.New().String()[:8])

	opts.SetOnConnectHandler(func(c paho.Client) {
		log.Info().Msg("MQTT broker connected")
	})

	opts.SetConnectionLostHandler(func(c paho.Client, err error) {
		log.Error().Err(err).Msg("MQTT broker connection lost, attempting reconnect...")
	})

	return opts
}

// NewClient initializes a new MQTT client using the provided connection settings.
// tlsEnabled: true khi kết nối tới cloud broker (HiveMQ Cloud, port 8883).
func NewClient(host string, port int, clientID, username, password string, tlsEnabled bool) (*Client, error) {
	opts := NewClientOptions(host, port, clientID, username, password, tlsEnabled)
	client := paho.NewClient(opts)
	return &Client{client: client}, nil
}

// Connect establishes the connection to the MQTT broker, waiting until context is done or connected.
func (c *Client) Connect(ctx context.Context) error {
	token := c.client.Connect()
	
	// Wait for token or context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("mqtt connect cancelled: %w", ctx.Err())
	case <-token.Done():
		if err := token.Error(); err != nil {
			return fmt.Errorf("mqtt connect failed: %w", err)
		}
	}
	return nil
}

// Disconnect closes the connection gracefully.
func (c *Client) Disconnect() {
	if c.client.IsConnected() {
		c.client.Disconnect(250)
		log.Info().Msg("MQTT broker disconnected")
	}
}

// Subscribe wraps the subscription function.
func (c *Client) Subscribe(topic string, qos byte, callback paho.MessageHandler) error {
	token := c.client.Subscribe(topic, qos, callback)
	token.Wait()
	if err := token.Error(); err != nil {
		return fmt.Errorf("mqtt subscribe failed for topic %s: %w", topic, err)
	}
	return nil
}
