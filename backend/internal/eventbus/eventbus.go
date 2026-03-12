package eventbus

import (
	"sync"
	"sync/atomic"
)

// Event represents a domain event published through the bus.
type Event struct {
	Type    string      // e.g. "content.published", "comment.created"
	Payload interface{} // event-specific data
}

// Handler is the function signature for event handlers.
type Handler func(Event)

// Subscriber wraps a handler with sync/async semantics.
type Subscriber struct {
	ID      uint64
	Handler Handler
	Async   bool
}

// SyncHandler creates a synchronous subscriber handler.
func SyncHandler(fn Handler) Subscriber {
	return Subscriber{Handler: fn, Async: false}
}

// AsyncHandler creates an asynchronous subscriber handler.
func AsyncHandler(fn Handler) Subscriber {
	return Subscriber{Handler: fn, Async: true}
}

// EventBus defines the publish/subscribe contract.
type EventBus interface {
	Publish(event Event)
	Subscribe(eventType string, sub Subscriber) uint64
	Unsubscribe(eventType string, id uint64)
}

// Bus is an in-process EventBus implementation.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[string][]Subscriber
	nextID      atomic.Uint64
}

// New creates a new in-process EventBus.
func New() *Bus {
	return &Bus{
		subscribers: make(map[string][]Subscriber),
	}
}

// Publish dispatches an event to all subscribers of that type.
// Sync subscribers are called in order. Async subscribers run in goroutines.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	subs := make([]Subscriber, len(b.subscribers[event.Type]))
	copy(subs, b.subscribers[event.Type])
	b.mu.RUnlock()

	for _, sub := range subs {
		if sub.Async {
			go sub.Handler(event)
		} else {
			sub.Handler(event)
		}
	}
}

// Subscribe registers a handler for an event type. Returns a subscription ID.
func (b *Bus) Subscribe(eventType string, sub Subscriber) uint64 {
	id := b.nextID.Add(1)
	sub.ID = id
	b.mu.Lock()
	b.subscribers[eventType] = append(b.subscribers[eventType], sub)
	b.mu.Unlock()
	return id
}

// Unsubscribe removes a handler by subscription ID.
func (b *Bus) Unsubscribe(eventType string, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subscribers[eventType]
	for i, s := range subs {
		if s.ID == id {
			b.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
			return
		}
	}
}
