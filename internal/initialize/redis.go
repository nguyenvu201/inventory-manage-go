package initialize

import (
	"context"
	"fmt"
	"time"

	"inventory-manage/global"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	redisMaxRetries = 3
)

// InitRedis creates a Redis client and verifies connectivity with exponential backoff.
// Redis is OPTIONAL — the service starts in degraded mode (no cache) if unavailable.
// Any code using global.Rdb must guard with: if global.Rdb != nil { ... }
func InitRedis() {
	r := global.Config.Redis

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", r.Host, r.Port),
		Password: r.Password,
		DB:       r.Database,
		PoolSize: r.PoolSize,
	})

	ctx := context.Background()

	// Retry with exponential backoff
	for attempt := 1; attempt <= redisMaxRetries; attempt++ {
		_, err := rdb.Ping(ctx).Result()
		if err == nil {
			global.Rdb = rdb
			global.Logger.Info("Redis connected successfully",
				zap.String("addr", fmt.Sprintf("%s:%d", r.Host, r.Port)),
				zap.Int("db", r.Database),
			)
			return
		}

		backoff := time.Duration(attempt*attempt) * time.Second
		global.Logger.Warn("Redis connection failed, retrying...",
			zap.Int("attempt", attempt),
			zap.Duration("backoff", backoff),
			zap.Error(err),
		)
		time.Sleep(backoff)
	}

	// Graceful degradation — Redis is not available, service continues without cache.
	global.Logger.Warn("⚠️  Redis unavailable — service starting in DEGRADED mode (caching disabled)",
		zap.String("addr", fmt.Sprintf("%s:%d", r.Host, r.Port)),
	)
	global.Rdb = nil
}
