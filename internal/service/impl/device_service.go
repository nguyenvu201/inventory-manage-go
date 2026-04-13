package impl

import (
	"context"
	"fmt"
	"time"

	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
)

// DeviceServiceImpl is the concrete implementation of IDeviceService.
// It references the repository by its interface, not the concrete struct.
type DeviceServiceImpl struct {
	repo service.IDeviceRepository
}

// NewDeviceService creates a DeviceServiceImpl.
// Wire will inject the IDeviceRepository implementation.
func NewDeviceService(repo service.IDeviceRepository) service.IDeviceService {
	return &DeviceServiceImpl{repo: repo}
}

func (s *DeviceServiceImpl) RegisterDevice(ctx context.Context, d *model.Device) error {
	if d.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if d.SKUCode == "" {
		return fmt.Errorf("sku_code is required")
	}
	if !d.IsValidStatus() {
		return model.ErrInvalidStatus
	}

	now := time.Now()
	d.CreatedAt = now
	d.UpdatedAt = now

	return s.repo.Save(ctx, d)
}

func (s *DeviceServiceImpl) GetDevice(ctx context.Context, id string) (*model.Device, error) {
	if id == "" {
		return nil, fmt.Errorf("device_id is required")
	}
	return s.repo.FindByID(ctx, id)
}

func (s *DeviceServiceImpl) ListDevices(ctx context.Context, q model.DeviceQuery) ([]*model.Device, error) {
	return s.repo.FindAll(ctx, q)
}

func (s *DeviceServiceImpl) UpdateDevice(ctx context.Context, d *model.Device) error {
	if d.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if !d.IsValidStatus() {
		return model.ErrInvalidStatus
	}

	existing, err := s.repo.FindByID(ctx, d.DeviceID)
	if err != nil {
		return fmt.Errorf("DeviceService.UpdateDevice: %w", err)
	}

	existing.Name = d.Name
	if d.SKUCode != "" {
		existing.SKUCode = d.SKUCode
	}
	existing.Location = d.Location
	existing.Status = d.Status
	existing.UpdatedAt = time.Now()

	return s.repo.Update(ctx, existing)
}

func (s *DeviceServiceImpl) RemoveDevice(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("device_id is required")
	}
	return s.repo.Delete(ctx, id)
}
