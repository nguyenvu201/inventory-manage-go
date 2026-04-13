package global

import (
	"inventory-manage/pkg/logger"
	"inventory-manage/pkg/setting"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	// Config holds the full application configuration loaded from YAML.
	Config setting.Config

	// Logger is the structured Zap logger (shared across all packages).
	Logger *logger.LoggerZap

	// Pdb is the PostgreSQL connection pool (pgx/v5).
	Pdb *pgxpool.Pool

	// Rdb is the Redis client.
	Rdb *redis.Client
)
