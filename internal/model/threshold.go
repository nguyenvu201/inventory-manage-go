package model

import "time"

// RuleType constants
const (
	RuleTypeLowStock  = "low_stock"
	RuleTypeCritical  = "critical"
	RuleTypeOverstock = "overstock"
)

// ThresholdRule represents a configured rule for alerting on inventory levels.
type ThresholdRule struct {
	ID                string    `json:"id" db:"id"`
	SKUCode           string    `json:"sku_code" db:"sku_code"`
	RuleType          string    `json:"rule_type" db:"rule_type"`
	TriggerPercentage *float64  `json:"trigger_percentage" db:"trigger_percentage"`
	TriggerQty        *int      `json:"trigger_qty" db:"trigger_qty"`
	CooldownMinutes   int       `json:"cooldown_minutes" db:"cooldown_minutes"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// ThresholdBreachedEvent is emitted when a rule is triggered.
type ThresholdBreachedEvent struct {
	EventID           string    `json:"event_id"`
	DeviceID          string    `json:"device_id"`
	SKUCode           string    `json:"sku_code"`
	RuleType          string    `json:"rule_type"`
	CurrentPercentage float64   `json:"current_percentage"`
	CurrentQty        int       `json:"current_qty"`
	Timestamp         time.Time `json:"timestamp"`
}

// ThresholdRuleQuery defines the filter params for CRUD API.
type ThresholdRuleQuery struct {
	SKUCode *string `form:"sku_code"`
	Limit   uint64  `form:"limit"`
	Offset  uint64  `form:"offset"`
}
