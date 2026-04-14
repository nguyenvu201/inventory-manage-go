package impl

import (
	"context"
	"fmt"
	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
)

type thresholdService struct {
	repo service.IThresholdRepository
}

func NewThresholdService(repo service.IThresholdRepository) service.IThresholdService {
	return &thresholdService{repo: repo}
}

func (s *thresholdService) CreateRule(ctx context.Context, rule *model.ThresholdRule) error {
	if rule.SKUCode == "" {
		return fmt.Errorf("ThresholdService.CreateRule: sku_code is required")
	}
	if rule.RuleType == "" {
		return fmt.Errorf("ThresholdService.CreateRule: rule_type is required")
	}
	if rule.TriggerPercentage == nil && rule.TriggerQty == nil {
		return fmt.Errorf("ThresholdService.CreateRule: either trigger_percentage or trigger_qty must be specified")
	}
	
	err := s.repo.Save(ctx, rule)
	if err != nil {
		return fmt.Errorf("ThresholdService.CreateRule: %w", err)
	}
	return nil
}

func (s *thresholdService) GetRules(ctx context.Context, query model.ThresholdRuleQuery) ([]*model.ThresholdRule, error) {
	rules, err := s.repo.FindAll(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ThresholdService.GetRules: %w", err)
	}
	return rules, nil
}

func (s *thresholdService) GetRuleByID(ctx context.Context, id string) (*model.ThresholdRule, error) {
	if id == "" {
		return nil, fmt.Errorf("ThresholdService.GetRuleByID: id is required")
	}
	rule, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ThresholdService.GetRuleByID: %w", err)
	}
	return rule, nil
}

func (s *thresholdService) UpdateRule(ctx context.Context, id string, rule *model.ThresholdRule) error {
	if id == "" {
		return fmt.Errorf("ThresholdService.UpdateRule: id is required")
	}
	rule.ID = id
	err := s.repo.Update(ctx, rule)
	if err != nil {
		return fmt.Errorf("ThresholdService.UpdateRule: %w", err)
	}
	return nil
}

func (s *thresholdService) DeleteRule(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("ThresholdService.DeleteRule: id is required")
	}
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("ThresholdService.DeleteRule: %w", err)
	}
	return nil
}
