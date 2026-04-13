package impl_test

import (
	"context"
	"fmt"
	"testing"

	"inventory-manage/internal/model"
	"inventory-manage/internal/service/impl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDeviceRepo struct {
	saveErr   error
	findErr   error
	findAllErr error
	updateErr error
	deleteErr error
	device    *model.Device
	devices   []*model.Device
}

func (m *mockDeviceRepo) Save(ctx context.Context, d *model.Device) error {
	return m.saveErr
}

func (m *mockDeviceRepo) FindByID(ctx context.Context, id string) (*model.Device, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.device, nil
}

func (m *mockDeviceRepo) FindAll(ctx context.Context, q model.DeviceQuery) ([]*model.Device, error) {
	return m.devices, m.findAllErr
}

func (m *mockDeviceRepo) Update(ctx context.Context, d *model.Device) error {
	return m.updateErr
}

func (m *mockDeviceRepo) Delete(ctx context.Context, id string) error {
	return m.deleteErr
}

func TestDeviceService_RegisterDevice(t *testing.T) {
	tests := []struct {
		name    string
		input   *model.Device
		repoErr error
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid device",
			input: &model.Device{
				DeviceID: "D-01",
				SKUCode:  "SKU-01",
				Status:   model.StatusActive,
			},
			repoErr: nil,
			wantErr: false,
		},
		{
			name: "Missing DeviceID",
			input: &model.Device{
				SKUCode: "SKU-01",
				Status:  model.StatusActive,
			},
			wantErr: true,
			errMsg:  "device_id is required",
		},
		{
			name: "Missing SKUCode",
			input: &model.Device{
				DeviceID: "D-01",
				Status:   model.StatusActive,
			},
			wantErr: true,
			errMsg:  "sku_code is required",
		},
		{
			name: "Invalid status",
			input: &model.Device{
				DeviceID: "D-01",
				SKUCode:  "SKU-01",
				Status:   "unknown_status",
			},
			wantErr: true,
			errMsg:  model.ErrInvalidStatus.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockDeviceRepo{saveErr: tt.repoErr}
			svc := impl.NewDeviceService(repo)

			err := svc.RegisterDevice(context.Background(), tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDeviceService_GetDevice(t *testing.T) {
	t.Run("Valid retrieval", func(t *testing.T) {
		dev := &model.Device{DeviceID: "D-01"}
		repo := &mockDeviceRepo{device: dev}
		svc := impl.NewDeviceService(repo)

		res, err := svc.GetDevice(context.Background(), "D-01")
		require.NoError(t, err)
		assert.Equal(t, dev, res)
	})

	t.Run("Missing ID", func(t *testing.T) {
		repo := &mockDeviceRepo{}
		svc := impl.NewDeviceService(repo)

		_, err := svc.GetDevice(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "device_id is required")
	})
}

func TestDeviceService_ListDevices(t *testing.T) {
	repo := &mockDeviceRepo{devices: []*model.Device{{DeviceID: "D-01"}}}
	svc := impl.NewDeviceService(repo)

	res, err := svc.ListDevices(context.Background(), model.DeviceQuery{})
	require.NoError(t, err)
	assert.Len(t, res, 1)
}

func TestDeviceService_UpdateDevice(t *testing.T) {
	t.Run("Valid update", func(t *testing.T) {
		repo := &mockDeviceRepo{device: &model.Device{DeviceID: "D-01"}}
		svc := impl.NewDeviceService(repo)

		err := svc.UpdateDevice(context.Background(), &model.Device{DeviceID: "D-01", SKUCode: "SKU-02", Status: model.StatusActive})
		require.NoError(t, err)
	})

	t.Run("Missing ID", func(t *testing.T) {
		repo := &mockDeviceRepo{}
		svc := impl.NewDeviceService(repo)

		err := svc.UpdateDevice(context.Background(), &model.Device{Status: model.StatusActive})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "device_id is required")
	})

	t.Run("Invalid status", func(t *testing.T) {
		repo := &mockDeviceRepo{}
		svc := impl.NewDeviceService(repo)

		err := svc.UpdateDevice(context.Background(), &model.Device{DeviceID: "D-01", Status: "unknown"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), model.ErrInvalidStatus.Error())
	})

	t.Run("Not found", func(t *testing.T) {
		repo := &mockDeviceRepo{findErr: fmt.Errorf("not found")}
		svc := impl.NewDeviceService(repo)

		err := svc.UpdateDevice(context.Background(), &model.Device{DeviceID: "D-01", Status: model.StatusActive})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "UpdateDevice: not found")
	})
}

func TestDeviceService_RemoveDevice(t *testing.T) {
	t.Run("Valid remove", func(t *testing.T) {
		repo := &mockDeviceRepo{}
		svc := impl.NewDeviceService(repo)

		err := svc.RemoveDevice(context.Background(), "D-01")
		require.NoError(t, err)
	})

	t.Run("Missing ID", func(t *testing.T) {
		repo := &mockDeviceRepo{}
		svc := impl.NewDeviceService(repo)

		err := svc.RemoveDevice(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "device_id is required")
	})
}
