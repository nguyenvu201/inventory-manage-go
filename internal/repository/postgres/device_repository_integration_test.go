//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"inventory-manage/internal/domain/device"
	"inventory-manage/internal/repository/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Using the helper defined in telemetry_repository_integration_test.go
	// assuming connectDB was made reusable, but if not we can re-create or reuse.
	pool, _ := setupTestDB(t) // Reuse setupTestDB from telemetry_repository_integration_test.go
	
	repo := postgres.NewDeviceRepository(pool)

	// Clean up devices table before tests
	pool.Exec(ctx, "TRUNCATE devices CASCADE")

	t.Run("AC-02: Save new device", func(t *testing.T) {
		d := &device.Device{
			DeviceID:  "DEV-001",
			Name:      "Scale 1",
			SKUCode:   "SKU-A",
			Location:  "Warehouse A",
			Status:    device.StatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.Save(ctx, d)
		require.NoError(t, err)

		// Assert it exists
		saved, err := repo.FindByID(ctx, "DEV-001")
		require.NoError(t, err)
		assert.Equal(t, "DEV-001", saved.DeviceID)
		assert.Equal(t, "Scale 1", saved.Name)
	})

	t.Run("AC-06: Duplicate device ID returns ErrDuplicateDevice", func(t *testing.T) {
		d := &device.Device{
			DeviceID:  "DEV-001", // duplicate
			Name:      "Scale Duplicate",
			SKUCode:   "SKU-B",
			Status:    device.StatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.Save(ctx, d)
		require.ErrorIs(t, err, device.ErrDuplicateDevice)
	})

	t.Run("AC-04: Find NotFound returns ErrDeviceNotFound", func(t *testing.T) {
		_, err := repo.FindByID(ctx, "DEV-999")
		require.ErrorIs(t, err, device.ErrDeviceNotFound)
	})

	t.Run("AC-03: FindAll with filters", func(t *testing.T) {
		// Insert a second device
		d2 := &device.Device{
			DeviceID:  "DEV-002",
			Name:      "Scale 2",
			SKUCode:   "SKU-B",
			Status:    device.StatusInactive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, repo.Save(ctx, d2))

		// Find All
		all, err := repo.FindAll(ctx, device.DeviceQuery{})
		require.NoError(t, err)
		assert.Len(t, all, 2)

		// Find by Status
		inactive := device.StatusInactive
		filtered, err := repo.FindAll(ctx, device.DeviceQuery{Status: &inactive})
		require.NoError(t, err)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "DEV-002", filtered[0].DeviceID)

		// Find by SKU
		sku := "SKU-A"
		filteredSKU, err := repo.FindAll(ctx, device.DeviceQuery{SKUCode: &sku})
		require.NoError(t, err)
		assert.Len(t, filteredSKU, 1)
		assert.Equal(t, "DEV-001", filteredSKU[0].DeviceID)
	})

	t.Run("AC-05: Update device", func(t *testing.T) {
		d, err := repo.FindByID(ctx, "DEV-001")
		require.NoError(t, err)

		d.Name = "Scale 1 Updated"
		d.Status = device.StatusMaintenance
		err = repo.Update(ctx, d)
		require.NoError(t, err)

		updated, err := repo.FindByID(ctx, "DEV-001")
		require.NoError(t, err)
		assert.Equal(t, "Scale 1 Updated", updated.Name)
		assert.Equal(t, device.StatusMaintenance, updated.Status)
	})

	t.Run("AC-05: Delete device", func(t *testing.T) {
		err := repo.Delete(ctx, "DEV-002")
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, "DEV-002")
		require.ErrorIs(t, err, device.ErrDeviceNotFound)

		// Delete non-existent
		err = repo.Delete(ctx, "DEV-002")
		require.ErrorIs(t, err, device.ErrDeviceNotFound)
	})
}
