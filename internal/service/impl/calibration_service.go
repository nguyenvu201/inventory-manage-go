package impl

import (
	"context"
	"fmt"

	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
)

// CalibrationServiceImpl is the concrete implementation of ICalibrationService.
type CalibrationServiceImpl struct {
	repo service.ICalibrationRepository
}

// NewCalibrationService creates a CalibrationServiceImpl.
func NewCalibrationService(repo service.ICalibrationRepository) service.ICalibrationService {
	return &CalibrationServiceImpl{repo: repo}
}

func (s *CalibrationServiceImpl) RegisterCalibration(ctx context.Context, cfg *model.CalibrationConfig) error {
	if cfg.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if cfg.ZeroValue < 0 {
		return fmt.Errorf("zero_value must be non-negative")
	}
	if cfg.SpanValue <= 0 {
		return fmt.Errorf("span_value must be positive")
	}
	if cfg.CapacityMax <= 0 {
		return fmt.Errorf("capacity_max must be positive")
	}

	return s.repo.Save(ctx, cfg)
}

func (s *CalibrationServiceImpl) GetActiveCalibration(ctx context.Context, deviceID string) (*model.CalibrationConfig, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device_id is required")
	}
	return s.repo.GetActive(ctx, deviceID)
}

func (s *CalibrationServiceImpl) UpdateCalibration(ctx context.Context, deviceID string, params *model.UpdateCalibrationParams) error {
	if deviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if params.ZeroValue < 0 {
		return fmt.Errorf("zero_value must be non-negative")
	}
	if params.SpanValue <= params.ZeroValue {
		return fmt.Errorf("span_value must be strictly greater than zero_value")
	}
	if params.CapacityMax <= 0 {
		return fmt.Errorf("capacity_max must be positive")
	}
	
	validUnits := map[string]bool{"kg": true, "g": true, "lb": true, "oz": true}
	if !validUnits[params.Unit] {
		return fmt.Errorf("invalid unit, must be one of: kg, g, lb, oz")
	}

	validTypes := map[string]bool{
		model.CalibrationTypeInitial:         true,
		model.CalibrationTypePeriodic:        true,
		model.CalibrationTypeDriftCorrection: true,
	}
	if !validTypes[params.CalibrationType] {
		return fmt.Errorf("invalid calibration_type")
	}

	cfg := &model.CalibrationConfig{
		DeviceID:       deviceID,
		ZeroValue:      params.ZeroValue,
		SpanValue:      params.SpanValue,
		Unit:           params.Unit,
		CapacityMax:    params.CapacityMax,
		HardwareConfig: params.HardwareConfig,
		CalibrationType: params.CalibrationType,
		CreatedBy:      params.CreatedBy,
	}

	return s.repo.UpdateCalibrationTx(ctx, deviceID, cfg)
}
