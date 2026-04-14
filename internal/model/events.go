package model

import "context"

// IEventBus defines a simple Event Bus for publishing and subscribing to events internally.
type IEventBus interface {
	Publish(topic string, event interface{}) error
	Subscribe(ctx context.Context, topic string) (<-chan interface{}, error)
}
