package worker_test

import (
	"context"
	"testing"
	"time"

	"inventory-manage/internal/domain/telemetry"
	"inventory-manage/internal/worker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepo simulates db
type mockRepo struct {
	saveCount      int
	saveBatchCount int
	recordsSaved   int
	errResponse    error
}

func (m *mockRepo) Save(ctx context.Context, record *telemetry.RawTelemetry) error {
	m.saveCount++
	return m.errResponse
}

func (m *mockRepo) SaveBatch(ctx context.Context, records []*telemetry.RawTelemetry) error {
	m.saveBatchCount++
	m.recordsSaved += len(records)
	return m.errResponse
}

func (m *mockRepo) FindByDeviceID(ctx context.Context, q telemetry.TelemetryQuery) ([]*telemetry.RawTelemetry, error) {
	return nil, nil // unused in worker test
}

func (m *mockRepo) IsDuplicate(ctx context.Context, deviceID string, fCnt uint32) (bool, error) {
	return false, nil // unused in worker test
}

func TestStorageWorker_FlushOnBatchSize(t *testing.T) {
	// AC-04: flush on 10 records
	inChan := make(chan telemetry.TelemetryPayload, 20)
	repo := &mockRepo{}
	
	storageWorker := worker.NewStorageWorker(repo, inChan)
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start worker asynchronously
	done := make(chan struct{})
	go func() {
		_ = storageWorker.Start(ctx)
		close(done)
	}()

	// Push 10 payloads (trigger flush by size)
	for i := 0; i < 10; i++ {
		inChan <- telemetry.TelemetryPayload{DeviceID: "SCALE-01"}
	}

	// Give worker time to process and flush
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 1, repo.saveBatchCount, "Expected exactly 1 batch flush after 10 records")
	assert.Equal(t, 10, repo.recordsSaved, "Expected exactly 10 records saved")

	// Shutdown
	cancel()
	<-done
}

func TestStorageWorker_FlushOnTickAndChannelClose(t *testing.T) {
	inChan := make(chan telemetry.TelemetryPayload, 20)
	repo := &mockRepo{}
	
	storageWorker := worker.NewStorageWorker(repo, inChan)
	// We won't cancel via ctx here, we will mock closing the channel
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		_ = storageWorker.Start(ctx)
		close(done)
	}()

	// Push 5 records (not enough for 10-batch flush)
	for i := 0; i < 5; i++ {
		inChan <- telemetry.TelemetryPayload{DeviceID: "SCALE-01"}
	}

	// Not closing immediately. Wait briefly shouldn't flush because tick is 2s.
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 0, repo.saveBatchCount, "Expected 0 flushes before closing")

	// Close channel to trigger final flush
	close(inChan)
	
	<-done // Wait for worker to finish safely
	
	assert.Equal(t, 1, repo.saveBatchCount, "Expected 1 flush on channel close")
	assert.Equal(t, 5, repo.recordsSaved, "Expected 5 records saved before close")
}

func TestStorageWorker_ContextCancellationFlush(t *testing.T) {
	inChan := make(chan telemetry.TelemetryPayload, 20)
	repo := &mockRepo{}
	
	storageWorker := worker.NewStorageWorker(repo, inChan)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		err := storageWorker.Start(ctx)
		require.ErrorIs(t, err, context.Canceled)
		close(done)
	}()

	inChan <- telemetry.TelemetryPayload{DeviceID: "SCALE-CTX"}
	
	// Cancel forces exit and flush
	cancel()
	<-done

	assert.Equal(t, 1, repo.saveBatchCount)
	assert.Equal(t, 1, repo.recordsSaved)
}
