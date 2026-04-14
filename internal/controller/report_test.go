package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"inventory-manage/internal/controller"
	"inventory-manage/internal/model"
	"inventory-manage/pkg/response"
)

type mockReportService struct {
	mock.Mock
}

func (m *mockReportService) GetConsumptionTrend(ctx context.Context, query model.ConsumptionQuery) ([]*model.ConsumptionDataPoint, string, error) {
	args := m.Called(ctx, query)
	var pts []*model.ConsumptionDataPoint
	if args.Get(0) != nil {
		pts = args.Get(0).([]*model.ConsumptionDataPoint)
	}
	return pts, args.String(1), args.Error(2)
}

func (m *mockReportService) GetConsumptionSummary(ctx context.Context, query model.ConsumptionQuery) (*model.ConsumptionSummary, error) {
	args := m.Called(ctx, query)
	var sum *model.ConsumptionSummary
	if args.Get(0) != nil {
		sum = args.Get(0).(*model.ConsumptionSummary)
	}
	return sum, args.Error(1)
}

func TestReportController_GetConsumptionTrend(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := new(mockReportService)
		ctrl := controller.NewReportController(mockSvc)
		r := gin.Default()
		r.GET("/api/v1/reports/consumption", ctrl.GetConsumptionTrend)

		pts := []*model.ConsumptionDataPoint{
			{Timestamp: time.Now(), NetWeightKg: 10, Qty: 2, Percentage: 50},
		}
		mockSvc.On("GetConsumptionTrend", mock.Anything, mock.AnythingOfType("model.ConsumptionQuery")).Return(pts, "next_cursor", nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/reports/consumption?sku_code=SKU-A&from=2026-04-01T00:00:00Z&to=2026-04-10T00:00:00Z&interval=1d", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(response.ErrCodeSuccess), resp["code"])
	})

	t.Run("bad request", func(t *testing.T) {
		mockSvc := new(mockReportService)
		ctrl := controller.NewReportController(mockSvc)
		r := gin.Default()
		r.GET("/api/v1/reports/consumption", ctrl.GetConsumptionTrend)

		// missing required params
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/reports/consumption", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := new(mockReportService)
		ctrl := controller.NewReportController(mockSvc)
		r := gin.Default()
		r.GET("/api/v1/reports/consumption", ctrl.GetConsumptionTrend)

		mockSvc.On("GetConsumptionTrend", mock.Anything, mock.AnythingOfType("model.ConsumptionQuery")).Return(nil, "", assert.AnError)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/reports/consumption?sku_code=SKU-A&from=2026-04-01T00:00:00Z&to=2026-04-10T00:00:00Z&interval=1d", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(response.ErrCodeInternalServer), resp["code"])
	})
}

func TestReportController_GetConsumptionSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := new(mockReportService)
		ctrl := controller.NewReportController(mockSvc)
		r := gin.Default()
		r.GET("/api/v1/reports/consumption/summary", ctrl.GetConsumptionSummary)

		sum := &model.ConsumptionSummary{SKUCode: "SKU-A", TotalConsumptionKg: 50}
		mockSvc.On("GetConsumptionSummary", mock.Anything, mock.AnythingOfType("model.ConsumptionQuery")).Return(sum, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/reports/consumption/summary?sku_code=SKU-A&from=2026-04-01T00:00:00Z&to=2026-04-10T00:00:00Z&interval=1d", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(response.ErrCodeSuccess), resp["code"])
	})

	t.Run("bad request", func(t *testing.T) {
		mockSvc := new(mockReportService)
		ctrl := controller.NewReportController(mockSvc)
		r := gin.Default()
		r.GET("/api/v1/reports/consumption/summary", ctrl.GetConsumptionSummary)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/reports/consumption/summary", bytes.NewBufferString("{invalid_json}"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := new(mockReportService)
		ctrl := controller.NewReportController(mockSvc)
		r := gin.Default()
		r.GET("/api/v1/reports/consumption/summary", ctrl.GetConsumptionSummary)

		mockSvc.On("GetConsumptionSummary", mock.Anything, mock.AnythingOfType("model.ConsumptionQuery")).Return(nil, assert.AnError)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/reports/consumption/summary?sku_code=SKU-A&from=2026-04-01T00:00:00Z&to=2026-04-10T00:00:00Z&interval=1d", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(response.ErrCodeInternalServer), resp["code"])
	})
}
