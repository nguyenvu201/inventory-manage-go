package initialize

import (
	"context"
	"fmt"
	"time"

	"inventory-manage/global"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// InitPostgres creates a pgx/v5 connection pool using settings from global.Config.Postgres.
// The pool is stored in global.Pdb and is reused by all repository implementations.
func InitPostgres() {
	p := global.Config.Postgres

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.Username, p.Password, p.Host, p.Port, p.DBName, p.SSLMode,
	)

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		loggerOrPanic("InitPostgres: failed to parse DSN", err)
	}

	cfg.MaxConns = p.MaxConns
	if p.MinConns > 0 {
		cfg.MinConns = p.MinConns
	}
	if p.ConnMaxLifetime > 0 {
		cfg.MaxConnLifetime = time.Duration(p.ConnMaxLifetime) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		loggerOrPanic("InitPostgres: failed to create pool", err)
	}

	if err := pool.Ping(ctx); err != nil {
		loggerOrPanic("InitPostgres: ping failed", err)
	}

	global.Pdb = pool
	global.Logger.Info("PostgreSQL connected successfully",
		zap.String("host", p.Host),
		zap.Int("port", p.Port),
		zap.String("db", p.DBName),
	)
}
