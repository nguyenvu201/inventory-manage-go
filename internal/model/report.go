package model

import "time"

// ConsumptionQuery represents the filters for retrieving historical consumption reports
type ConsumptionQuery struct {
	SKUCode  string    `json:"sku_code" form:"sku_code" binding:"required"`
	From     time.Time `json:"from" form:"from" time_format:"2006-01-02T15:04:05Z07:00" binding:"required"`
	To       time.Time `json:"to" form:"to" time_format:"2006-01-02T15:04:05Z07:00" binding:"required"`
	Interval string    `json:"interval" form:"interval" binding:"required,oneof=1h 1d 1w"`
	Limit    int       `json:"limit" form:"limit" binding:"omitempty,min=1,max=1000"`
	Cursor   string    `json:"cursor,omitempty" form:"cursor"`
}

// ConsumptionDataPoint represents a single aggregated data point over an interval
type ConsumptionDataPoint struct {
	Timestamp   time.Time `json:"timestamp" db:"bucket"`
	NetWeightKg float64   `json:"net_weight_kg" db:"avg_net_weight_kg"`
	Qty         float64   `json:"qty" db:"avg_qty"`
	Percentage  float64   `json:"percentage" db:"avg_percentage"`
}

// ConsumptionSummary represents a high-level summary of consumption over a period
type ConsumptionSummary struct {
	SKUCode            string  `json:"sku_code"`
	TotalConsumptionKg float64 `json:"total_consumption_kg"`
	OpeningQty         int     `json:"opening_qty"`
	ClosingQty         int     `json:"closing_qty"`
	OpeningWeightKg    float64 `json:"opening_weight_kg"`
	ClosingWeightKg    float64 `json:"closing_weight_kg"`
}
