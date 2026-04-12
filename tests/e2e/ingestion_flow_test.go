//go:build e2e

// Package e2e implements end-to-end flow tests for the Inventory Management System.
// These tests verify complete system flows from input (MQTT uplink) to output (API response).
//
// E2E Test Coverage (Sprint 1):
//   - TestE2E_TelemetryIngestionFlow: MQTT → Validator → DB → API (TASK-002/003/004)
//
// Requirements:
//   - make docker-up (all infrastructure services healthy)
//   - make run OR app binary started (HTTP on :8080)
//   - Migrations applied (make migrate)
//
// Usage:
//
//	make test-e2e
//	go test -tags e2e ./tests/e2e/... -v -timeout 120s
package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func baseURL() string { return getEnv("BASE_URL", "http://localhost:8080") }

// ── Placeholder E2E Tests (to be completed in TASK-002 / TASK-004) ───────────

// TestE2E_HealthFlow verifies the basic service flow is operational.
// This is a placeholder — full ingestion E2E will be added in TASK-002 + TASK-004.
// Task: INV-SPR01-TASK-001 (infrastructure baseline)
func TestE2E_HealthFlow(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	// Step 1: Verify service is up
	resp, err := client.Get(baseURL() + "/health")
	require.NoError(t, err, "service must be running for E2E tests")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])

	t.Log("E2E baseline: service healthy ✓")
	t.Log("Full ingestion E2E (MQTT → DB → API) will be implemented in TASK-002/TASK-004")
}

// TestE2E_TelemetryIngestionFlow is the primary E2E test for Sprint 1 completion.
// It will be fully implemented during TASK-002 (Gateway) + TASK-004 (Raw Storage).
//
// Flow:
//  1. Publish MQTT uplink (simulate ChirpStack gateway)
//  2. Poll DB until record appears (max 5s)
//  3. Verify stored data matches payload
//  4. Test idempotency: duplicate f_cnt → record count stays 1
//  5. Verify API response contains the record
//
// Task Coverage: INV-SPR01-TASK-002 (AC-01,02), TASK-003 (AC-01,02,03), TASK-004 (AC-01,03)
func TestE2E_TelemetryIngestionFlow(t *testing.T) {
	t.Skip("TODO: implement after TASK-002 (Gateway) + TASK-004 (Raw Storage) are completed")

	ctx := context.Background()
	deviceID := fmt.Sprintf("E2E-SCALE-%d", time.Now().UnixNano())

	// TODO TASK-002: publish MQTT uplink
	_ = ctx
	_ = deviceID

	// TODO TASK-004: poll DB for record
	// require.Eventually(t, func() bool { ... }, 5*time.Second, 200*time.Millisecond)

	// TODO: verify data, idempotency, API response
}
