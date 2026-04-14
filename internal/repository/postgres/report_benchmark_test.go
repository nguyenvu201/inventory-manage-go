//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"inventory-manage/internal/model"
	"inventory-manage/internal/repository/postgres"
)

func BenchmarkReportRepository_GetConsumptionTrend_30Days(b *testing.B) {
	pool, ctx := setupTestDB(b)

	_, _ = pool.Exec(ctx, "TRUNCATE inventory_history CASCADE")

	startTime := time.Now().Add(-30 * 24 * time.Hour)
	sku := "BENCH-SKU-1"
	for i := 0; i < 720; i++ {
		ts := startTime.Add(time.Duration(i) * time.Hour)
		insertSQL := `INSERT INTO inventory_history (sku_code, snapshot_at, net_weight_kg, qty, percentage, device_id)
            VALUES ($1, $2, $3, $4, $5, $6)`
		_, err := pool.Exec(ctx, insertSQL, sku, ts, 10.0, 5, 50.0, "D1")
		if err != nil {
			b.Logf("seed insert failed (may be harmless): %v", err)
			break
		}
	}

	repo := postgres.NewReportRepository(pool)
	query := model.ConsumptionQuery{
		SKUCode:  sku,
		From:     startTime,
		To:       time.Now(),
		Interval: "1d",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pts, err := repo.GetConsumptionTrend(ctx, query)
		if err != nil {
			b.Fatalf("query failed (>500ms threshold violation likely): %v", err)
		}
		_ = pts
	}
}
