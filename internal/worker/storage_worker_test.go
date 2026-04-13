// Package worker_test implements tests for the storage worker.
// AC Coverage:
//   AC-04: TestStorageWorker_FlushOnBatchSize
//   AC-04: TestStorageWorker_FlushOnTickAndChannelClose
//   AC-04: TestStorageWorker_ContextCancellationFlush
// IEC 62304 Classification: Software Safety Class B
package worker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"inventory-manage/internal/model"
	"inventory-manage/internal/worker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepo simulates the telemetry repository.
type mockRepo struct {
	mu             sync.Mutex
	saveCount      int
	saveBatchCount int
	recordsSaved   int
	errResponse    error
}

func (m *mockRepo) Save(ctx context.Context, record *model.RawTelemetry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveCount++
	return m.errResponse
}

func (m *mockRepo) SaveBatch(ctx context.Context, records []*model.RawTelemetry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveBatchCount++
	m.recordsSaved += len(records)
	return m.errResponse
}

func (m *mockRepo) getStats() (int, int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveCount, m.saveBatchCount, m.recordsSaved
}

func (m *mockRepo) FindByDeviceID(ctx context.Context, q model.TelemetryQuery) ([]*model.RawTelemetry, error) {
	return nil, nil // unused in worker test
}

func (m *mockRepo) IsDuplicate(ctx context.Context, deviceID string, fCnt uint32) (bool, error) {
	return false, nil // unused in worker test
}

func TestStorageWorker_FlushOnBatchSize(t *testing.T) {
	// AC-04: flush on 10 records
	inChan := make(chan model.TelemetryPayload, 20)
	repo := &mockRepo{}

	storageWorker := worker.NewStorageWorker(repo, inChan)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		_ = storageWorker.Start(ctx)
		close(done)
	}()

	// Push 10 payloads (trigger flush by size)
	for i := 0; i < 10; i++ {
		inChan <- model.TelemetryPayload{DeviceID: "SCALE-01"}
	}

	time.Sleep(100 * time.Millisecond)

	_, batchCount, recordsSaved := repo.getStats()
	assert.Equal(t, 1, batchCount, "Expected exactly 1 batch flush after 10 records")
	assert.Equal(t, 10, recordsSaved, "Expected exactly 10 records saved")

	cancel()
	<-done
}

func TestStorageWorker_FlushOnTickAndChannelClose(t *testing.T) {
	inChan := make(chan model.TelemetryPayload, 20)
	repo := &mockRepo{}

	storageWorker := worker.NewStorageWorker(repo, inChan)
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		_ = storageWorker.Start(ctx)
		close(done)
	}()

	// Push 5 records (not enough for 10-batch flush)
	for i := 0; i < 5; i++ {
		inChan <- model.TelemetryPayload{DeviceID: "SCALE-01"}
	}

	time.Sleep(100 * time.Millisecond)
	_, batchCount, _ := repo.getStats()
	assert.Equal(t, 0, batchCount, "Expected 0 flushes before closing")

	// Close channel to trigger final flush
	close(inChan)
	<-done

	_, finalBatch, finalRecords := repo.getStats()
	assert.Equal(t, 1, finalBatch, "Expected 1 flush on channel close")
	assert.Equal(t, 5, finalRecords, "Expected 5 records saved before close")
}

func TestStorageWorker_ContextCancellationFlush(t *testing.T) {
	inChan := make(chan model.TelemetryPayload, 20)
	repo := &mockRepo{}

	storageWorker := worker.NewStorageWorker(repo, inChan)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		err := storageWorker.Start(ctx)
		require.ErrorIs(t, err, context.Canceled)
		close(done)
	}()

	inChan <- model.TelemetryPayload{DeviceID: "SCALE-CTX"}

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	_, batchCount, recordsSaved := repo.getStats()
	assert.Equal(t, 1, batchCount)
	assert.Equal(t, 1, recordsSaved)
}
