package telemetry

import "context"

// Repository defines the contract for telemetry data persistence.
// This interface is defined here in the domain layer (consumer side).
// The implementation lives in internal/repository/postgres/.
type Repository interface {
	Save(ctx context.Context, t *RawTelemetry) error
	SaveBatch(ctx context.Context, records []*RawTelemetry) error
	FindByDeviceID(ctx context.Context, q TelemetryQuery) ([]*RawTelemetry, error)
	IsDuplicate(ctx context.Context, deviceID string, fCnt uint32) (bool, error)
}
