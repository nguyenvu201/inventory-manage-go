package usecase_test

import (
	"context"
	"testing"
	"time"

	"inventory-manage/internal/domain/device"
	"inventory-manage/internal/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDeviceRepo struct {
	SaveFunc     func(ctx context.Context, d *device.Device) error
	FindByIDFunc func(ctx context.Context, id string) (*device.Device, error)
	FindAllFunc  func(ctx context.Context, q device.DeviceQuery) ([]*device.Device, error)
	UpdateFunc   func(ctx context.Context, d *device.Device) error
	DeleteFunc   func(ctx context.Context, id string) error
}

func (m *mockDeviceRepo) Save(ctx context.Context, d *device.Device) error { return m.SaveFunc(ctx, d) }
func (m *mockDeviceRepo) FindByID(ctx context.Context, id string) (*device.Device, error) {
	return m.FindByIDFunc(ctx, id)
}
func (m *mockDeviceRepo) FindAll(ctx context.Context, q device.DeviceQuery) ([]*device.Device, error) {
	return m.FindAllFunc(ctx, q)
}
func (m *mockDeviceRepo) Update(ctx context.Context, d *device.Device) error {
	return m.UpdateFunc(ctx, d)
}
func (m *mockDeviceRepo) Delete(ctx context.Context, id string) error { return m.DeleteFunc(ctx, id) }

func TestDeviceUseCase_RegisterDevice(t *testing.T) {
	repo := &mockDeviceRepo{}
	uc := usecase.NewDeviceUseCase(repo)
	ctx := context.Background()

	tests := []struct {
		name    string
		input   *device.Device
		setup   func()
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid device",
			input: &device.Device{
				DeviceID: "D1",
				SKUCode:  "S1",
				Status:   device.StatusActive,
			},
			setup: func() {
				repo.SaveFunc = func(ctx context.Context, d *device.Device) error { return nil }
			},
			wantErr: false,
		},
		{
			name: "Missing ID",
			input: &device.Device{
				SKUCode: "S1",
				Status:  device.StatusActive,
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "device_id is required",
		},
		{
			name: "Missing SKU",
			input: &device.Device{
				DeviceID: "D1",
				Status:   device.StatusActive,
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "sku_code is required",
		},
		{
			name: "Invalid Status",
			input: &device.Device{
				DeviceID: "D1",
				SKUCode:  "S1",
				Status:   "broken",
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  device.ErrInvalidStatus.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := uc.RegisterDevice(ctx, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDeviceUseCase_UpdateDevice(t *testing.T) {
	repo := &mockDeviceRepo{}
	uc := usecase.NewDeviceUseCase(repo)
	ctx := context.Background()

	t.Run("Valid Update", func(t *testing.T) {
		repo.FindByIDFunc = func(ctx context.Context, id string) (*device.Device, error) {
			return &device.Device{DeviceID: "D1", Name: "Old", SKUCode: "OldS", Status: device.StatusActive, CreatedAt: time.Now()}, nil
		}
		repo.UpdateFunc = func(ctx context.Context, d *device.Device) error {
			assert.Equal(t, "New", d.Name)
			assert.Equal(t, "NewS", d.SKUCode)
			return nil
		}

		err := uc.UpdateDevice(ctx, &device.Device{DeviceID: "D1", Name: "New", SKUCode: "NewS", Status: device.StatusActive})
		require.NoError(t, err)
	})

	t.Run("Not Found", func(t *testing.T) {
		repo.FindByIDFunc = func(ctx context.Context, id string) (*device.Device, error) {
			return nil, device.ErrDeviceNotFound
		}

		err := uc.UpdateDevice(ctx, &device.Device{DeviceID: "D1", Status: device.StatusActive})
		require.ErrorIs(t, err, device.ErrDeviceNotFound)
	})

	t.Run("Missing ID", func(t *testing.T) {
		err := uc.UpdateDevice(ctx, &device.Device{Status: device.StatusActive})
		assert.EqualError(t, err, "device_id is required")
	})

	t.Run("Invalid Status", func(t *testing.T) {
		err := uc.UpdateDevice(ctx, &device.Device{DeviceID: "D1", Status: "broken"})
		assert.ErrorIs(t, err, device.ErrInvalidStatus)
	})
}

func TestDeviceUseCase_GetDevice(t *testing.T) {
	repo := &mockDeviceRepo{}
	uc := usecase.NewDeviceUseCase(repo)
	ctx := context.Background()

	t.Run("Valid", func(t *testing.T) {
		repo.FindByIDFunc = func(ctx context.Context, id string) (*device.Device, error) {
			assert.Equal(t, "D1", id)
			return &device.Device{DeviceID: "D1"}, nil
		}
		d, err := uc.GetDevice(ctx, "D1")
		require.NoError(t, err)
		assert.Equal(t, "D1", d.DeviceID)
	})

	t.Run("Missing ID", func(t *testing.T) {
		_, err := uc.GetDevice(ctx, "")
		assert.EqualError(t, err, "device_id is required")
	})
}

func TestDeviceUseCase_ListDevices(t *testing.T) {
	repo := &mockDeviceRepo{}
	uc := usecase.NewDeviceUseCase(repo)
	ctx := context.Background()

	t.Run("Valid", func(t *testing.T) {
		repo.FindAllFunc = func(ctx context.Context, q device.DeviceQuery) ([]*device.Device, error) {
			return []*device.Device{{DeviceID: "D1"}, {DeviceID: "D2"}}, nil
		}

		res, err := uc.ListDevices(ctx, device.DeviceQuery{})
		require.NoError(t, err)
		assert.Len(t, res, 2)
	})
}

func TestDeviceUseCase_RemoveDevice(t *testing.T) {
	repo := &mockDeviceRepo{}
	uc := usecase.NewDeviceUseCase(repo)
	ctx := context.Background()

	t.Run("Valid", func(t *testing.T) {
		repo.DeleteFunc = func(ctx context.Context, id string) error {
			assert.Equal(t, "D1", id)
			return nil
		}
		err := uc.RemoveDevice(ctx, "D1")
		require.NoError(t, err)
	})

	t.Run("Missing ID", func(t *testing.T) {
		err := uc.RemoveDevice(ctx, "")
		assert.EqualError(t, err, "device_id is required")
	})
}
