package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"inventory-manage/internal/domain/device"
	"inventory-manage/internal/handler"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCalibUseCase struct {
	RegisterFunc  func(ctx context.Context, config *device.CalibrationConfig) error
	GetActiveFunc func(ctx context.Context, deviceID string) (*device.CalibrationConfig, error)
}

func (m *mockCalibUseCase) RegisterCalibration(ctx context.Context, config *device.CalibrationConfig) error {
	return m.RegisterFunc(ctx, config)
}

func (m *mockCalibUseCase) GetActiveCalibration(ctx context.Context, deviceID string) (*device.CalibrationConfig, error) {
	return m.GetActiveFunc(ctx, deviceID)
}

func setupCalibRouter(uc device.CalibrationUseCase) chi.Router {
	r := chi.NewRouter()
	h := handler.NewCalibrationHandler(uc)
	h.RegisterRoutes(r)
	return r
}

func TestCalibrationHandler_Create(t *testing.T) {
	uc := &mockCalibUseCase{}
	r := setupCalibRouter(uc)

	t.Run("Valid creation", func(t *testing.T) {
		uc.RegisterFunc = func(ctx context.Context, config *device.CalibrationConfig) error {
			assert.Equal(t, "DEV-001", config.DeviceID)
			return nil
		}
		payload := []byte(`{"unit":"kg"}`)
		req := httptest.NewRequest(http.MethodPost, "/devices/DEV-001/calibration", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		payload := []byte(`{`)
		req := httptest.NewRequest(http.MethodPost, "/devices/DEV-001/calibration", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestCalibrationHandler_GetActive(t *testing.T) {
	uc := &mockCalibUseCase{}
	r := setupCalibRouter(uc)

	t.Run("Status 404", func(t *testing.T) {
		uc.GetActiveFunc = func(ctx context.Context, deviceID string) (*device.CalibrationConfig, error) {
			return nil, device.ErrDeviceNotFound
		}

		req := httptest.NewRequest(http.MethodGet, "/devices/DEV-NOT-FOUND/calibration/active", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Status 200", func(t *testing.T) {
		uc.GetActiveFunc = func(ctx context.Context, deviceID string) (*device.CalibrationConfig, error) {
			return &device.CalibrationConfig{DeviceID: deviceID, Unit: "kg"}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/devices/DEV-001/calibration/active", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	})
}
