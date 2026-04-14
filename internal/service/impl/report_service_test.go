package impl_test

import (
	"context"
	"encoding/json"
	"inventory-manage/global"
	"inventory-manage/internal/model"
	"inventory-manage/internal/service/impl"
	"inventory-manage/pkg/logger"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
    "go.uber.org/zap"
)

func setupTestRedis(t *testing.T) *miniredis.Miniredis {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(func() { mr.Close() })

	global.Rdb = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
    global.Logger = &logger.LoggerZap{Logger: zap.NewNop()}
	t.Cleanup(func() { global.Rdb = nil })
	return mr
}

type mockReportRepo struct {
	mock.Mock
}

func (m *mockReportRepo) GetConsumptionTrend(ctx context.Context, query model.ConsumptionQuery) ([]*model.ConsumptionDataPoint, error) {
	args := m.Called(ctx, query)
	var pts []*model.ConsumptionDataPoint
	if args.Get(0) != nil {
		pts = args.Get(0).([]*model.ConsumptionDataPoint)
	}
	return pts, args.Error(1)
}

func (m *mockReportRepo) GetConsumptionSummary(ctx context.Context, query model.ConsumptionQuery) (*model.ConsumptionSummary, error) {
	args := m.Called(ctx, query)
	var sum *model.ConsumptionSummary
	if args.Get(0) != nil {
		sum = args.Get(0).(*model.ConsumptionSummary)
	}
	return sum, args.Error(1)
}

func TestReportService_GetConsumptionTrend(t *testing.T) {
	t.Run("Valid Span < 7 days no cache", func(t *testing.T) {
		repo := new(mockReportRepo)
		svc := impl.NewReportService(repo)
		global.Rdb = nil

		q := model.ConsumptionQuery{
			SKUCode:  "SKU-1",
			From:     time.Now().Add(-24 * time.Hour),
			To:       time.Now(),
			Interval: "1h",
		}

		pts := []*model.ConsumptionDataPoint{
			{Timestamp: time.Now(), NetWeightKg: 10, Qty: 5, Percentage: 50},
		}
		repo.On("GetConsumptionTrend", mock.Anything, q).Return(pts, nil)

		res, next, err := svc.GetConsumptionTrend(context.Background(), q)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "", next)
	})

	t.Run("Valid Span > 7 days cache miss and store", func(t *testing.T) {
		mr := setupTestRedis(t)
		repo := new(mockReportRepo)
		svc := impl.NewReportService(repo)

		q := model.ConsumptionQuery{
			SKUCode:  "SKU-1",
			From:     time.Now().Add(-10 * 24 * time.Hour), // 10 days span
			To:       time.Now(),
			Interval: "1h",
		}

		pts := []*model.ConsumptionDataPoint{
			{Timestamp: time.Now(), NetWeightKg: 10, Qty: 5, Percentage: 50},
		}
		repo.On("GetConsumptionTrend", mock.Anything, q).Return(pts, nil)

		res, next, err := svc.GetConsumptionTrend(context.Background(), q)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "", next)

		// Assert cache set
		mr.FastForward(time.Minute)
		keys, _ := global.Rdb.Keys(context.Background(), "*").Result()
		assert.NotEmpty(t, keys)
	})

	t.Run("Valid Span > 7 days cache hit", func(t *testing.T) {
		repo := new(mockReportRepo)
		svc := impl.NewReportService(repo)
        setupTestRedis(t)

		q := model.ConsumptionQuery{
			SKUCode:  "SKU-1",
			From:     time.Unix(0, 0),
			To:       time.Unix(0, 0).Add(10 * 24 * time.Hour),
			Interval: "1d",
            Limit:    10,
		}

		// pre-populate cache
		cached := map[string]interface{}{
			"points": []*model.ConsumptionDataPoint{{NetWeightKg: 99}},
			"next_cursor": "cur",
		}
		bytes, _ := json.Marshal(cached)
        global.Rdb.Set(context.Background(), "reports:consumption:trend:SKU-1:0:864000:1d:10:", bytes, 0)

		res, next, err := svc.GetConsumptionTrend(context.Background(), q)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, float64(99), res[0].NetWeightKg)
		assert.Equal(t, "cur", next)
	})

	t.Run("Repo error returns error", func(t *testing.T) {
		repo := new(mockReportRepo)
		svc := impl.NewReportService(repo)
		global.Rdb = nil

		q := model.ConsumptionQuery{
			SKUCode:  "SKU-1",
			From:     time.Now().Add(-24 * time.Hour),
			To:       time.Now(),
		}

		repo.On("GetConsumptionTrend", mock.Anything, q).Return(nil, assert.AnError)

		_, _, err := svc.GetConsumptionTrend(context.Background(), q)
		assert.Error(t, err)
	})
    
    t.Run("Next cursor logic limits reached", func(t *testing.T) {
		repo := new(mockReportRepo)
		svc := impl.NewReportService(repo)
		global.Rdb = nil

		q := model.ConsumptionQuery{
			From:     time.Now().Add(-24 * time.Hour),
			To:       time.Now(),
            Limit:    1,
		}

        ts := time.Now()
		pts := []*model.ConsumptionDataPoint{
			{Timestamp: ts, NetWeightKg: 10},
		}
		repo.On("GetConsumptionTrend", mock.Anything, q).Return(pts, nil)

		res, next, err := svc.GetConsumptionTrend(context.Background(), q)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, ts.Format(time.RFC3339), next)
	})
}

func TestReportService_GetConsumptionSummary(t *testing.T) {
	t.Run("Valid Span < 7 days no cache", func(t *testing.T) {
		repo := new(mockReportRepo)
		svc := impl.NewReportService(repo)
		global.Rdb = nil

		q := model.ConsumptionQuery{
			SKUCode:  "SKU-1",
			From:     time.Now().Add(-24 * time.Hour),
			To:       time.Now(),
			Interval: "1d",
		}

		sum := &model.ConsumptionSummary{TotalConsumptionKg: 20}
		repo.On("GetConsumptionSummary", mock.Anything, q).Return(sum, nil)

		res, err := svc.GetConsumptionSummary(context.Background(), q)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, float64(20), res.TotalConsumptionKg)
	})

	t.Run("Valid Span > 7 days cache hit", func(t *testing.T) {
		setupTestRedis(t)
		repo := new(mockReportRepo)
		svc := impl.NewReportService(repo)

		q := model.ConsumptionQuery{
			SKUCode:  "SKU-1",
			From:     time.Unix(0, 0),
			To:       time.Unix(0, 0).Add(10 * 24 * time.Hour),
			Interval: "1d",
		}

		sum := &model.ConsumptionSummary{TotalConsumptionKg: 50}
		b, _ := json.Marshal(sum)
        global.Rdb.Set(context.Background(), "reports:consumption:summary:SKU-1:0:864000", b, 0)

		res, err := svc.GetConsumptionSummary(context.Background(), q)
		assert.NoError(t, err)
		assert.Equal(t, float64(50), res.TotalConsumptionKg)
	})

	t.Run("Valid Span > 7 days cache miss and store", func(t *testing.T) {
		setupTestRedis(t)
		repo := new(mockReportRepo)
		svc := impl.NewReportService(repo)

		q := model.ConsumptionQuery{
			SKUCode:  "SKU-1",
			From:     time.Unix(1000, 0),
			To:       time.Unix(1000, 0).Add(10 * 24 * time.Hour),
		}

		sum := &model.ConsumptionSummary{TotalConsumptionKg: 30}
		repo.On("GetConsumptionSummary", mock.Anything, q).Return(sum, nil)

		res, err := svc.GetConsumptionSummary(context.Background(), q)
		assert.NoError(t, err)
		assert.Equal(t, float64(30), res.TotalConsumptionKg)

		// Assert cache set
		keys, _ := global.Rdb.Keys(context.Background(), "*").Result()
		assert.NotEmpty(t, keys)
	})

	t.Run("Repo error returns error", func(t *testing.T) {
		repo := new(mockReportRepo)
		svc := impl.NewReportService(repo)
		global.Rdb = nil

		q := model.ConsumptionQuery{
			From:     time.Now().Add(-24 * time.Hour),
			To:       time.Now(),
		}

		repo.On("GetConsumptionSummary", mock.Anything, q).Return(nil, assert.AnError)

		_, err := svc.GetConsumptionSummary(context.Background(), q)
		assert.Error(t, err)
	})
}
