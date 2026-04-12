package postgres

import (
	"context"
	"errors"
	"fmt"

	"inventory-manage/internal/domain/device"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type calibrationRepository struct {
	db *pgxpool.Pool
	psql sq.StatementBuilderType
}

func NewCalibrationRepository(db *pgxpool.Pool) device.CalibrationRepository {
	return &calibrationRepository{
		db:   db,
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

// Save inserts a new configuration and implicitly deactivates the previous active one.
func (r *calibrationRepository) Save(ctx context.Context, config *device.CalibrationConfig) error {
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

func (r *calibrationRepository) GetActive(ctx context.Context, deviceID string) (*device.CalibrationConfig, error) {
	q, args, err := r.psql.
		Select("id", "device_id", "zero_value", "span_value", "unit", "capacity_max", "hardware_config", "effective_from", "created_by", "created_at").
		From("calibration_configs").
		Where(sq.Eq{"device_id": deviceID}).
		Where(sq.Expr("deactivated_at IS NULL")).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("building get_active query: %w", err)
	}

	var c device.CalibrationConfig
	err = r.db.QueryRow(ctx, q, args...).Scan(
		&c.ID, &c.DeviceID, &c.ZeroValue, &c.SpanValue, &c.Unit, &c.CapacityMax, &c.HardwareConfig, &c.EffectiveFrom, &c.CreatedBy, &c.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no active config: %w", device.ErrDeviceNotFound)
		}
		return nil, fmt.Errorf("querying active config: %w", err)
	}

	return &c, nil
}
