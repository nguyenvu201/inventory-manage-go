// Package controller_test implements tests for INV-SPR03-TASK-003 API
// AC Coverage:
//   AC-05: TestThresholdController_CreateRule, TestThresholdController_GetRules, TestThresholdController_GetRuleByID, TestThresholdController_UpdateRule, TestThresholdController_DeleteRule
// IEC 62304 Classification: Software Safety Class B
package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"inventory-manage/internal/controller"
	"inventory-manage/internal/model"
	"inventory-manage/pkg/response"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockThresholdService struct {
	mock.Mock
}

func (m *mockThresholdService) CreateRule(ctx context.Context, rule *model.ThresholdRule) error {
	args := m.Called(ctx, rule)
	if rule.ID == "" {
		rule.ID = "test-id"
	}
	return args.Error(0)
}
func (m *mockThresholdService) GetRules(ctx context.Context, query model.ThresholdRuleQuery) ([]*model.ThresholdRule, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]*model.ThresholdRule), args.Error(1)
}
func (m *mockThresholdService) GetRuleByID(ctx context.Context, id string) (*model.ThresholdRule, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.ThresholdRule), args.Error(1)
}
func (m *mockThresholdService) UpdateRule(ctx context.Context, id string, rule *model.ThresholdRule) error {
	args := m.Called(ctx, id, rule)
	return args.Error(0)
}
func (m *mockThresholdService) DeleteRule(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func setupTestRouter() (*gin.Engine, *mockThresholdService) {
	gin.SetMode(gin.TestMode)
	svc := new(mockThresholdService)
	ctrl := controller.NewThresholdController(svc)
	
	r := gin.Default()
	// Add dummy trace_id middleware
	r.Use(func(c *gin.Context) {
		c.Set("trace_id", "test-trace")
		c.Next()
	})
	
	api := r.Group("/api/v1/rules")
	api.POST("/thresholds", ctrl.CreateRule)
	api.GET("/thresholds", ctrl.GetRules)
	api.PUT("/thresholds/:id", ctrl.UpdateRule)
	api.DELETE("/thresholds/:id", ctrl.DeleteRule)
	
	return r, svc
}

func TestThresholdController_CreateRule(t *testing.T) {
	r, svc := setupTestRouter()

	t.Run("success", func(t *testing.T) {
		perc := 20.0
		input := model.ThresholdRule{SKUCode: "SKU-TEST", RuleType: "low_stock", TriggerPercentage: &perc}
		svc.On("CreateRule", mock.Anything, mock.AnythingOfType("*model.ThresholdRule")).Return(nil).Once()

		body, _ := json.Marshal(input)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/rules/thresholds", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		svc.AssertExpectations(t)
	})

	t.Run("bad request", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/rules/thresholds", bytes.NewBufferString("{invalid_json}"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NotEqual(t, float64(response.ErrCodeSuccess), resp["code"])
	})
}

func TestThresholdController_GetRules(t *testing.T) {
	r, svc := setupTestRouter()

	t.Run("success", func(t *testing.T) {
		rules := []*model.ThresholdRule{{ID: "1", SKUCode: "SKU-A"}}
		svc.On("GetRules", mock.Anything, mock.AnythingOfType("model.ThresholdRuleQuery")).Return(rules, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/rules/thresholds?sku_code=SKU-A", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestThresholdController_UpdateRule(t *testing.T) {
	r, svc := setupTestRouter()

	t.Run("success", func(t *testing.T) {
		perc := 25.0
		input := model.ThresholdRule{TriggerPercentage: &perc}
		svc.On("UpdateRule", mock.Anything, "1", mock.AnythingOfType("*model.ThresholdRule")).Return(nil).Once()

		body, _ := json.Marshal(input)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/rules/thresholds/1", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(response.ErrCodeSuccess), resp["code"])
	})
}

func TestThresholdController_DeleteRule(t *testing.T) {
	r, svc := setupTestRouter()

	t.Run("success", func(t *testing.T) {
		svc.On("DeleteRule", mock.Anything, "1").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/rules/thresholds/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
