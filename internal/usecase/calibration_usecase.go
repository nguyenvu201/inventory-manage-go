package usecase

import (
	"context"
	"errors"

	"inventory-manage/internal/domain/device"
)

type calibrationUseCase struct {
	repo device.CalibrationRepository
}

func NewCalibrationUseCase(repo device.CalibrationRepository) device.CalibrationUseCase {
	return &calibrationUseCase{repo: repo}
}

func (uc *calibrationUseCase) RegisterCalibration(ctx context.Context, config *device.CalibrationConfig) error {
	if config.DeviceID == "" {
		return errors.New("device_id is required")
	}
	if config.CapacityMax <= 0 {
		return errors.New("capacity_max must be greater than zero")
	}
	if config.Unit == "" {
		return errors.New("unit is required")
	}
	if config.HardwareConfig == nil {
		config.HardwareConfig = make(map[string]interface{})
	}

	return uc.repo.Save(ctx, config)
}

func (uc *calibrationUseCase) GetActiveCalibration(ctx context.Context, deviceID string) (*device.CalibrationConfig, error) {
	if deviceID == "" {
		return nil, errors.New("device_id is required")
	}
	return uc.repo.GetActive(ctx, deviceID)
}
