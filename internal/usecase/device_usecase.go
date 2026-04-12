package usecase

import (
	"context"
	"fmt"
	"time"

	"inventory-manage/internal/domain/device"
)

type DeviceUseCaseImpl struct {
	repo device.Repository
}

func NewDeviceUseCase(repo device.Repository) device.UseCase {
	return &DeviceUseCaseImpl{repo: repo}
}

func (uc *DeviceUseCaseImpl) RegisterDevice(ctx context.Context, d *device.Device) error {
	if d.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if d.SKUCode == "" {
		return fmt.Errorf("sku_code is required")
	}
	if !d.IsValidStatus() {
		return device.ErrInvalidStatus
	}

	d.CreatedAt = time.Now()
	d.UpdatedAt = time.Now()

	return uc.repo.Save(ctx, d)
}

func (uc *DeviceUseCaseImpl) GetDevice(ctx context.Context, id string) (*device.Device, error) {
	if id == "" {
		return nil, fmt.Errorf("device_id is required")
	}
	return uc.repo.FindByID(ctx, id)
}

func (uc *DeviceUseCaseImpl) ListDevices(ctx context.Context, q device.DeviceQuery) ([]*device.Device, error) {
	return uc.repo.FindAll(ctx, q)
}

func (uc *DeviceUseCaseImpl) UpdateDevice(ctx context.Context, d *device.Device) error {
	if d.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if !d.IsValidStatus() {
		return device.ErrInvalidStatus
	}

	// We only overwrite the modifiable fields
	existing, err := uc.repo.FindByID(ctx, d.DeviceID)
	if err != nil {
		return err
	}

	existing.Name = d.Name
	if d.SKUCode != "" {
		existing.SKUCode = d.SKUCode
	}
	existing.Location = d.Location
	existing.Status = d.Status
	// updated_at is handled by DB trigger, but we update struct here anyway
	existing.UpdatedAt = time.Now()

	return uc.repo.Update(ctx, existing)
}

func (uc *DeviceUseCaseImpl) RemoveDevice(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("device_id is required")
	}
	return uc.repo.Delete(ctx, id)
}
