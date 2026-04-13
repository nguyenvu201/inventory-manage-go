package model

import "errors"

// WeightResult represents the result of converting raw ADC readings to physical weights
type WeightResult struct {
	GrossWeightKg float64 `json:"gross_weight_kg"`
	NetWeightKg   float64 `json:"net_weight_kg"`
}

// Domain errors
var (
	ErrNoActiveCalibration = errors.New("no active calibration found for device")
	ErrInvalidSpanValue    = errors.New("invalid calibration span value (cannot be zero)")
	ErrSKUNotFound         = errors.New("sku config not found")
)

// SKUConfig represents the configuration for a given SKU (Stock Keeping Unit).
type SKUConfig struct {
	SKUCode         string  `json:"sku_code" db:"sku_code"`
	UnitWeightKg    float64 `json:"unit_weight_kg" db:"unit_weight_kg"`
	FullCapacityKg  float64 `json:"full_capacity_kg" db:"full_capacity_kg"`
	TareWeightKg    float64 `json:"tare_weight_kg" db:"tare_weight_kg"`
	ResolutionKg    float64 `json:"resolution_kg" db:"resolution_kg"`
	ReorderPointQty int     `json:"reorder_point_qty" db:"reorder_point_qty"`
	UnitLabel       string  `json:"unit_label" db:"unit_label"`
}

// InventorySnapshot represents the current inventory level of a device holding a specific SKU.
type InventorySnapshot struct {
	DeviceID    string  `json:"device_id" db:"device_id"`
	SKUCode     string  `json:"sku_code" db:"sku_code"`
	NetWeightKg float64 `json:"net_weight_kg" db:"net_weight_kg"`
	Qty         int     `json:"qty" db:"qty"`
	Percentage  float64 `json:"percentage" db:"percentage"`
}
