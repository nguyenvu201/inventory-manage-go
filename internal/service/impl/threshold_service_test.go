// Package impl implements tests for INV-SPR03-TASK-003
// AC Coverage:
//   AC-05: TestThresholdService_CreateRule, TestThresholdService_GetRules, TestThresholdService_UpdateRule, TestThresholdService_DeleteRule
// IEC 62304 Classification: Software Safety Class B
package impl

import (
	"context"
	"errors"
	"inventory-manage/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestThresholdService_CreateRule(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   *model.ThresholdRule
		repoErr error
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid low stock rule",
			input: &model.ThresholdRule{
				SKUCode:           "SKU-A",
				RuleType:          model.RuleTypeLowStock,
				TriggerPercentage: newFloat(20),
			},
			repoErr: nil,
			wantErr: false,
		},
		{
			name: "Missing SKUCode",
			input: &model.ThresholdRule{
				RuleType:          model.RuleTypeLowStock,
				TriggerPercentage: newFloat(20),
			},
			wantErr: true,
			errMsg:  "sku_code is required",
		},
		{
			name: "Missing RuleType",
			input: &model.ThresholdRule{
				SKUCode:           "SKU-A",
				TriggerPercentage: newFloat(20),
			},
			wantErr: true,
			errMsg:  "rule_type is required",
		},
		{
			name: "Missing triggers",
			input: &model.ThresholdRule{
				SKUCode:  "SKU-A",
				RuleType: model.RuleTypeLowStock,
			},
			wantErr: true,
			errMsg:  "either trigger_percentage or trigger_qty must be specified",
		},
		{
			name: "Repo error",
			input: &model.ThresholdRule{
				SKUCode:           "SKU-A",
				RuleType:          model.RuleTypeLowStock,
				TriggerPercentage: newFloat(20),
			},
			repoErr: errors.New("db error"),
			wantErr: true,
			errMsg:  "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockThresholdRepo)

			if !tt.wantErr || tt.repoErr != nil {
				repo.On("Save", ctx, tt.input).Return(tt.repoErr)
			}

			svc := NewThresholdService(repo)
			err := svc.CreateRule(ctx, tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestThresholdService_GetRules(t *testing.T) {
	ctx := context.Background()
	repo := new(mockThresholdRepo)

	query := model.ThresholdRuleQuery{}
	rules := []*model.ThresholdRule{{ID: "1"}}

	repo.On("FindAll", ctx, query).Return(rules, nil)

	svc := NewThresholdService(repo)
	res, err := svc.GetRules(ctx, query)

	assert.NoError(t, err)
	assert.Equal(t, rules, res)
}

func TestThresholdService_GetRuleByID(t *testing.T) {
	ctx := context.Background()

	t.Run("Valid", func(t *testing.T) {
		repo := new(mockThresholdRepo)
		rule := &model.ThresholdRule{ID: "1"}
		repo.On("FindByID", ctx, "1").Return(rule, nil)

		svc := NewThresholdService(repo)
		res, err := svc.GetRuleByID(ctx, "1")

		assert.NoError(t, err)
		assert.Equal(t, rule, res)
	})

	t.Run("Missing ID", func(t *testing.T) {
		repo := new(mockThresholdRepo)
		svc := NewThresholdService(repo)
		_, err := svc.GetRuleByID(ctx, "")
		assert.Error(t, err)
	})
}

func TestThresholdService_UpdateRule(t *testing.T) {
	ctx := context.Background()

	t.Run("Valid", func(t *testing.T) {
		repo := new(mockThresholdRepo)
		rule := &model.ThresholdRule{TriggerPercentage: newFloat(10)}

		repo.On("Update", ctx, mock.MatchedBy(func(r *model.ThresholdRule) bool {
			return r.ID == "1"
		})).Return(nil)

		svc := NewThresholdService(repo)
		err := svc.UpdateRule(ctx, "1", rule)
		assert.NoError(t, err)
	})

	t.Run("Missing ID", func(t *testing.T) {
		repo := new(mockThresholdRepo)
		svc := NewThresholdService(repo)
		err := svc.UpdateRule(ctx, "", &model.ThresholdRule{})
		assert.Error(t, err)
	})
}

func TestThresholdService_DeleteRule(t *testing.T) {
	ctx := context.Background()

	t.Run("Valid", func(t *testing.T) {
		repo := new(mockThresholdRepo)
		repo.On("Delete", ctx, "1").Return(nil)

		svc := NewThresholdService(repo)
		err := svc.DeleteRule(ctx, "1")
		assert.NoError(t, err)
	})

	t.Run("Missing ID", func(t *testing.T) {
		repo := new(mockThresholdRepo)
		svc := NewThresholdService(repo)
		err := svc.DeleteRule(ctx, "")
		assert.Error(t, err)
	})
}

func newFloat(f float64) *float64 { return &f }
