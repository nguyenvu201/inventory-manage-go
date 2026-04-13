package impl_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"inventory-manage/global"
	"inventory-manage/internal/model"
	"inventory-manage/internal/service/impl"
	pkglogger "inventory-manage/pkg/logger"
)

// Package impl_test implements tests for INV-SPR02-TASK-004 Drift Detection
// AC Coverage:
//   AC-05: TestDriftDetection_WarningLogs
// IEC 62304 Classification: Software Safety Class B

// mockDriftRepo just simulates returning an existing active config.
type mockDriftRepo struct {
	activeConfig *model.CalibrationConfig
}

func (m *mockDriftRepo) Save(ctx context.Context, cfg *model.CalibrationConfig) error {
	return nil
}

func (m *mockDriftRepo) GetActive(ctx context.Context, deviceID string) (*model.CalibrationConfig, error) {
	if m.activeConfig != nil {
		return m.activeConfig, nil
	}
	return nil, model.ErrDeviceNotFound
}

func (m *mockDriftRepo) UpdateCalibrationTx(ctx context.Context, deviceID string, config *model.CalibrationConfig) error {
	return nil
}

func (m *mockDriftRepo) GetAuditHistory(ctx context.Context, deviceID string, offset, limit uint64) ([]model.CalibrationAuditLog, error) {
	return nil, nil
}

func TestDriftDetection_WarningLogs(t *testing.T) {
	// Pre-requisites: Setup mock Zap Logger
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	
	// Temporarily override global logger
	origLogger := global.Logger
	global.Logger = &pkglogger.LoggerZap{Logger: logger}
	defer func() { global.Logger = origLogger }()

	ctx := context.Background()

	tests := []struct {
		name         string
		oldZero      float64
		newZero      float64
		expectWarning bool
	}{
		{
			name:         "AC-05: drift < 5% does not create warning (2.0%)",
			oldZero:      1000.0,
			newZero:      1020.0,
			expectWarning: false,
		},
		{
			name:         "AC-05: drift > 5% creates warning (10.0%)",
			oldZero:      1000.0,
			newZero:      1100.0,
			expectWarning: true,
		},
		{
			name:         "AC-05: drift > 5% drops below zero creates warning (-10.0%)",
			oldZero:      1000.0,
			newZero:      900.0,
			expectWarning: true,
		},
		{
			name:         "AC-05: zero_value initially 0, now heavily changed",
			oldZero:      0.0,
			newZero:      500.0,
			expectWarning: true,
		},
		{
			name:         "AC-05: exactly 5% edge case",
			oldZero:      1000.0,
			newZero:      1050.0,
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs.TakeAll() // Clear logs

			repo := &mockDriftRepo{
				activeConfig: &model.CalibrationConfig{
					ZeroValue: tt.oldZero,
				},
			}
			svc := impl.NewCalibrationService(repo)

			err := svc.UpdateCalibration(ctx, "SCALE-001", &model.UpdateCalibrationParams{
				ZeroValue:       tt.newZero,
				SpanValue:       2000.0,
				Unit:            "kg",
				CapacityMax:     2000.0,
				CalibrationType: model.CalibrationTypePeriodic,
			})
			require.NoError(t, err)

			var foundWarning bool
			for _, log := range logs.All() {
				if log.Message == "calibration drift detected exceeding safe limits" && log.Level == zapcore.WarnLevel {
					foundWarning = true
					break
				}
			}

			assert.Equal(t, tt.expectWarning, foundWarning)
		})
	}
}
