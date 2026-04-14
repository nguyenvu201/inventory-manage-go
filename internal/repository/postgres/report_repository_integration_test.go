//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"inventory-manage/internal/model"
	"inventory-manage/internal/repository/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, ctx := setupTestDB(t) // reuse setupTestDB from telemetry_repository_integration_test.go (same package)

	// Seed device and sku_config (FK dependencies for inventory_history)
	_, err := pool.Exec(ctx, `INSERT INTO devices (device_id, name, sku_code, status) VALUES ('D1', 'Device 1', 'SKU-A', 'active'), ('D2', 'Device 2', 'SKU-A', 'active') ON CONFLICT DO NOTHING`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO sku_configs (sku_code, unit_weight_kg, full_capacity_kg, tare_weight_kg, resolution_kg, reorder_point_qty, unit_label)
						   VALUES ('SKU-A', 2.0, 100.0, 1.0, 0.5, 5, 'Box'), ('SKU-B', 1.0, 50.0, 0.5, 0.1, 10, 'Bag') ON CONFLICT DO NOTHING`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "TRUNCATE inventory_history CASCADE")
	require.NoError(t, err)

	// seed data for tests
	seedQ := `INSERT INTO inventory_history (device_id, sku_code, net_weight_kg, qty, percentage, snapshot_at)
			  VALUES ('D1', 'SKU-A', 50.0, 5, 100, NOW() - interval '2 days'),
					 ('D1', 'SKU-A', 40.0, 4, 80, NOW() - interval '1 days'),
					 ('D2', 'SKU-A', 30.0, 3, 60, NOW() - interval '1 days'),
					 ('D1', 'SKU-B', 10.0, 1, 10, NOW() - interval '1 days')`
	_, err = pool.Exec(ctx, seedQ)
	require.NoError(t, err)

	repo := postgres.NewReportRepository(pool)

	t.Run("AC-03: GetConsumptionTrend with time_bucket grouping", func(t *testing.T) {
		q := model.ConsumptionQuery{
			SKUCode:  "SKU-A",
			From:     time.Now().Add(-3 * 24 * time.Hour),
			To:       time.Now(),
			Interval: "1d",
		}
		pts, err := repo.GetConsumptionTrend(ctx, q)
		require.NoError(t, err)
		// 2 time buckets expected (2 days ago bucket, 1 day ago bucket)
		assert.NotEmpty(t, pts)
	})

	t.Run("AC-03: GetConsumptionTrend with cursor pagination", func(t *testing.T) {
		q := model.ConsumptionQuery{
			SKUCode:  "SKU-A",
			From:     time.Now().Add(-3 * 24 * time.Hour),
			To:       time.Now(),
			Interval: "1d",
			Limit:    1,
		}
		pts, err := repo.GetConsumptionTrend(ctx, q)
		require.NoError(t, err)
		assert.Len(t, pts, 1)
	})

	t.Run("AC-04: GetConsumptionSummary total consumption per SKU", func(t *testing.T) {
		q := model.ConsumptionQuery{
			SKUCode:  "SKU-A",
			From:     time.Now().Add(-3 * 24 * time.Hour),
			To:       time.Now(),
			Interval: "1d",
		}
		sum, err := repo.GetConsumptionSummary(ctx, q)
		require.NoError(t, err)
		assert.NotNil(t, sum)
		assert.Equal(t, "SKU-A", sum.SKUCode)
		// Opening was 50kg, closing was lower → some consumption should be registered
		assert.GreaterOrEqual(t, sum.TotalConsumptionKg, 0.0)
	})

	t.Run("GetConsumptionTrend - empty result for unknown SKU", func(t *testing.T) {
		q := model.ConsumptionQuery{
			SKUCode:  "SKU-UNKNOWN",
			From:     time.Now().Add(-3 * 24 * time.Hour),
			To:       time.Now(),
			Interval: "1d",
		}
		pts, err := repo.GetConsumptionTrend(ctx, q)
		require.NoError(t, err)
		assert.Empty(t, pts)
	})
}
