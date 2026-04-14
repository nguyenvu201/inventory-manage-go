//go:build integration

package postgres_test

import (
	"inventory-manage/internal/model"
)

import (
	"context"
	"testing"
	"time"

	"inventory-manage/internal/service"
	"inventory-manage/internal/repository/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareDeviceForCalibration(t *testing.T, repo service.IDeviceRepository, devID string) {
	err := repo.Save(context.Background(), &model.Device{
		DeviceID: devID,
		Name:     "Test Scale",
		SKUCode:  "SKU-1",
		Status:   model.StatusActive,
	})
	require.NoError(t, err)
}

func TestCalibrationRepository_SaveAndGetActive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dbPool, _ := setupTestDB(t)
	repo := postgres.NewCalibrationRepository(dbPool)
	devRepo := postgres.NewDeviceRepository(dbPool)
	ctx := context.Background()

	devID := "CALIB-TEST-01"
	prepareDeviceForCalibration(t, devRepo, devID)

	t.Run("Save initial config", func(t *testing.T) {
		cfg := &model.CalibrationConfig{
			DeviceID:       devID,
			ZeroValue:      0.01,
			SpanValue:      500.0,
			Unit:           "kg",
			CapacityMax:    1000.0,
			HardwareConfig: map[string]interface{}{"adc_rate": 80},
			CreatedBy:      "admin",
		}

		err := repo.Save(ctx, cfg)
		require.NoError(t, err)
		assert.NotZero(t, cfg.ID)
		assert.False(t, cfg.EffectiveFrom.IsZero())

		active, err := repo.GetActive(ctx, devID)
		require.NoError(t, err)
		assert.Equal(t, cfg.ID, active.ID)
		assert.Equal(t, 500.0, active.SpanValue)
	})

	t.Run("Save second config deactivates first", func(t *testing.T) {
		// Verify first is active
		first, err := repo.GetActive(ctx, devID)
		require.NoError(t, err)

		// Create second
		cfg2 := &model.CalibrationConfig{
			DeviceID:       devID,
			ZeroValue:      0.02,
			SpanValue:      501.0,
			Unit:           "kg",
			CapacityMax:    1000.0,
			HardwareConfig: map[string]interface{}{"adc_rate": 10},
			CreatedBy:      "system",
		}
		time.Sleep(10 * time.Millisecond) // ensure timestamp diff
		err = repo.Save(ctx, cfg2)
		require.NoError(t, err)

		second, err := repo.GetActive(ctx, devID)
		require.NoError(t, err)

		assert.NotEqual(t, first.ID, second.ID)
		assert.Equal(t, 501.0, second.SpanValue)

		// Direct query to check deactivated_at of first
		var deactivatedAt *time.Time
		err = dbPool.QueryRow(ctx, "SELECT deactivated_at FROM calibration_configs WHERE id = $1", first.ID).Scan(&deactivatedAt)
		require.NoError(t, err)
		assert.NotNil(t, deactivatedAt)
	})

	t.Run("Foreign key violation", func(t *testing.T) {
		cfg := &model.CalibrationConfig{
			DeviceID:    "NON-EXISTENT",
			ZeroValue:   0,
			SpanValue:   1,
			Unit:        "kg",
			CapacityMax: 10,
			HardwareConfig: map[string]interface{}{"adc_rate": 80},
			CreatedBy:   "admin",
		}
		err := repo.Save(ctx, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "device not found")
	})
}
