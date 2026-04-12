package mqtt

import (
	"context"
	"fmt"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog/log"
)

// Client wraps the Paho MQTT client to manage lifecycle and connections.
type Client struct {
	client paho.Client
}

// NewClientOptions configures the MQTT client options with auto-reconnect and exponential backoff (AC-03).
func NewClientOptions(host string, port int, clientID, username, password string) *paho.ClientOptions {
	opts := paho.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", host, port))
	opts.SetClientID(clientID)
	
	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}

	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(2 * time.Second)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	opts.SetOnConnectHandler(func(c paho.Client) {
		log.Info().Msg("MQTT broker connected")
	})

	opts.SetConnectionLostHandler(func(c paho.Client, err error) {
		log.Error().Err(err).Msg("MQTT broker connection lost, attempting reconnect...")
	})

	return opts
}

// NewClient initializes a new MQTT client using the provided connection settings.
func NewClient(host string, port int, clientID, username, password string) (*Client, error) {
	opts := NewClientOptions(host, port, clientID, username, password)
	
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
