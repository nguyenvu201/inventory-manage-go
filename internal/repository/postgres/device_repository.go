package postgres

import (
	"context"
	"errors"

	"inventory-manage/internal/domain/device"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type DeviceRepository struct {
	pool *pgxpool.Pool
}

func NewDeviceRepository(pool *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{pool: pool}
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return device.ErrDeviceNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 23505 is unique_violation
		if pgErr.Code == "23505" {
			return device.ErrDuplicateDevice
		}
	}
	return err
}

func (r *DeviceRepository) Save(ctx context.Context, d *device.Device) error {
	query, args, err := psql.Insert("devices").
		Columns("device_id", "name", "sku_code", "location", "status", "created_at", "updated_at").
		Values(d.DeviceID, d.Name, d.SKUCode, d.Location, d.Status, d.CreatedAt, d.UpdatedAt).
		ToSql()

	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, query, args...)
	return mapDBError(err)
}

func (r *DeviceRepository) FindByID(ctx context.Context, id string) (*device.Device, error) {
	query, args, err := psql.Select("device_id", "name", "sku_code", "location", "status", "created_at", "updated_at").
		From("devices").
		Where(squirrel.Eq{"device_id": id}).
		ToSql()

	if err != nil {
		return nil, err
	}

	var d device.Device
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&d.DeviceID, &d.Name, &d.SKUCode, &d.Location, &d.Status, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, mapDBError(err)
	}

	return &d, nil
}

func (r *DeviceRepository) FindAll(ctx context.Context, q device.DeviceQuery) ([]*device.Device, error) {
	builder := psql.Select("device_id", "name", "sku_code", "location", "status", "created_at", "updated_at").
		From("devices").
		OrderBy("created_at DESC")

	if q.Status != nil {
		builder = builder.Where(squirrel.Eq{"status": *q.Status})
	}
	if q.SKUCode != nil {
		builder = builder.Where(squirrel.Eq{"sku_code": *q.SKUCode})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*device.Device
	for rows.Next() {
		var d device.Device
		if err := rows.Scan(&d.DeviceID, &d.Name, &d.SKUCode, &d.Location, &d.Status, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		devices = append(devices, &d)
	}

	return devices, rows.Err()
}

func (r *DeviceRepository) Update(ctx context.Context, d *device.Device) error {
	query, args, err := psql.Update("devices").
		Set("name", d.Name).
		Set("sku_code", d.SKUCode).
		Set("location", d.Location).
		Set("status", d.Status).
		Where(squirrel.Eq{"device_id": d.DeviceID}).
		ToSql()

	if err != nil {
		return err
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return mapDBError(err)
	}

	if tag.RowsAffected() == 0 {
		return device.ErrDeviceNotFound
	}
	return nil
}

func (r *DeviceRepository) Delete(ctx context.Context, id string) error {
	query, args, err := psql.Delete("devices").
		Where(squirrel.Eq{"device_id": id}).
		ToSql()

	if err != nil {
		return err
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return device.ErrDeviceNotFound
	}
	return nil
}
