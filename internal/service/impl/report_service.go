package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"inventory-manage/global"
	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
)

type ReportServiceImpl struct {
	repo service.IReportRepository
}

func NewReportService(repo service.IReportRepository) service.IReportService {
	return &ReportServiceImpl{repo: repo}
}

func (s *ReportServiceImpl) GetConsumptionTrend(ctx context.Context, query model.ConsumptionQuery) ([]*model.ConsumptionDataPoint, string, error) {
	// Cache logic: cache if span > 7 days
	span := query.To.Sub(query.From)
	useCache := span > 7*24*time.Hour && global.Rdb != nil

	cacheKey := ""
	if useCache {
		cacheKey = fmt.Sprintf("reports:consumption:trend:%s:%d:%d:%s:%d:%s",
			query.SKUCode, query.From.Unix(), query.To.Unix(), query.Interval, query.Limit, query.Cursor)

		val, err := global.Rdb.Get(ctx, cacheKey).Result()
		if err == nil {
			var cached struct {
				Points     []*model.ConsumptionDataPoint `json:"points"`
				NextCursor string                        `json:"next_cursor"`
			}
			if unmarshalErr := json.Unmarshal([]byte(val), &cached); unmarshalErr == nil {
				return cached.Points, cached.NextCursor, nil
			}
		} else if err != redis.Nil {
			global.Logger.Warn("Failed to get cache for GetConsumptionTrend", zap.Error(err), zap.String("sku_code", query.SKUCode))
		}
	}

	points, err := s.repo.GetConsumptionTrend(ctx, query)
	if err != nil {
		return nil, "", fmt.Errorf("ReportService.GetConsumptionTrend: %w", err)
	}

	nextCursor := ""
	if len(points) > 0 && query.Limit > 0 && len(points) == query.Limit {
		// next cursor is the timestamp of the last item
		nextCursor = points[len(points)-1].Timestamp.Format(time.RFC3339)
	}

	if useCache {
		cachedMap := map[string]interface{}{
			"points":      points,
			"next_cursor": nextCursor,
		}
		if bytes, err := json.Marshal(cachedMap); err == nil {
			errSet := global.Rdb.Set(ctx, cacheKey, bytes, 5*time.Minute).Err()
			if errSet != nil {
				global.Logger.Warn("Failed to set cache for GetConsumptionTrend", zap.Error(errSet))
			}
		}
	}

	return points, nextCursor, nil
}

func (s *ReportServiceImpl) GetConsumptionSummary(ctx context.Context, query model.ConsumptionQuery) (*model.ConsumptionSummary, error) {
	span := query.To.Sub(query.From)
	useCache := span > 7*24*time.Hour && global.Rdb != nil

	cacheKey := ""
	if useCache {
		cacheKey = fmt.Sprintf("reports:consumption:summary:%s:%d:%d", query.SKUCode, query.From.Unix(), query.To.Unix())
		val, err := global.Rdb.Get(ctx, cacheKey).Result()
		if err == nil {
			var summary model.ConsumptionSummary
			if unmarshalErr := json.Unmarshal([]byte(val), &summary); unmarshalErr == nil {
				return &summary, nil
			}
		} else if err != redis.Nil {
			global.Logger.Warn("Failed to get cache for GetConsumptionSummary", zap.Error(err))
		}
	}

	summary, err := s.repo.GetConsumptionSummary(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ReportService.GetConsumptionSummary: %w", err)
	}

	if useCache && summary != nil {
		if bytes, err := json.Marshal(summary); err == nil {
			errSet := global.Rdb.Set(ctx, cacheKey, bytes, 5*time.Minute).Err()
			if errSet != nil {
				global.Logger.Warn("Failed to set cache for GetConsumptionSummary", zap.Error(errSet))
			}
		}
	}

	return summary, nil
}
