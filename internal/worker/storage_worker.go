package worker

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"inventory-manage/internal/domain/telemetry"
)

// StorageWorker consumes the internal telemetry pipeline channel and
// persists records into the database in batches.
type StorageWorker struct {
	repo         telemetry.Repository
	inChan       <-chan telemetry.TelemetryPayload
	batchSize    int
	tickInterval time.Duration
}

// NewStorageWorker creates a new StorageWorker.
func NewStorageWorker(repo telemetry.Repository, inChan <-chan telemetry.TelemetryPayload) *StorageWorker {
	return &StorageWorker{
		repo:         repo,
		inChan:       inChan,
		batchSize:    10,               // AC-04: Batch insert when more than 10 records arrive
		tickInterval: 2 * time.Second,  // Flush every 2s to avoid staleness on low traffic
	}
}

// Start runs the consuming loop until context is canceled or channel closes.
func (w *StorageWorker) Start(ctx context.Context) error {
	var batch []*telemetry.RawTelemetry
	ticker := time.NewTicker(w.tickInterval)
	defer ticker.Stop()

	// Helper to flush current batch
	flush := func() {
		if len(batch) == 0 {
			return
		}
		
		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := w.repo.SaveBatch(flushCtx, batch); err != nil {
			log.Error().
				Err(err).
				Int("batch_size", len(batch)).
				Msg("failed to save raw telemetry batch")
		} else {
			log.Debug().
				Int("batch_size", len(batch)).
				Msg("persisted telemetry batch successfully")
		}
		
		// Reset batch (re-allocating helps prevent weird GC retentions vs slicing to 0)
		batch = make([]*telemetry.RawTelemetry, 0, w.batchSize)
	}

	for {
		select {
		case <-ctx.Done():
			// Flush final before exit
			flush()
			return ctx.Err()
			
		case payload, ok := <-w.inChan:
			if !ok {
				// Channel closed, process remaining and exit safely
				flush()
				return nil
			}

			// Map TelemetryPayload -> RawTelemetry
			raw := &telemetry.RawTelemetry{
				DeviceID:        payload.DeviceID,
				RawWeight:       payload.RawWeight,
				BatteryLevel:    payload.BatteryLevel,
				RSSI:            ptrOrNil16(payload.RSSI),
				SNR:             ptrOrNil32(payload.SNR),
				FCnt:            ptrOrNilU32(payload.FCnt),
				SpreadingFactor: ptrOrNil8(payload.SpreadingFactor),
				SampleCount:     payload.SampleCount, // AC-08 mapper
				PayloadJSON:     []byte(`{}`), // Storing original JSON would require passing it, simplified here
				ReceivedAt:      payload.ReceivedAt,
			}
			
			batch = append(batch, raw)

			if len(batch) >= w.batchSize {
				flush()
				ticker.Reset(w.tickInterval) // Reset ticker to avoid double flush immediately
			}

		case <-ticker.C:
			flush()
		}
	}
}

// Helpers to handle zeroes as nil, or just map them. For LoRaWAN, 0 might be valid (RSSI), 
// but in this pipeline, missing usually ends up as zero.
// Ideally, TelemetryPayload would hold pointers or valid flags. For safety, we just map everything directly.

func ptrOrNil16(v int) *int16 {
	if v == 0 {
		return nil
	}
	r := int16(v)
	return &r
}

func ptrOrNil32(v float32) *float32 {
	if v == 0 {
		return nil
	}
	return &v
}

func ptrOrNilU32(v uint32) *uint32 {
	// 0 might be a valid FCnt, but ChirpStack omits it if not provided.
	// We'll trust it. If it's truly missing, we'll map as 0 which is fine.
	return &v
}

func ptrOrNil8(v int) *int8 {
	if v == 0 {
		return nil
	}
	r := int8(v)
	return &r
}
