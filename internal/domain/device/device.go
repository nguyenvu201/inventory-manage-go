package device

import (
	"context"
	"errors"
	"time"
)

var (
	ErrDeviceNotFound  = errors.New("device not found")
	ErrDuplicateDevice = errors.New("device already exists")
	ErrInvalidStatus   = errors.New("invalid device status")
)

type DeviceStatus string

const (
	StatusActive      DeviceStatus = "active"
	StatusInactive    DeviceStatus = "inactive"
	StatusMaintenance DeviceStatus = "maintenance"
)

type Device struct {
	DeviceID  string       `json:"device_id"`
	Name      string       `json:"name"`
	SKUCode   string       `json:"sku_code"`
	Location  string       `json:"location"`
	Status    DeviceStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func (d *Device) IsValidStatus() bool {
	switch d.Status {
	case StatusActive, StatusInactive, StatusMaintenance:
		return true
	default:
		return false
	}
}

type DeviceQuery struct {
	Status  *DeviceStatus
	SKUCode *string
}

type Repository interface {
	Save(ctx context.Context, d *Device) error
	FindByID(ctx context.Context, id string) (*Device, error)
	FindAll(ctx context.Context, q DeviceQuery) ([]*Device, error)
	Update(ctx context.Context, d *Device) error
	Delete(ctx context.Context, id string) error
}

type UseCase interface {
	RegisterDevice(ctx context.Context, d *Device) error
	GetDevice(ctx context.Context, id string) (*Device, error)
	ListDevices(ctx context.Context, q DeviceQuery) ([]*Device, error)
	UpdateDevice(ctx context.Context, d *Device) error
	RemoveDevice(ctx context.Context, id string) error
}
