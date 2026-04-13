package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"inventory-manage/global"
	"inventory-manage/internal/controller"
	"inventory-manage/internal/model"
	"inventory-manage/pkg/logger"
	"inventory-manage/pkg/response"
)

func init() {
	if global.Logger == nil {
		global.Logger = &logger.LoggerZap{Logger: zap.NewNop()}
	}
}

// mockInventoryRepo implements service.IInventoryRepository
type mockInventoryRepo struct {
	snapshots   []*model.InventorySnapshot
	skuConfig   *model.SKUConfig
	skuConfErr  error
	snapshotErr error
}

func (m *mockInventoryRepo) UpsertSnapshot(ctx context.Context, snapshot *model.InventorySnapshot) error {
	return nil
}

func (m *mockInventoryRepo) GetSnapshotBySKU(ctx context.Context, skuCode string) ([]*model.InventorySnapshot, error) {
	if m.snapshotErr != nil {
		return nil, m.snapshotErr
	}
	// Return the mock array if it matches
	return m.snapshots, nil
}

func (m *mockInventoryRepo) GetCurrentSnapshots(ctx context.Context) ([]*model.InventorySnapshot, error) {
	if m.snapshotErr != nil {
		return nil, m.snapshotErr
	}
	return m.snapshots, nil
}

func (m *mockInventoryRepo) GetSKUConfig(ctx context.Context, skuCode string) (*model.SKUConfig, error) {
	if m.skuConfErr != nil {
		return nil, m.skuConfErr
	}
	return m.skuConfig, nil
}

func TestInventoryController_GetCurrentInventory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSnaps := []*model.InventorySnapshot{
		{DeviceID: "D1", SKUCode: "A1", Qty: 10, Percentage: 50},
	}

	tests := []struct {
		name       string
		repoErr    error
		wantStatus int
		wantCode   int
		wantItems  int
	}{
		{
			name:       "AC-04: successfully return snapshots",
			repoErr:    nil,
			wantStatus: http.StatusOK,
			wantCode:   response.ErrCodeSuccess,
			wantItems:  1,
		},
		{
			name:       "AC-04: internal error",
			repoErr:    errors.New("db error"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   response.ErrCodeInternalServer,
			wantItems:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockInventoryRepo{snapshots: mockSnaps, snapshotErr: tt.repoErr}
			ctrl := controller.NewInventoryController(repo)

			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/inventory/current", nil)
			ctx.Set("trace_id", "test-123")

			ctrl.GetCurrentInventory(ctx)

			assert.Equal(t, tt.wantStatus, w.Code)
			var body map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &body)
			assert.Equal(t, float64(tt.wantCode), body["code"])
			if tt.wantItems > 0 {
				data := body["data"].([]interface{})
				assert.Len(t, data, tt.wantItems)
			}
		})
	}
}

func TestInventoryController_GetInventoryBySKU(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSnaps := []*model.InventorySnapshot{
		{DeviceID: "D1", SKUCode: "A1", Qty: 10, Percentage: 50},
	}

	tests := []struct {
		name       string
		skuCode    string
		confErr    error
		snapErr    error
		wantStatus int
		wantCode   int
	}{
		{
			name:       "AC-05: successfully return SKU snapshots",
			skuCode:    "A1",
			confErr:    nil,
			snapErr:    nil,
			wantStatus: http.StatusOK,
			wantCode:   response.ErrCodeSuccess,
		},
		{
			name:       "AC-05: sku not found",
			skuCode:    "B2",
			confErr:    model.ErrSKUNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   response.ErrCodeDeviceNotFound, // Maps to 40001
		},
		{
			name:       "Missing SKU Code",
			skuCode:    "",
			wantStatus: http.StatusBadRequest,
			wantCode:   response.ErrCodeParamInvalid,
		},
		{
			name:       "Internal error resolving config",
			skuCode:    "A1",
			confErr:    errors.New("db error"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   response.ErrCodeInternalServer,
		},
		{
			name:       "Internal error resolving snapshots",
			skuCode:    "A1",
			snapErr:    errors.New("db error"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   response.ErrCodeInternalServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockInventoryRepo{snapshots: mockSnaps, skuConfErr: tt.confErr, snapshotErr: tt.snapErr}
			ctrl := controller.NewInventoryController(repo)

			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Params = gin.Params{{Key: "sku_code", Value: tt.skuCode}}
			ctx.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/inventory/"+tt.skuCode+"/current", nil)
			ctx.Set("trace_id", "test-123")

			ctrl.GetInventoryBySKU(ctx)

			assert.Equal(t, tt.wantStatus, w.Code)
			var body map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &body)
			assert.Equal(t, float64(tt.wantCode), body["code"])
		})
	}
}
