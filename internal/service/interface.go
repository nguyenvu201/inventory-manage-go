package service

import (
	"context"

	"inventory-manage/internal/model"
)

// ── Device ───────────────────────────────────────────────────────────────────

// IDeviceService defines the contract for device management business logic.
// The interface is defined at the consumer side (service package) following
// Go conventions for dependency inversion.
type IDeviceService interface {
	RegisterDevice(ctx context.Context, d *model.Device) error
	GetDevice(ctx context.Context, id string) (*model.Device, error)
	ListDevices(ctx context.Context, q model.DeviceQuery) ([]*model.Device, error)
	UpdateDevice(ctx context.Context, d *model.Device) error
	RemoveDevice(ctx context.Context, id string) error
}

// ── Calibration ──────────────────────────────────────────────────────────────

// ICalibrationService defines the contract for calibration management.
type ICalibrationService interface {
	RegisterCalibration(ctx context.Context, cfg *model.CalibrationConfig) error
	GetActiveCalibration(ctx context.Context, deviceID string) (*model.CalibrationConfig, error)
	UpdateCalibration(ctx context.Context, deviceID string, params *model.UpdateCalibrationParams) error
}

// ── Repository interfaces (defined here, implemented in repository/) ─────────

// IDeviceRepository is the persistence contract for devices.
type IDeviceRepository interface {
	Save(ctx context.Context, d *model.Device) error
	FindByID(ctx context.Context, id string) (*model.Device, error)
	FindAll(ctx context.Context, q model.DeviceQuery) ([]*model.Device, error)
	Update(ctx context.Context, d *model.Device) error
	Delete(ctx context.Context, id string) error
}

// ICalibrationRepository is the persistence contract for calibration configs.
type ICalibrationRepository interface {
	Save(ctx context.Context, cfg *model.CalibrationConfig) error
	GetActive(ctx context.Context, deviceID string) (*model.CalibrationConfig, error)
	UpdateCalibrationTx(ctx context.Context, deviceID string, config *model.CalibrationConfig) error
}

// ITelemetryRepository is the persistence contract for telemetry records.
type ITelemetryRepository interface {
	Save(ctx context.Context, t *model.RawTelemetry) error
	SaveBatch(ctx context.Context, records []*model.RawTelemetry) error
	FindByDeviceID(ctx context.Context, q model.TelemetryQuery) ([]*model.RawTelemetry, error)
	IsDuplicate(ctx context.Context, deviceID string, fCnt uint32) (bool, error)
}
