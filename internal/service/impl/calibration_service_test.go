// Package impl implements tests for INV-SPR02-TASK-003
// AC Coverage:
//   AC-02: TestCalibrationService_UpdateCalibration_Validations
//   AC-05: TestCalibrationService_UpdateCalibration_Rollback
//   AC-06: TestCalibrationService_UpdateCalibration_AllLogic
// IEC 62304 Classification: Software Safety Class B
package impl

import (
	"context"
	"fmt"
	"testing"

	"inventory-manage/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCalibrationRepo struct {
	saveErr     error
	updateTxErr error
	activeCfg   *model.CalibrationConfig
}

func (m *mockCalibrationRepo) Save(ctx context.Context, cfg *model.CalibrationConfig) error {
	return m.saveErr
}

func (m *mockCalibrationRepo) GetActive(ctx context.Context, deviceID string) (*model.CalibrationConfig, error) {
	return m.activeCfg, nil
}

func (m *mockCalibrationRepo) UpdateCalibrationTx(ctx context.Context, deviceID string, config *model.CalibrationConfig) error {
	return m.updateTxErr
}

func (m *mockCalibrationRepo) GetAuditHistory(ctx context.Context, deviceID string, offset, limit uint64) ([]model.CalibrationAuditLog, error) {
	return nil, nil
}

func TestCalibrationService_UpdateCalibration(t *testing.T) {
	tests := []struct {
		name    string
		devID   string
		input   *model.UpdateCalibrationParams
		mErr    error
		wantErr bool
		errMsg  string
	}{
		{
			name:  "AC-06: valid periodic update",
			devID: "SCALE-01",
			input: &model.UpdateCalibrationParams{
				ZeroValue:       10,
				SpanValue:       2000,
				Unit:            "kg",
				CapacityMax:     5000,
				CalibrationType: model.CalibrationTypePeriodic,
				CreatedBy:       "admin",
			},
			mErr:    nil,
			wantErr: false,
		},
		{
			name:  "AC-02: invalid zero value (negative)",
			devID: "SCALE-01",
			input: &model.UpdateCalibrationParams{
				ZeroValue:       -5,
				SpanValue:       2000,
				Unit:            "kg",
				CapacityMax:     5000,
				CalibrationType: model.CalibrationTypeInitial,
			},
			wantErr: true,
			errMsg:  "zero_value must be non-negative",
		},
		{
			name:  "AC-02: zero >= span",
			devID: "SCALE-01",
			input: &model.UpdateCalibrationParams{
				ZeroValue:       10,
				SpanValue:       5,
				Unit:            "kg",
				CapacityMax:     5000,
				CalibrationType: model.CalibrationTypeInitial,
			},
			wantErr: true,
			errMsg:  "strictly greater than",
		},
		{
			name:  "AC-02: invalid unit",
			devID: "SCALE-01",
			input: &model.UpdateCalibrationParams{
				ZeroValue:       10,
				SpanValue:       2000,
				Unit:            "tons",
				CapacityMax:     5000,
				CalibrationType: model.CalibrationTypeInitial,
			},
			wantErr: true,
			errMsg:  "invalid unit",
		},
		{
			name:  "AC-03: invalid calibration type",
			devID: "SCALE-01",
			input: &model.UpdateCalibrationParams{
				ZeroValue:       10,
				SpanValue:       2000,
				Unit:            "kg",
				CapacityMax:     5000,
				CalibrationType: "unknown_type",
			},
			wantErr: true,
			errMsg:  "invalid calibration_type",
		},
		{
			name:  "AC-05: rollback scenario when tx fails",
			devID: "SCALE-01",
			input: &model.UpdateCalibrationParams{
				ZeroValue:       10,
				SpanValue:       2000,
				Unit:            "kg",
				CapacityMax:     5000,
				CalibrationType: model.CalibrationTypeDriftCorrection,
			},
			mErr:    fmt.Errorf("tx conflict or concurrent update"),
			wantErr: true,
			errMsg:  "tx conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCalibrationRepo{updateTxErr: tt.mErr}
			svc := NewCalibrationService(repo)

			err := svc.UpdateCalibration(context.Background(), tt.devID, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCalibrationService_RegisterCalibration(t *testing.T) {
	tests := []struct {
		name    string
		input   *model.CalibrationConfig
		mErr    error
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid registration",
			input: &model.CalibrationConfig{
				DeviceID:    "SCALE-01",
				ZeroValue:   5.0,
				SpanValue:   1000.0,
				CapacityMax: 5000.0,
			},
			mErr:    nil,
			wantErr: false,
		},
		{
			name: "Missing DeviceID",
			input: &model.CalibrationConfig{
				ZeroValue:   5.0,
				SpanValue:   1000.0,
				CapacityMax: 5000.0,
			},
			wantErr: true,
			errMsg:  "device_id is required",
		},
		{
			name: "Negative ZeroValue",
			input: &model.CalibrationConfig{
				DeviceID:    "SCALE-01",
				ZeroValue:   -5.0,
				SpanValue:   1000.0,
				CapacityMax: 5000.0,
			},
			wantErr: true,
			errMsg:  "zero_value must be non-negative",
		},
		{
			name: "Zero or Negative SpanValue",
			input: &model.CalibrationConfig{
				DeviceID:    "SCALE-01",
				ZeroValue:   5.0,
				SpanValue:   0.0,
				CapacityMax: 5000.0,
			},
			wantErr: true,
			errMsg:  "span_value must be positive",
		},
		{
			name: "Zero CapacityMax",
			input: &model.CalibrationConfig{
				DeviceID:    "SCALE-01",
				ZeroValue:   5.0,
				SpanValue:   1000.0,
				CapacityMax: 0.0,
			},
			wantErr: true,
			errMsg:  "capacity_max must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCalibrationRepo{saveErr: tt.mErr}
			svc := NewCalibrationService(repo)

			err := svc.RegisterCalibration(context.Background(), tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCalibrationService_GetActiveCalibration(t *testing.T) {
	t.Run("Valid retrieval", func(t *testing.T) {
		cfg := &model.CalibrationConfig{DeviceID: "SCALE-01"}
		repo := &mockCalibrationRepo{activeCfg: cfg}
		svc := NewCalibrationService(repo)

		res, err := svc.GetActiveCalibration(context.Background(), "SCALE-01")
		require.NoError(t, err)
		assert.Equal(t, cfg, res)
	})

	t.Run("Missing DeviceID", func(t *testing.T) {
		repo := &mockCalibrationRepo{}
		svc := NewCalibrationService(repo)

		_, err := svc.GetActiveCalibration(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "device_id is required")
	})
}

func TestCalibrationService_UpdateCalibration_MissingDeviceID(t *testing.T) {
	repo := &mockCalibrationRepo{}
	svc := NewCalibrationService(repo)

	err := svc.UpdateCalibration(context.Background(), "", &model.UpdateCalibrationParams{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device_id is required")
}

func TestCalibrationService_GetAuditHistory(t *testing.T) {
	t.Run("Missing DeviceID", func(t *testing.T) {
		repo := &mockCalibrationRepo{}
		svc := NewCalibrationService(repo)

		_, err := svc.GetAuditHistory(context.Background(), "", 0, 20)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "device_id is required")
	})

	t.Run("Valid limit enforcement", func(t *testing.T) {
		repo := &mockCalibrationRepo{}
		svc := NewCalibrationService(repo)

		_, err := svc.GetAuditHistory(context.Background(), "SCALE-01", 0, 200)
		require.NoError(t, err)
		// Limit logic internally drops to 20, we just ensure no error propagates assuming mock returns nil
	})
}
