package telemetry

import (
	"context"
	"sync"
)

// Processor handles pre-storage data manipulation, such as the moving average filter.
type Processor struct {
	buffers sync.Map // map[string]*ringBuffer
}

// NewProcessor creates a new Processor.
func NewProcessor() *Processor {
	return &Processor{}
}

// Process applies server-side processing to the telemetry payload.
// For AC-08: Applies a 5-reading moving average if SampleCount <= 1.
func (p *Processor) Process(ctx context.Context, payload TelemetryPayload) (TelemetryPayload, error) {
	if payload.SampleCount > 1 {
		// Data is pre-averaged by the node. Do not perturb it.
		return payload, nil
	}

	// Apply 5-reading moving average server-side
	val, ok := p.buffers.Load(payload.DeviceID)
	var rb *ringBuffer
	if !ok {
		rb = newRingBuffer(5)
		p.buffers.Store(payload.DeviceID, rb)
	} else {
		rb = val.(*ringBuffer)
	}

	avg := rb.AddAndAverage(payload.RawWeight)
	
	// Create a new payload with the averaged weight
	finalPayload := payload
	finalPayload.RawWeight = avg

	return finalPayload, nil
}

// ringBuffer is a thread-safe circular buffer for moving average calculations.
type ringBuffer struct {
	mu       sync.RWMutex
	capacity int
	data     []float64
	index    int
}

func newRingBuffer(capacity int) *ringBuffer {
	return &ringBuffer{
		capacity: capacity,
		data:     make([]float64, 0, capacity),
		index:    0,
	}
}

// AddAndAverage adds a new value and returns the current average.
func (b *ringBuffer) AddAndAverage(val float64) float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.data) < b.capacity {
		b.data = append(b.data, val)
	} else {
		b.data[b.index] = val
		b.index = (b.index + 1) % b.capacity
	}

	var sum float64
	for _, v := range b.data {
		sum += v
	}
	return sum / float64(len(b.data))
}
