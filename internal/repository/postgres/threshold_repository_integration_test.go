//go:build integration

package postgres_test

import (
	"context"
	"inventory-manage/internal/model"
	"inventory-manage/internal/repository/postgres"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThresholdRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	pool, _ := setupTestDB(t) 
	repo := postgres.NewThresholdRepository(pool)

	pool.Exec(ctx, "TRUNCATE threshold_rules CASCADE")

	var savedRuleID string

	t.Run("Save new rule", func(t *testing.T) {
		pct := 20.0
		rule := &model.ThresholdRule{
			ID:                "RULE-001",
			SKUCode:           "SKU-A",
			RuleType:          model.RuleTypeLowStock,
			TriggerPercentage: &pct,
			CooldownMinutes:   30,
			IsActive:          true,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.Save(ctx, rule)
		require.NoError(t, err)

		savedRuleID = rule.ID

		saved, err := repo.FindByID(ctx, savedRuleID)
		require.NoError(t, err)
		assert.Equal(t, "SKU-A", saved.SKUCode)
		assert.Equal(t, model.RuleTypeLowStock, saved.RuleType)
		assert.Equal(t, 20.0, *saved.TriggerPercentage)
		assert.Equal(t, true, saved.IsActive)
	})

	t.Run("Find NotFound returns ErrNotFound", func(t *testing.T) {
		_, err := repo.FindByID(ctx, "RULE-999")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no rows in result set")
	})

	t.Run("FindAll with filters", func(t *testing.T) {
		pct := 10.0
		r2 := &model.ThresholdRule{
			ID:                "RULE-002",
			SKUCode:           "SKU-B",
			RuleType:          model.RuleTypeCritical,
			TriggerPercentage: &pct,
			CooldownMinutes:   15,
			IsActive:          false,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		require.NoError(t, repo.Save(ctx, r2))

		all, err := repo.FindAll(ctx, model.ThresholdRuleQuery{})
		require.NoError(t, err)
		assert.Len(t, all, 2)

		active := true
		filtered, err := repo.FindAll(ctx, model.ThresholdRuleQuery{IsActive: &active})
		require.NoError(t, err)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "RULE-001", filtered[0].ID)
	})

	t.Run("FindBySKU", func(t *testing.T) {
		rules, err := repo.FindBySKU(ctx, "SKU-A")
		require.NoError(t, err)
		assert.Len(t, rules, 1)
		assert.Equal(t, "RULE-001", rules[0].ID)
	})

	t.Run("Update rule", func(t *testing.T) {
		rule, err := repo.FindByID(ctx, savedRuleID)
		require.NoError(t, err)

		qty := 50
		rule.TriggerQty = &qty
		rule.IsActive = false
		err = repo.Update(ctx, rule)
		require.NoError(t, err)

		updated, err := repo.FindByID(ctx, savedRuleID)
		require.NoError(t, err)
		assert.Equal(t, 50, *updated.TriggerQty)
		assert.Equal(t, false, updated.IsActive)
	})

	t.Run("Delete rule", func(t *testing.T) {
		err := repo.Delete(ctx, savedRuleID)
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, savedRuleID)
		require.Error(t, err)
	})
}
