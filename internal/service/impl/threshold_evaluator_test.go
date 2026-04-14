package impl

import (
	"context"
	"errors"
	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockThresholdRepo
type mockThresholdRepo struct {
	mock.Mock
}

func (m *mockThresholdRepo) Save(ctx context.Context, rule *model.ThresholdRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}
func (m *mockThresholdRepo) FindAll(ctx context.Context, query model.ThresholdRuleQuery) ([]*model.ThresholdRule, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]*model.ThresholdRule), args.Error(1)
}
func (m *mockThresholdRepo) FindByID(ctx context.Context, id string) (*model.ThresholdRule, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.ThresholdRule), args.Error(1)
}
func (m *mockThresholdRepo) FindBySKU(ctx context.Context, skuCode string) ([]*model.ThresholdRule, error) {
	args := m.Called(ctx, skuCode)
	return args.Get(0).([]*model.ThresholdRule), args.Error(1)
}
func (m *mockThresholdRepo) Update(ctx context.Context, rule *model.ThresholdRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}
func (m *mockThresholdRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// mockEventBus
type mockEventBus struct {
	mock.Mock
}

func (m *mockEventBus) Publish(topic string, event interface{}) error {
	args := m.Called(topic, event)
	return args.Error(0)
}

func (m *mockEventBus) Subscribe(ctx context.Context, topic string) (<-chan interface{}, error) {
	args := m.Called(ctx, topic)
	return args.Get(0).(<-chan interface{}), args.Error(1)
}

// Ensure interface
var _ service.IThresholdRepository = (*mockThresholdRepo)(nil)
var _ model.IEventBus = (*mockEventBus)(nil)

// Package impl implements tests for INV-SPR03-TASK-003
// AC Coverage:
//   AC-06: TestThresholdEvaluator_Evaluate
// IEC 62304 Classification: Software Safety Class B

func TestThresholdEvaluator_Evaluate(t *testing.T) {
	ctx := context.Background()

	perc30 := 30.0
	tests := []struct {
		name          string
		snapshot      *model.InventorySnapshot
		rules         []*model.ThresholdRule
		prepareMock   func(repo *mockThresholdRepo, bus *mockEventBus)
		expectPublish bool
		secondEval    bool
		wantErr       bool
	}{
		{
			name: "AC-06: threshold breach triggers event",
			snapshot: &model.InventorySnapshot{
				DeviceID:   "DEV-1",
				SKUCode:    "SKU-A",
				Percentage: 20.0,
				Qty:        10,
			},
			rules: []*model.ThresholdRule{
				{
					SKUCode:           "SKU-A",
					RuleType:          model.RuleTypeLowStock,
					TriggerPercentage: &perc30,
					IsActive:          true,
					CooldownMinutes:   60,
				},
			},
			prepareMock: func(repo *mockThresholdRepo, bus *mockEventBus) {
				repo.On("FindBySKU", ctx, "SKU-A").Return([]*model.ThresholdRule{
					{
						SKUCode:           "SKU-A",
						RuleType:          model.RuleTypeLowStock,
						TriggerPercentage: &perc30,
						IsActive:          true,
						CooldownMinutes:   60,
					},
				}, nil)
				bus.On("Publish", "threshold.breached", mock.Anything).Return(nil).Once()
			},
			expectPublish: true,
		},
		{
			name: "AC-06: rule is disabled, no event",
			snapshot: &model.InventorySnapshot{
				DeviceID:   "DEV-1",
				SKUCode:    "SKU-B",
				Percentage: 20.0,
			},
			rules: []*model.ThresholdRule{
				{
					SKUCode:           "SKU-B",
					RuleType:          model.RuleTypeLowStock,
					TriggerPercentage: &perc30,
					IsActive:          false, // disabled
					CooldownMinutes:   60,
				},
			},
			prepareMock: func(repo *mockThresholdRepo, bus *mockEventBus) {
				repo.On("FindBySKU", ctx, "SKU-B").Return([]*model.ThresholdRule{
					{
						SKUCode:           "SKU-B",
						RuleType:          model.RuleTypeLowStock,
						TriggerPercentage: &perc30,
						IsActive:          false,
						CooldownMinutes:   60,
					},
				}, nil)
				// Publish should not be called
			},
			expectPublish: false,
		},
		{
			name: "AC-06: within cooldown window",
			snapshot: &model.InventorySnapshot{
				DeviceID:   "DEV-1",
				SKUCode:    "SKU-C",
				Percentage: 20.0,
			},
			rules: []*model.ThresholdRule{
				{
					SKUCode:           "SKU-C",
					RuleType:          model.RuleTypeLowStock,
					TriggerPercentage: &perc30,
					IsActive:          true,
					CooldownMinutes:   60,
				},
			},
			prepareMock: func(repo *mockThresholdRepo, bus *mockEventBus) {
				repo.On("FindBySKU", ctx, "SKU-C").Return([]*model.ThresholdRule{
					{
						SKUCode:           "SKU-C",
						RuleType:          model.RuleTypeLowStock,
						TriggerPercentage: &perc30,
						IsActive:          true,
						CooldownMinutes:   60,
					},
				}, nil).Twice() // Because we'll evaluate twice
				bus.On("Publish", "threshold.breached", mock.Anything).Return(nil).Once() // Only once despite 2 evals
			},
			expectPublish: true,
			secondEval:    true,
		},
		{
			name: "Repo error",
			snapshot: &model.InventorySnapshot{
				DeviceID: "DEV-1", SKUCode: "SKU-ERR",
			},
			prepareMock: func(repo *mockThresholdRepo, bus *mockEventBus) {
				repo.On("FindBySKU", ctx, "SKU-ERR").Return(([]*model.ThresholdRule)(nil), errors.New("db error"))
			},
			expectPublish: false,
			wantErr:       true,
		},
		{
			name: "Trigger Qty rule (Overstock)",
			snapshot: &model.InventorySnapshot{
				DeviceID: "DEV-1", SKUCode: "SKU-QTY",
				Qty: 100, // overstock trigger is 90
			},
			prepareMock: func(repo *mockThresholdRepo, bus *mockEventBus) {
				qty := 90
				repo.On("FindBySKU", ctx, "SKU-QTY").Return([]*model.ThresholdRule{
					{
						SKUCode: "SKU-QTY", RuleType: model.RuleTypeOverstock,
						TriggerQty: &qty, IsActive: true, CooldownMinutes: 10,
					},
				}, nil)
				bus.On("Publish", "threshold.breached", mock.Anything).Return(errors.New("publish error")).Once()
			},
			expectPublish: true, // expect it to try publishing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockThresholdRepo)
			bus := new(mockEventBus)

			if tt.prepareMock != nil {
				tt.prepareMock(repo, bus)
			}

			eval := NewThresholdEvaluator(repo, bus, nil)

			err := eval.Evaluate(ctx, tt.snapshot)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.secondEval {
				err = eval.Evaluate(ctx, tt.snapshot)
				assert.NoError(t, err)
			}

			repo.AssertExpectations(t)
			bus.AssertExpectations(t)
		})
	}
}
