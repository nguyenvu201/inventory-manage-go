//go:build integration
// +build integration

// Package postgres_test implements tests for INV-SPR01-TASK-004
// AC Coverage:
//   AC-03: Implement TelemetryRepository + PostgreSQL implementation
//   AC-04: Batch insert tests
//   AC-06: Write an integration test
//   AC-07: Unique constraint on (device_id, f_cnt)
// IEC 62304 Classification: Software Safety Class B
package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq" // needed for golang-migrate
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	testpg "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/golang-migrate/migrate/v4"
	migpg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"inventory-manage/internal/model"
	"inventory-manage/internal/repository/postgres"
)

func runMigrations(t testing.TB, connStr string) {
	// Locate migrations relative to test file execution
	dir, err := os.Getwd()
	require.NoError(t, err)

	// Since we are inside internal/repository/postgres, migrations is 3 levels up
	migrationsPath := filepath.Join(dir, "..", "..", "..", "migrations")
	
	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)
	defer db.Close()

	// Wait for DB to be truly ready (testcontainers sometimes returns slightly early)
	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.NoError(t, db.Ping(), "Database not ready in time")

	driver, err := migpg.WithInstance(db, &migpg.Config{})
	require.NoError(t, err)

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres", driver)
	require.NoError(t, err)

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err, "Migration failed")
	}
}

func setupTestDB(t testing.TB) (*pgxpool.Pool, context.Context) {
	ctx := context.Background()

	pgContainer, err := testpg.RunContainer(ctx,
		testcontainers.WithImage("timescale/timescaledb:latest-pg15"),
		testpg.WithDatabase("test_inventory"),
		testpg.WithUsername("test_user"),
		testpg.WithPassword("test_pass"),
	)
	require.NoError(t, err)

	t.Cleanup(func() { pgContainer.Terminate(ctx) })

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Run migrations
	runMigrations(t, connStr)

	// Setup PGX Pool
	config, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err)

	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	
	t.Cleanup(func() { pool.Close() })

	return pool, ctx
}

func ptr[T any](v T) *T {
	return &v
}

func TestTelemetryRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, ctx := setupTestDB(t)
	repo := postgres.NewTelemetryRepository(pool)

	t.Run("AC-03: Save valid telemetry record", func(t *testing.T) {
		record := &model.RawTelemetry{
			DeviceID:        "SCALE-INT-01",
			RawWeight:       5000.0,
			BatteryLevel:    85,
			FCnt:            ptr(uint32(1001)),
			RSSI:            -75,
			SNR:             8.5,
			SpreadingFactor: 9,
			SampleCount:     1,
			PayloadJSON:     []byte(`{"raw":true}`),
			ReceivedAt:      time.Now(),
		}
		err := repo.Save(ctx, record)
		require.NoError(t, err)
		assert.NotZero(t, record.ID)

		// Verify retrieval
		res, err := repo.FindByDeviceID(ctx, model.TelemetryQuery{DeviceID: "SCALE-INT-01"})
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, float64(5000.0), res[0].RawWeight)
	})

	t.Run("AC-07: Duplicate f_cnt returns ErrDuplicatePacket", func(t *testing.T) {
		record := &model.RawTelemetry{
			DeviceID:     "SCALE-INT-02",
			RawWeight:    5000.0,
			BatteryLevel: 85,
			FCnt:         ptr(uint32(9999)),
			ReceivedAt:   time.Now(),
		}
		require.NoError(t, repo.Save(ctx, record))

		// Second insert with same f_cnt must fail uniquely
		err := repo.Save(ctx, record)
		require.ErrorIs(t, err, model.ErrDuplicatePacket)
	})

	t.Run("AC-04: Batch Insert and idempotency inside batch", func(t *testing.T) {
		now := time.Now()
		var batch []*model.RawTelemetry
		for i := 0; i < 15; i++ {
			batch = append(batch, &model.RawTelemetry{
				DeviceID:     "SCALE-BATCH",
				RawWeight:    float64(10 * i),
				BatteryLevel: int8(i),
				FCnt:         ptr(uint32(10 + i)),
				ReceivedAt:   now.Add(time.Duration(i) * time.Second),
			})
		}

		err := repo.SaveBatch(ctx, batch)
		require.NoError(t, err)

		results, err := repo.FindByDeviceID(ctx, model.TelemetryQuery{DeviceID: "SCALE-BATCH"})
		require.NoError(t, err)
		assert.Len(t, results, 15)

		// Test batch swallowing duplicates silently
		duplicateBatch := []*model.RawTelemetry{batch[0], batch[1]}
		err = repo.SaveBatch(ctx, duplicateBatch)
		require.NoError(t, err, "SaveBatch must silently discard uniqueness violations")
		
		postCount, _ := repo.FindByDeviceID(ctx, model.TelemetryQuery{DeviceID: "SCALE-BATCH"})
		assert.Len(t, postCount, 15, "Count should not increase")
	})
}
