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
