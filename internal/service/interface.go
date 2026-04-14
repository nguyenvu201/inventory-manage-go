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
	GetAuditHistory(ctx context.Context, deviceID string, offset, limit uint64) ([]model.CalibrationAuditLog, error)
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
	Save(ctx context.Context, config *model.CalibrationConfig) error
	GetActive(ctx context.Context, deviceID string) (*model.CalibrationConfig, error)
	UpdateCalibrationTx(ctx context.Context, deviceID string, newConfig *model.CalibrationConfig) error
	GetAuditHistory(ctx context.Context, deviceID string, offset, limit uint64) ([]model.CalibrationAuditLog, error)
}

// ITelemetryRepository is the persistence contract for telemetry records.
type ITelemetryRepository interface {
	Save(ctx context.Context, t *model.RawTelemetry) error
	SaveBatch(ctx context.Context, records []*model.RawTelemetry) error
	FindByDeviceID(ctx context.Context, q model.TelemetryQuery) ([]*model.RawTelemetry, error)
	IsDuplicate(ctx context.Context, deviceID string, fCnt uint32) (bool, error)
}

// IInventoryRepository is the persistence contract for inventory snapshots and SKU configs.
type IInventoryRepository interface {
	UpsertSnapshot(ctx context.Context, snapshot *model.InventorySnapshot) error
	GetSnapshotBySKU(ctx context.Context, skuCode string) ([]*model.InventorySnapshot, error)
	GetCurrentSnapshots(ctx context.Context) ([]*model.InventorySnapshot, error)
	GetSKUConfig(ctx context.Context, skuCode string) (*model.SKUConfig, error)
}

// ── Threshold Rules ──────────────────────────────────────────────────────────

// IThresholdService defines the contract for threshold rule management (CRUD).
type IThresholdService interface {
	CreateRule(ctx context.Context, rule *model.ThresholdRule) error
	GetRules(ctx context.Context, query model.ThresholdRuleQuery) ([]*model.ThresholdRule, error)
	GetRuleByID(ctx context.Context, id string) (*model.ThresholdRule, error)
	UpdateRule(ctx context.Context, id string, rule *model.ThresholdRule) error
	DeleteRule(ctx context.Context, id string) error
}

// IThresholdEvaluator defines the contract for evaluating threshold rules based on inventory changes.
type IThresholdEvaluator interface {
	Evaluate(ctx context.Context, snapshot *model.InventorySnapshot) error
}

// IThresholdRepository is the persistence contract for threshold rules.
type IThresholdRepository interface {
	Save(ctx context.Context, rule *model.ThresholdRule) error
	FindAll(ctx context.Context, query model.ThresholdRuleQuery) ([]*model.ThresholdRule, error)
	FindByID(ctx context.Context, id string) (*model.ThresholdRule, error)
	FindBySKU(ctx context.Context, skuCode string) ([]*model.ThresholdRule, error)
	Update(ctx context.Context, rule *model.ThresholdRule) error
	Delete(ctx context.Context, id string) error
}
