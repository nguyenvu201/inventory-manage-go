package postgres

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
)

type ReportRepository struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) service.IReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) GetConsumptionTrend(ctx context.Context, query model.ConsumptionQuery) ([]*model.ConsumptionDataPoint, error) {
	intervalStr := "1 hour"
	if query.Interval == "1d" {
		intervalStr = "1 day"
	} else if query.Interval == "1w" {
		intervalStr = "1 week"
	}

	bucketExpr := fmt.Sprintf("time_bucket('%s', snapshot_at) AS bucket", intervalStr)

	stmt := sq.Select(bucketExpr,
		"AVG(net_weight_kg) AS avg_net_weight_kg",
		"AVG(qty) AS avg_qty",
		"AVG(percentage) AS avg_percentage").
		From("inventory_history").
		Where(sq.Eq{"sku_code": query.SKUCode}).
		Where(sq.GtOrEq{"snapshot_at": query.From}).
		Where(sq.LtOrEq{"snapshot_at": query.To})

	if query.Cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339, query.Cursor)
		if err == nil {
			stmt = stmt.Where(sq.Gt{"snapshot_at": cursorTime})
		}
	}

	stmt = stmt.GroupBy("bucket").OrderBy("bucket ASC")

	if query.Limit > 0 {
		stmt = stmt.Limit(uint64(query.Limit))
	}

	sql, args, err := stmt.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("ReportRepository.GetConsumptionTrend build query: %w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("ReportRepository.GetConsumptionTrend query: %w", err)
	}
	defer rows.Close()

	var points []*model.ConsumptionDataPoint
	for rows.Next() {
		var p model.ConsumptionDataPoint
		if err := rows.Scan(&p.Timestamp, &p.NetWeightKg, &p.Qty, &p.Percentage); err != nil {
			return nil, fmt.Errorf("ReportRepository.GetConsumptionTrend scan: %w", err)
		}
		points = append(points, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ReportRepository.GetConsumptionTrend rows err: %w", err)
	}

	return points, nil
}

func (r *ReportRepository) GetConsumptionSummary(ctx context.Context, query model.ConsumptionQuery) (*model.ConsumptionSummary, error) {
	// A summary requires fetching the earliest reading and the latest reading in the window,
	// and potentially summing all negative weight deltas over consecutive readings to find TotalConsumption.
	// Since calculating exact accurate deltas is complex in SQL without window functions,
	// we will calculate the sum of drops using the LAG() window function.
	
	// CTE to get state at each interval
	cteSql := `
	WITH ordered_snapshots AS (
		SELECT snapshot_at, net_weight_kg, qty,
		       LAG(net_weight_kg) OVER (ORDER BY snapshot_at ASC) as prev_weight,
		       LAG(qty) OVER (ORDER BY snapshot_at ASC) as prev_qty
		FROM inventory_history
		WHERE sku_code = $1 AND snapshot_at >= $2 AND snapshot_at <= $3
	)
	SELECT 
		COALESCE(SUM(GREATEST(prev_weight - net_weight_kg, 0)), 0) as total_consumption_kg,
		(SELECT qty FROM ordered_snapshots ORDER BY snapshot_at ASC LIMIT 1) as opening_qty,
		(SELECT qty FROM ordered_snapshots ORDER BY snapshot_at DESC LIMIT 1) as closing_qty,
		(SELECT net_weight_kg FROM ordered_snapshots ORDER BY snapshot_at ASC LIMIT 1) as opening_weight_kg,
		(SELECT net_weight_kg FROM ordered_snapshots ORDER BY snapshot_at DESC LIMIT 1) as closing_weight_kg
	FROM ordered_snapshots;`

	summary := &model.ConsumptionSummary{SKUCode: query.SKUCode}
	
	err := r.db.QueryRow(ctx, cteSql, query.SKUCode, query.From, query.To).Scan(
		&summary.TotalConsumptionKg,
		&summary.OpeningQty,
		&summary.ClosingQty,
		&summary.OpeningWeightKg,
		&summary.ClosingWeightKg,
	)

	if err != nil {
		// If there are no rows or all null
		if err.Error() == "no rows in result set" || err.Error() == "sql: Scan error on column index 1: converting NULL to int is unsupported" {
			return summary, nil // empty window, all 0
		}
		return nil, fmt.Errorf("ReportRepository.GetConsumptionSummary scan: %w", err)
	}

	return summary, nil
}
