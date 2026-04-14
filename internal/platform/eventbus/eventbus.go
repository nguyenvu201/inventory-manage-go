package eventbus

import (
	"context"
	"inventory-manage/internal/model"
	"sync"
)

type InMemoryEventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan interface{}
}

func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		subscribers: make(map[string][]chan interface{}),
	}
}

func (b *InMemoryEventBus) Publish(topic string, event interface{}) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if chans, found := b.subscribers[topic]; found {
		for _, ch := range chans {
			// Publish asynchronously to prevent blocking
			go func(c chan interface{}) {
				c <- event
			}(ch)
		}
	}
	return nil
}

func (b *InMemoryEventBus) Subscribe(ctx context.Context, topic string) (<-chan interface{}, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan interface{}, 100)
	b.subscribers[topic] = append(b.subscribers[topic], ch)

	// In a complete implementation, handle ctx.Done() to remove subscriber
	go func() {
		<-ctx.Done()
		// TODO: Implement removal of subscriber from the map
	}()

	return ch, nil
}

// Ensure the implementation satisfies the interface
var _ model.IEventBus = (*InMemoryEventBus)(nil)
