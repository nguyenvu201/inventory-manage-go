package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"inventory-manage/internal/model"
	"inventory-manage/internal/service"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type thresholdRepository struct {
	db *pgxpool.Pool
}

// NewThresholdRepository creates a new threshold repository implementation
func NewThresholdRepository(db *pgxpool.Pool) service.IThresholdRepository {
	return &thresholdRepository{db: db}
}

func (r *thresholdRepository) Save(ctx context.Context, rule *model.ThresholdRule) error {
	query, args, err := sq.Insert("threshold_rules").
		Columns("sku_code", "rule_type", "trigger_percentage", "trigger_qty", "cooldown_minutes", "is_active").
		Values(rule.SKUCode, rule.RuleType, rule.TriggerPercentage, rule.TriggerQty, rule.CooldownMinutes, rule.IsActive).
		Suffix("RETURNING id, created_at, updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("sq.Insert: %w", err)
	}

	err = r.db.QueryRow(ctx, query, args...).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("ThresholdRepository.Save: %w", err)
	}
	return nil
}



func (r *thresholdRepository) FindByID(ctx context.Context, id string) (*model.ThresholdRule, error) {
	query, args, err := sq.Select("id", "sku_code", "rule_type", "trigger_percentage", "trigger_qty", "cooldown_minutes", "is_active", "created_at", "updated_at").
		From("threshold_rules").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("sq.Select: %w", err)
	}

	rule := &model.ThresholdRule{}
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&rule.ID, &rule.SKUCode, &rule.RuleType, &rule.TriggerPercentage,
		&rule.TriggerQty, &rule.CooldownMinutes, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("ThresholdRepository.FindByID: no rule found with id %s", id)
		}
		return nil, fmt.Errorf("ThresholdRepository.FindByID: %w", err)
	}
	return rule, nil
}

func (r *thresholdRepository) FindBySKU(ctx context.Context, skuCode string) ([]*model.ThresholdRule, error) {
	query, args, err := sq.Select("id", "sku_code", "rule_type", "trigger_percentage", "trigger_qty", "cooldown_minutes", "is_active", "created_at", "updated_at").
		From("threshold_rules").
		Where(sq.Eq{"sku_code": skuCode, "is_active": true}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("sq.Select: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ThresholdRepository.FindBySKU: %w", err)
	}
	defer rows.Close()

	var rules []*model.ThresholdRule
	for rows.Next() {
		rule := &model.ThresholdRule{}
		err := rows.Scan(
			&rule.ID, &rule.SKUCode, &rule.RuleType, &rule.TriggerPercentage,
			&rule.TriggerQty, &rule.CooldownMinutes, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ThresholdRepository.FindBySKU scan: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *thresholdRepository) Update(ctx context.Context, rule *model.ThresholdRule) error {
	rule.UpdatedAt = time.Now()
	query, args, err := sq.Update("threshold_rules").
		Set("trigger_percentage", rule.TriggerPercentage).
		Set("trigger_qty", rule.TriggerQty).
		Set("cooldown_minutes", rule.CooldownMinutes).
		Set("is_active", rule.IsActive).
		Set("updated_at", rule.UpdatedAt).
		Where(sq.Eq{"id": rule.ID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("sq.Update: %w", err)
	}

	cmdTag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ThresholdRepository.Update: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("ThresholdRepository.Update: no rule found with id %s", rule.ID)
	}
	return nil
}

func (r *thresholdRepository) Delete(ctx context.Context, id string) error {
	query, args, err := sq.Delete("threshold_rules").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("sq.Delete: %w", err)
	}

	cmdTag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ThresholdRepository.Delete: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("ThresholdRepository.Delete: no rule found with id %s", id)
	}
	return nil
}

func (r *thresholdRepository) FindAll(ctx context.Context, q model.ThresholdRuleQuery) ([]*model.ThresholdRule, error) {
	builder := sq.Select("id", "sku_code", "rule_type", "trigger_percentage", "trigger_qty", "cooldown_minutes", "is_active", "created_at", "updated_at").
		From("threshold_rules").
		PlaceholderFormat(sq.Dollar)

	if q.SKUCode != nil {
		builder = builder.Where(sq.Eq{"sku_code": *q.SKUCode})
	}
	if q.Limit > 0 {
		builder = builder.Limit(q.Limit)
	}
	if q.Offset > 0 {
		builder = builder.Offset(q.Offset)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("sq.Select: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ThresholdRepository.FindAll: %w", err)
	}
	defer rows.Close()

	var rules []*model.ThresholdRule
	for rows.Next() {
		rule := &model.ThresholdRule{}
		err := rows.Scan(
			&rule.ID, &rule.SKUCode, &rule.RuleType, &rule.TriggerPercentage,
			&rule.TriggerQty, &rule.CooldownMinutes, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ThresholdRepository.FindAll scan: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}
