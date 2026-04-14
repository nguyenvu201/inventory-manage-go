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

	pool, _ := setupTestDB(t)
	ctx := context.Background()
	repo := postgres.NewThresholdRepository(pool)

	// seed sku_config dependency
	_, err := pool.Exec(ctx, `
		INSERT INTO sku_configs (sku_code, unit_weight_kg, full_capacity_kg, tare_weight_kg, resolution_kg, reorder_point_qty, unit_label)
		VALUES ('SKU-A', 2.0, 100.0, 1.0, 0.5, 5, 'Box'), ('SKU-B', 1.0, 50.0, 0.5, 0.1, 10, 'Bag')
		ON CONFLICT DO NOTHING`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "TRUNCATE threshold_rules CASCADE")
	require.NoError(t, err)

	var savedRuleID string

	t.Run("Save new rule", func(t *testing.T) {
		pct := 20.0
		rule := &model.ThresholdRule{
			// ID left empty — generate UUID on save via gen_random_uuid() default
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
		require.NotEmpty(t, rule.ID, "repo.Save should populate rule.ID with generated UUID")

		savedRuleID = rule.ID

		saved, err := repo.FindByID(ctx, savedRuleID)
		require.NoError(t, err)
		assert.Equal(t, "SKU-A", saved.SKUCode)
		assert.Equal(t, model.RuleTypeLowStock, saved.RuleType)
		assert.Equal(t, 20.0, *saved.TriggerPercentage)
		assert.Equal(t, true, saved.IsActive)
	})

	t.Run("Find NotFound returns error", func(t *testing.T) {
		// Use a valid UUID that does not exist
		_, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		require.Error(t, err)
	})

	t.Run("FindAll with SKUCode filter", func(t *testing.T) {
		pct := 10.0
		r2 := &model.ThresholdRule{
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

		skuFilter := "SKU-A"
		filtered, err := repo.FindAll(ctx, model.ThresholdRuleQuery{SKUCode: &skuFilter})
		require.NoError(t, err)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "SKU-A", filtered[0].SKUCode)
	})

	t.Run("FindBySKU", func(t *testing.T) {
		rules, err := repo.FindBySKU(ctx, "SKU-A")
		require.NoError(t, err)
		assert.Len(t, rules, 1)
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
