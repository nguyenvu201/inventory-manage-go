package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"inventory-manage/internal/domain/device"
	"inventory-manage/internal/handler"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDeviceUseCase struct {
	RegisterFunc func(ctx context.Context, d *device.Device) error
	GetFunc      func(ctx context.Context, id string) (*device.Device, error)
	ListFunc     func(ctx context.Context, q device.DeviceQuery) ([]*device.Device, error)
	UpdateFunc   func(ctx context.Context, d *device.Device) error
	RemoveFunc   func(ctx context.Context, id string) error
}

func (m *mockDeviceUseCase) RegisterDevice(ctx context.Context, d *device.Device) error {
	return m.RegisterFunc(ctx, d)
}
func (m *mockDeviceUseCase) GetDevice(ctx context.Context, id string) (*device.Device, error) {
	return m.GetFunc(ctx, id)
}
func (m *mockDeviceUseCase) ListDevices(ctx context.Context, q device.DeviceQuery) ([]*device.Device, error) {
	return m.ListFunc(ctx, q)
}
func (m *mockDeviceUseCase) UpdateDevice(ctx context.Context, d *device.Device) error {
	return m.UpdateFunc(ctx, d)
}
func (m *mockDeviceUseCase) RemoveDevice(ctx context.Context, id string) error {
	return m.RemoveFunc(ctx, id)
}

func setupRouter(uc device.UseCase) chi.Router {
	r := chi.NewRouter()
	h := handler.NewDeviceHandler(uc)
	h.RegisterRoutes(r)
	return r
}

func TestDeviceHandler_Create(t *testing.T) {
	uc := &mockDeviceUseCase{}
	r := setupRouter(uc)

	t.Run("Valid creation", func(t *testing.T) {
		uc.RegisterFunc = func(ctx context.Context, d *device.Device) error {
			assert.Equal(t, "DEV-001", d.DeviceID)
			return nil
		}

		payload := []byte(`{"device_id":"DEV-001", "name":"Test"}`)
		req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader(payload))
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("Duplicate device", func(t *testing.T) {
		uc.RegisterFunc = func(ctx context.Context, d *device.Device) error {
			return device.ErrDuplicateDevice
		}

		payload := []byte(`{"device_id":"DEV-001"}`)
		req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader(payload))
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("Internal Error", func(t *testing.T) {
		uc.RegisterFunc = func(ctx context.Context, d *device.Device) error {
			return context.DeadlineExceeded // random error to trigger 400 bad request in current logic
		}

		payload := []byte(`{"device_id":"DEV-001"}`)
		req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader(payload))
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		payload := []byte(`{"device_id":"DEV-001"`) // broken json
		req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestDeviceHandler_Get(t *testing.T) {
	uc := &mockDeviceUseCase{}
	r := setupRouter(uc)

	t.Run("Status 404", func(t *testing.T) {
		uc.GetFunc = func(ctx context.Context, id string) (*device.Device, error) {
			return nil, device.ErrDeviceNotFound
		}

		req := httptest.NewRequest(http.MethodGet, "/devices/DEV-404", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
		
		var res map[string]interface{}
		json.NewDecoder(rec.Body).Decode(&res)
		assert.Equal(t, "error", res["status"])
	})

	t.Run("Valid Get", func(t *testing.T) {
		uc.GetFunc = func(ctx context.Context, id string) (*device.Device, error) {
			return &device.Device{DeviceID: id}, nil
		}
		req := httptest.NewRequest(http.MethodGet, "/devices/DEV-1", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Internal error", func(t *testing.T) {
		uc.GetFunc = func(ctx context.Context, id string) (*device.Device, error) {
			return nil, context.Canceled
		}
		req := httptest.NewRequest(http.MethodGet, "/devices/DEV-1", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestDeviceHandler_List(t *testing.T) {
	uc := &mockDeviceUseCase{}
	r := setupRouter(uc)

	t.Run("List with filtering", func(t *testing.T) {
		uc.ListFunc = func(ctx context.Context, q device.DeviceQuery) ([]*device.Device, error) {
			assert.NotNil(t, q.Status)
			assert.Equal(t, device.StatusActive, *q.Status)
			return []*device.Device{{DeviceID: "D1"}}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/devices?status=active&sku_code=ABC", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("List internal error", func(t *testing.T) {
		uc.ListFunc = func(ctx context.Context, q device.DeviceQuery) ([]*device.Device, error) {
			return nil, context.DeadlineExceeded
		}

		req := httptest.NewRequest(http.MethodGet, "/devices", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestDeviceHandler_Update(t *testing.T) {
	uc := &mockDeviceUseCase{}
	r := setupRouter(uc)

	t.Run("Update OK", func(t *testing.T) {
		uc.UpdateFunc = func(ctx context.Context, d *device.Device) error {
			assert.Equal(t, "DEV-001", d.DeviceID)
			assert.Equal(t, "NewName", d.Name)
			return nil
		}
		uc.GetFunc = func(ctx context.Context, id string) (*device.Device, error) {
			return &device.Device{DeviceID: id, Name: "NewName"}, nil
		}

		payload := []byte(`{"name":"NewName"}`)
		req := httptest.NewRequest(http.MethodPut, "/devices/DEV-001", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Update 404", func(t *testing.T) {
		uc.UpdateFunc = func(ctx context.Context, d *device.Device) error {
			return device.ErrDeviceNotFound
		}

		payload := []byte(`{"name":"NewName"}`)
		req := httptest.NewRequest(http.MethodPut, "/devices/DEV-404", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Update Invalid JSON", func(t *testing.T) {
		payload := []byte(`{`)
		req := httptest.NewRequest(http.MethodPut, "/devices/DEV-1", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestDeviceHandler_Delete(t *testing.T) {
	uc := &mockDeviceUseCase{}
	r := setupRouter(uc)

	t.Run("Delete OK", func(t *testing.T) {
		uc.RemoveFunc = func(ctx context.Context, id string) error {
			return nil
		}

		req := httptest.NewRequest(http.MethodDelete, "/devices/DEV-1", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Delete 404", func(t *testing.T) {
		uc.RemoveFunc = func(ctx context.Context, id string) error {
			return device.ErrDeviceNotFound
		}

		req := httptest.NewRequest(http.MethodDelete, "/devices/DEV-404", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}
