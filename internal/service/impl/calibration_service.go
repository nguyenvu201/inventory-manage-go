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
