package worker

import (
	"context"
	"time"

	"inventory-manage/global"
	"inventory-manage/internal/model"
	"inventory-manage/internal/service"

	"go.uber.org/zap"
)

// log returns the global logger or a no-op logger for tests.
func workerLog() *zap.Logger {
	if global.Logger != nil {
		return global.Logger.Logger
	}
	return zap.NewNop()
}

// StorageWorker consumes the internal telemetry pipeline channel and
// persists records into the database in batches.
type StorageWorker struct {
	repo         service.ITelemetryRepository
	inChan       <-chan model.TelemetryPayload
	batchSize    int
	tickInterval time.Duration
}

// NewStorageWorker creates a new StorageWorker.
func NewStorageWorker(repo service.ITelemetryRepository, inChan <-chan model.TelemetryPayload) *StorageWorker {
	return &StorageWorker{
		repo:         repo,
		inChan:       inChan,
		batchSize:    10,              // Batch insert when more than 10 records arrive
		tickInterval: 2 * time.Second, // Flush every 2s to avoid staleness on low traffic
	}
}

// Start runs the consuming loop until context is canceled or channel closes.
func (w *StorageWorker) Start(ctx context.Context) error {
	var batch []*model.RawTelemetry
	ticker := time.NewTicker(w.tickInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := w.repo.SaveBatch(flushCtx, batch); err != nil {
			workerLog().Error("failed to save raw telemetry batch",
				zap.Error(err),
				zap.Int("batch_size", len(batch)),
			)
		} else {
			workerLog().Debug("persisted telemetry batch successfully",
				zap.Int("batch_size", len(batch)),
			)
		}

		batch = make([]*model.RawTelemetry, 0, w.batchSize)
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return ctx.Err()

		case payload, ok := <-w.inChan:
			if !ok {
				flush()
				return nil
			}

			fcnt := payload.FCnt
			raw := &model.RawTelemetry{
				DeviceID:        payload.DeviceID,
				RawWeight:       payload.RawWeight,
				BatteryLevel:    payload.BatteryLevel,
				RSSI:            payload.RSSI,
				SNR:             payload.SNR,
				FCnt:            &fcnt,
				SpreadingFactor: payload.SpreadingFactor,
				SampleCount:     payload.SampleCount,
				PayloadJSON:     []byte(`{}`),
				ReceivedAt:      payload.ReceivedAt,
			}

			batch = append(batch, raw)
			if len(batch) >= w.batchSize {
				flush()
				ticker.Reset(w.tickInterval)
			}

		case <-ticker.C:
			flush()
		}
	}
}
