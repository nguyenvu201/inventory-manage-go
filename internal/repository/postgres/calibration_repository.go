package postgres

import (
	"context"
	"errors"
	"fmt"

	"inventory-manage/internal/model"
	"inventory-manage/internal/service"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type calibrationRepository struct {
	db *pgxpool.Pool
	psql sq.StatementBuilderType
}

func NewCalibrationRepository(db *pgxpool.Pool) service.ICalibrationRepository {
	return &calibrationRepository{
		db:   db,
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

// Save inserts a new configuration and implicitly deactivates the previous active one.
func (r *calibrationRepository) Save(ctx context.Context, config *model.CalibrationConfig) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db.Begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Deactivate any existing active configs for this device
	updateQ, updateArgs, err := r.psql.
		Update("calibration_configs").
		Set("deactivated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"device_id": config.DeviceID}).
		Where(sq.Expr("deactivated_at IS NULL")).
		ToSql()

	if err == nil {
		_, err = tx.Exec(ctx, updateQ, updateArgs...)
		if err != nil {
			return fmt.Errorf("deactivating old config: %w", err)
		}
	}

	// Insert the new active config
	insertQ, insertArgs, err := r.psql.
		Insert("calibration_configs").
		Columns("device_id", "zero_value", "span_value", "unit", "capacity_max", "hardware_config", "created_by").
		Values(config.DeviceID, config.ZeroValue, config.SpanValue, config.Unit, config.CapacityMax, config.HardwareConfig, config.CreatedBy).
		Suffix("RETURNING id, effective_from, created_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("building insert query: %w", err)
	}

	err = tx.QueryRow(ctx, insertQ, insertArgs...).Scan(&config.ID, &config.EffectiveFrom, &config.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// This means somehow deactivated_at wasn't set locally on another concurrent transaction
			return fmt.Errorf("concurrent insert violation: %w", err)
		}
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return fmt.Errorf("device not found: %w", err) // Foreign key constraint violation on devices
		}
		return fmt.Errorf("insert config: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *calibrationRepository) GetActive(ctx context.Context, deviceID string) (*model.CalibrationConfig, error) {
	q, args, err := r.psql.
		Select("id", "device_id", "zero_value", "span_value", "unit", "capacity_max", "hardware_config", "effective_from", "created_by", "created_at").
		From("calibration_configs").
		Where(sq.Eq{"device_id": deviceID}).
		Where(sq.Expr("deactivated_at IS NULL")).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("building get_active query: %w", err)
	}

	var c model.CalibrationConfig
	err = r.db.QueryRow(ctx, q, args...).Scan(
		&c.ID, &c.DeviceID, &c.ZeroValue, &c.SpanValue, &c.Unit, &c.CapacityMax, &c.HardwareConfig, &c.EffectiveFrom, &c.CreatedBy, &c.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no active config: %w", model.ErrDeviceNotFound)
		}
		return nil, fmt.Errorf("querying active config: %w", err)
	}

	return &c, nil
}

// UpdateCalibrationTx implements the 4 sequential steps for calibration updates in a single transaction.
func (r *calibrationRepository) UpdateCalibrationTx(ctx context.Context, deviceID string, config *model.CalibrationConfig) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Step 1: Verify and lock device to prevent orphaned records or concurrent races
	var devID string
	err = tx.QueryRow(ctx, "SELECT device_id FROM devices WHERE device_id = $1 FOR UPDATE", deviceID).Scan(&devID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("device verification: %w", model.ErrDeviceNotFound)
		}
		return fmt.Errorf("locking device: %w", err)
	}

	// Step 2: Retrieve and lock the current active configuration
	var activeConfigID int
	err = tx.QueryRow(ctx, "SELECT id FROM calibration_configs WHERE device_id = $1 AND deactivated_at IS NULL FOR UPDATE", deviceID).Scan(&activeConfigID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("locking active config: %w", err)
	}

	// Step 3: Deactivate old configuration if it exists
	if activeConfigID > 0 {
		_, err = tx.Exec(ctx, "UPDATE calibration_configs SET deactivated_at = NOW() WHERE id = $1", activeConfigID)
		if err != nil {
			return fmt.Errorf("deactivating old config: %w", err)
		}
	}

	// Step 4: Insert new configuration
	insertQ, insertArgs, err := r.psql.
		Insert("calibration_configs").
		Columns("device_id", "zero_value", "span_value", "unit", "capacity_max", "hardware_config", "calibration_type", "created_by").
		Values(config.DeviceID, config.ZeroValue, config.SpanValue, config.Unit, config.CapacityMax, config.HardwareConfig, config.CalibrationType, config.CreatedBy).
		Suffix("RETURNING id, effective_from, created_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("building insert query: %w", err)
	}

	err = tx.QueryRow(ctx, insertQ, insertArgs...).Scan(&config.ID, &config.EffectiveFrom, &config.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting new config: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (r *calibrationRepository) GetAuditHistory(ctx context.Context, deviceID string, offset, limit uint64) ([]model.CalibrationAuditLog, error) {
	q, args, err := r.psql.
		Select("id", "device_id", "action", "old_values", "new_values", "performed_by", "performed_at", "reason").
		From("calibration_audit_logs").
		Where(sq.Eq{"device_id": deviceID}).
		OrderBy("performed_at DESC").
		Limit(limit).
		Offset(offset).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("building audit logs query: %w", err)
	}

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("querying audit logs: %w", err)
	}
	defer rows.Close()

	var logs []model.CalibrationAuditLog
	for rows.Next() {
		var log model.CalibrationAuditLog
		if err := rows.Scan(
			&log.ID, &log.DeviceID, &log.Action, &log.OldValues, &log.NewValues,
			&log.PerformedBy, &log.PerformedAt, &log.Reason,
		); err != nil {
			return nil, fmt.Errorf("scanning audit log row: %w", err)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return logs, nil
}
