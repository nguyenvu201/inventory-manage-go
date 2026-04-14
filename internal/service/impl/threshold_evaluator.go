package impl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
	"inventory-manage/pkg/logger"

	"go.uber.org/zap"
)

type thresholdEvaluator struct {
	repo         service.IThresholdRepository
	eventBus     model.IEventBus
	logger       *logger.LoggerZap
	cooldowns    sync.Map // key: skuCode_ruleType, value: time.Time
}

func NewThresholdEvaluator(repo service.IThresholdRepository, eventBus model.IEventBus, logger *logger.LoggerZap) service.IThresholdEvaluator {
	return &thresholdEvaluator{
		repo:      repo,
		eventBus:  eventBus,
		logger:    logger,
	}
}

func (e *thresholdEvaluator) Evaluate(ctx context.Context, snapshot *model.InventorySnapshot) error {
	rules, err := e.repo.FindBySKU(ctx, snapshot.SKUCode)
	if err != nil {
		return fmt.Errorf("ThresholdEvaluator.Evaluate: failed to find rules for sku %s: %w", snapshot.SKUCode, err)
	}

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		breached := false

		// Check trigger conditions
		if rule.TriggerPercentage != nil {
			switch rule.RuleType {
			case model.RuleTypeLowStock, model.RuleTypeCritical:
				if snapshot.Percentage <= *rule.TriggerPercentage {
					breached = true
				}
			case model.RuleTypeOverstock:
				if snapshot.Percentage >= *rule.TriggerPercentage {
					breached = true
				}
			}
		}

		if rule.TriggerQty != nil && !breached {
			switch rule.RuleType {
			case model.RuleTypeLowStock, model.RuleTypeCritical:
				if snapshot.Qty <= *rule.TriggerQty {
					breached = true
				}
			case model.RuleTypeOverstock:
				if snapshot.Qty >= *rule.TriggerQty {
					breached = true
				}
			}
		}

		if breached {
			if e.isUnderCooldown(rule) {
				continue
			}

			event := model.ThresholdBreachedEvent{
				EventID:           fmt.Sprintf("%s-%d", rule.SKUCode, time.Now().UnixNano()),
				DeviceID:          snapshot.DeviceID,
				SKUCode:           snapshot.SKUCode,
				RuleType:          rule.RuleType,
				CurrentPercentage: snapshot.Percentage,
				CurrentQty:        snapshot.Qty,
				Timestamp:         time.Now(),
			}

			if err := e.eventBus.Publish("threshold.breached", event); err != nil {
				if e.logger != nil {
					e.logger.Error("failed to publish ThresholdBreachedEvent", zap.Error(err), zap.String("sku_code", snapshot.SKUCode))
				}
			}

			e.setCooldown(rule)
		}
	}

	return nil
}

func (e *thresholdEvaluator) isUnderCooldown(rule *model.ThresholdRule) bool {
	key := fmt.Sprintf("%s_%s", rule.SKUCode, rule.RuleType)
	val, ok := e.cooldowns.Load(key)
	if !ok {
		return false
	}
	lastFired := val.(time.Time)
	expiration := lastFired.Add(time.Duration(rule.CooldownMinutes) * time.Minute)
	return time.Now().Before(expiration)
}

func (e *thresholdEvaluator) setCooldown(rule *model.ThresholdRule) {
	key := fmt.Sprintf("%s_%s", rule.SKUCode, rule.RuleType)
	e.cooldowns.Store(key, time.Now())
}
