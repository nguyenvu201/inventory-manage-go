// Package mqtt_test implements tests for INV-SPR01-TASK-002
// AC Coverage:
//   AC-01: TestClient_Connect_Disconnect
//   AC-03: TestClient_ConfiguresBackoff
// IEC 62304 Classification: Software Safety Class B
package mqtt_test

import (
	"context"
	"testing"
	"time"

	inventorymqtt "inventory-manage/internal/platform/mqtt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// In a real environment, we would use testcontainers to spin up Mosquitto.
// For AC-01 configuration check, we inspect the client options.
func TestClient_ConfiguresBackoff(t *testing.T) {
	opts := inventorymqtt.NewClientOptions("localhost", 1883, "test-client", "", "")

	// Verify backoff options (AC-03)
	assert.True(t, opts.AutoReconnect, "AC-03: AutoReconnect must be enabled")
	assert.Equal(t, 2*time.Second, opts.MaxReconnectInterval, "AC-03: Backoff interval must be configured")
	assert.NotNil(t, opts.OnConnectionLost, "AC-03: OnConnectionLost callback must be set")
}

// TestClient_Connect_Disconnect does a live test if broker is available, otherwise skipped via short.
func TestClient_Connect_Disconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live MQTT connect test in short mode")
	}

	client, err := inventorymqtt.NewClient(
		"localhost", 1883, "test-connection-client", "", "",
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	require.NoError(t, err, "AC-01: Must connect successfully to local broker")

	client.Disconnect()
}
