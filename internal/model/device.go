package model

import (
	"errors"
	"time"
)

// ── Sentinel errors ──────────────────────────────────────────────────────────

var (
	ErrDeviceNotFound  = errors.New("device not found")
	ErrDuplicateDevice = errors.New("device already exists")
	ErrInvalidStatus   = errors.New("invalid device status")
)

// ── Types ────────────────────────────────────────────────────────────────────

type DeviceStatus string

const (
	StatusActive      DeviceStatus = "active"
	StatusInactive    DeviceStatus = "inactive"
	StatusMaintenance DeviceStatus = "maintenance"
)

// Device represents a registered IoT scale in the inventory system.
type Device struct {
	DeviceID  string       `json:"device_id"`
	Name      string       `json:"name"`
	SKUCode   string       `json:"sku_code"`
	Location  string       `json:"location"`
	Status    DeviceStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// IsValidStatus returns true if the device status is one of the allowed values.
func (d *Device) IsValidStatus() bool {
	switch d.Status {
	case StatusActive, StatusInactive, StatusMaintenance:
		return true
	default:
		return false
	}
}

// DeviceQuery holds optional filters for listing devices.
type DeviceQuery struct {
	Status  *DeviceStatus
	SKUCode *string
}
