package postgres

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
)

type InventoryRepository struct {
	db *pgxpool.Pool
}

func NewInventoryRepository(db *pgxpool.Pool) service.IInventoryRepository {
	return &InventoryRepository{db: db}
}

// UpsertSnapshot updates or inserts a new inventory snapshot for the device, and logs it to history.
func (r *InventoryRepository) UpsertSnapshot(ctx context.Context, snapshot *model.InventorySnapshot) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("InventoryRepository.UpsertSnapshot begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	query1, args1, err := sq.Insert("inventory_snapshots").
		Columns("device_id", "sku_code", "net_weight_kg", "qty", "percentage", "snapshot_at").
		Values(snapshot.DeviceID, snapshot.SKUCode, snapshot.NetWeightKg, snapshot.Qty, snapshot.Percentage, sq.Expr("NOW()")).
		Suffix("ON CONFLICT (device_id) DO UPDATE SET " +
			"sku_code = EXCLUDED.sku_code, " +
			"net_weight_kg = EXCLUDED.net_weight_kg, " +
			"qty = EXCLUDED.qty, " +
			"percentage = EXCLUDED.percentage, " +
			"snapshot_at = NOW()").
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("InventoryRepository.UpsertSnapshot build query 1: %w", err)
	}

	_, err = tx.Exec(ctx, query1, args1...)
	if err != nil {
		return fmt.Errorf("InventoryRepository.UpsertSnapshot exec 1: %w", err)
	}

	query2, args2, err := sq.Insert("inventory_history").
		Columns("device_id", "sku_code", "net_weight_kg", "qty", "percentage", "snapshot_at").
		Values(snapshot.DeviceID, snapshot.SKUCode, snapshot.NetWeightKg, snapshot.Qty, snapshot.Percentage, sq.Expr("NOW()")).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("InventoryRepository.UpsertSnapshot build query 2: %w", err)
	}

	_, err = tx.Exec(ctx, query2, args2...)
	if err != nil {
		return fmt.Errorf("InventoryRepository.UpsertSnapshot exec 2: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("InventoryRepository.UpsertSnapshot commit: %w", err)
	}

	return nil
}

// GetSnapshotBySKU returns all current snapshots matching the given sku_code.
func (r *InventoryRepository) GetSnapshotBySKU(ctx context.Context, skuCode string) ([]*model.InventorySnapshot, error) {
	query, args, err := sq.Select("device_id", "sku_code", "net_weight_kg", "qty", "percentage").
		From("inventory_snapshots").
		Where(sq.Eq{"sku_code": skuCode}).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("InventoryRepository.GetSnapshotBySKU build query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("InventoryRepository.GetSnapshotBySKU query: %w", err)
	}
	defer rows.Close()

	var results []*model.InventorySnapshot
	for rows.Next() {
		var s model.InventorySnapshot
		if err := rows.Scan(&s.DeviceID, &s.SKUCode, &s.NetWeightKg, &s.Qty, &s.Percentage); err != nil {
			return nil, fmt.Errorf("InventoryRepository.GetSnapshotBySKU scan: %w", err)
		}
		results = append(results, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("InventoryRepository.GetSnapshotBySKU rows err: %w", err)
	}

	return results, nil
}

// GetCurrentSnapshots returns all active snapshots across all devices.
func (r *InventoryRepository) GetCurrentSnapshots(ctx context.Context) ([]*model.InventorySnapshot, error) {
	query, args, err := sq.Select("device_id", "sku_code", "net_weight_kg", "qty", "percentage").
		From("inventory_snapshots").
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("InventoryRepository.GetCurrentSnapshots build query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("InventoryRepository.GetCurrentSnapshots query: %w", err)
	}
	defer rows.Close()

	var results []*model.InventorySnapshot
	for rows.Next() {
		var s model.InventorySnapshot
		if err := rows.Scan(&s.DeviceID, &s.SKUCode, &s.NetWeightKg, &s.Qty, &s.Percentage); err != nil {
			return nil, fmt.Errorf("InventoryRepository.GetCurrentSnapshots scan: %w", err)
		}
		results = append(results, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("InventoryRepository.GetCurrentSnapshots rows err: %w", err)
	}

	return results, nil
}

// GetSKUConfig returns the configuration for a given SKU.
func (r *InventoryRepository) GetSKUConfig(ctx context.Context, skuCode string) (*model.SKUConfig, error) {
	query, args, err := sq.Select("sku_code", "unit_weight_kg", "full_capacity_kg", "tare_weight_kg", "resolution_kg", "reorder_point_qty", "unit_label").
		From("sku_configs").
		Where(sq.Eq{"sku_code": skuCode}).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("InventoryRepository.GetSKUConfig build query: %w", err)
	}

	var conf model.SKUConfig
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&conf.SKUCode,
		&conf.UnitWeightKg,
		&conf.FullCapacityKg,
		&conf.TareWeightKg,
		&conf.ResolutionKg,
		&conf.ReorderPointQty,
		&conf.UnitLabel,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrSKUNotFound
		}
		return nil, fmt.Errorf("InventoryRepository.GetSKUConfig scan: %w", err)
	}

	return &conf, nil
}
