package usecase_test

import (
	"context"
	"testing"

	"inventory-manage/internal/domain/device"
	"inventory-manage/internal/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCalibRepo struct {
	SaveFunc      func(ctx context.Context, config *device.CalibrationConfig) error
	GetActiveFunc func(ctx context.Context, deviceID string) (*device.CalibrationConfig, error)
}

func (m *mockCalibRepo) Save(ctx context.Context, config *device.CalibrationConfig) error {
	return m.SaveFunc(ctx, config)
}

func (m *mockCalibRepo) GetActive(ctx context.Context, deviceID string) (*device.CalibrationConfig, error) {
	return m.GetActiveFunc(ctx, deviceID)
}

func TestCalibrationUseCase_RegisterCalibration(t *testing.T) {
	repo := &mockCalibRepo{}
	uc := usecase.NewCalibrationUseCase(repo)
	ctx := context.Background()

	tests := []struct {
		name    string
		input   *device.CalibrationConfig
		setup   func()
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid",
			input: &device.CalibrationConfig{
				DeviceID:    "D1",
				Unit:        "kg",
				CapacityMax: 100,
			},
			setup: func() {
				repo.SaveFunc = func(ctx context.Context, config *device.CalibrationConfig) error { return nil }
			},
		},
		{
			name: "Missing DeviceID",
			input: &device.CalibrationConfig{
				Unit:        "kg",
				CapacityMax: 100,
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "device_id is required",
		},
		{
			name: "Missing Unit",
			input: &device.CalibrationConfig{
				DeviceID:    "D1",
				CapacityMax: 100,
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "unit is required",
		},
		{
			name: "Invalid Capacity",
			input: &device.CalibrationConfig{
				DeviceID:    "D1",
				Unit:        "kg",
				CapacityMax: 0,
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "capacity_max must be greater than zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := uc.RegisterCalibration(ctx, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCalibrationUseCase_GetActiveCalibration(t *testing.T) {
	repo := &mockCalibRepo{}
	uc := usecase.NewCalibrationUseCase(repo)
	ctx := context.Background()

	t.Run("Valid", func(t *testing.T) {
		repo.GetActiveFunc = func(ctx context.Context, deviceID string) (*device.CalibrationConfig, error) {
			return &device.CalibrationConfig{DeviceID: "D1"}, nil
		}
		res, err := uc.GetActiveCalibration(ctx, "D1")
		require.NoError(t, err)
		assert.Equal(t, "D1", res.DeviceID)
	})

	t.Run("Missing DeviceID", func(t *testing.T) {
		_, err := uc.GetActiveCalibration(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "device_id is required")
	})
}
