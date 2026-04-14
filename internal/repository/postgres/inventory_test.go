//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inventory-manage/internal/model"
	"inventory-manage/internal/repository/postgres"
)

func TestInventoryRepository_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, ctx := setupTestDB(t)
	defer pool.Close()

	repo := postgres.NewInventoryRepository(pool)
	deviceRepo := postgres.NewDeviceRepository(pool)

	// Setup: insert dependencies (device and sku config)
	err := deviceRepo.Save(ctx, &model.Device{
		DeviceID: "DEV-INV-1",
		Name:     "Inventory Test Scale",
		Status:   model.StatusActive,
	})
	require.NoError(t, err)

	// Directly insert sku_config using pool just for testing foreign keys
	_, err = pool.Exec(ctx, `
		INSERT INTO sku_configs (sku_code, unit_weight_kg, full_capacity_kg, tare_weight_kg, resolution_kg, reorder_point_qty, unit_label)
		VALUES ('SKU-TEST', 2.0, 50.0, 1.0, 0.5, 5, 'Box')
	`)
	require.NoError(t, err)

	t.Run("AC-06: UpsertSnapshot - Insert new", func(t *testing.T) {
		snap := &model.InventorySnapshot{
			DeviceID:    "DEV-INV-1",
			SKUCode:     "SKU-TEST",
			NetWeightKg: 20.0,
			Qty:         10,
			Percentage:  40.0,
		}

		err := repo.UpsertSnapshot(ctx, snap)
		require.NoError(t, err)

		results, err := repo.GetCurrentSnapshots(ctx)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, 10, results[0].Qty)
	})

	t.Run("AC-06: UpsertSnapshot - Update existing", func(t *testing.T) {
		// Minor delay to ensure snapshot_at differs slightly if we wanted to test time
		time.Sleep(10 * time.Millisecond)

		snap := &model.InventorySnapshot{
			DeviceID:    "DEV-INV-1",
			SKUCode:     "SKU-TEST",
			NetWeightKg: 10.0,
			Qty:         5,
			Percentage:  20.0,
		}

		err := repo.UpsertSnapshot(ctx, snap)
		require.NoError(t, err)

		results, err := repo.GetSnapshotBySKU(ctx, "SKU-TEST")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, 5, results[0].Qty)
		assert.Equal(t, 20.0, results[0].Percentage)
	})

	t.Run("GetSKUConfig", func(t *testing.T) {
		config, err := repo.GetSKUConfig(ctx, "SKU-TEST")
		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, 2.0, config.UnitWeightKg)
		assert.Equal(t, 0.5, config.ResolutionKg)
	})
	
	t.Run("GetSKUConfig - NotFound", func(t *testing.T) {
		config, err := repo.GetSKUConfig(ctx, "SKU-UNKNOWN")
		require.ErrorIs(t, err, model.ErrSKUNotFound)
		require.Nil(t, config)
	})
}
