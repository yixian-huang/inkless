package eventbus_test

import (
	"sync"
	"testing"
	"time"

	"blotting-consultancy/internal/eventbus"
)

func TestPublishSync(t *testing.T) {
	bus := eventbus.New()
	received := false
	bus.Subscribe("test.event", eventbus.SyncHandler(func(e eventbus.Event) {
		received = true
		if e.Type != "test.event" {
			t.Errorf("expected event type test.event, got %s", e.Type)
		}
	}))

	bus.Publish(eventbus.Event{Type: "test.event", Payload: "hello"})

	if !received {
		t.Error("sync subscriber should have been called")
	}
}

func TestPublishAsync(t *testing.T) {
	bus := eventbus.New()
	var wg sync.WaitGroup
	wg.Add(1)
	bus.Subscribe("test.async", eventbus.AsyncHandler(func(e eventbus.Event) {
		defer wg.Done()
	}))

	bus.Publish(eventbus.Event{Type: "test.async"})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Error("async subscriber timed out")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := eventbus.New()
	count := 0
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		bus.Subscribe("multi", eventbus.SyncHandler(func(e eventbus.Event) {
			mu.Lock()
			count++
			mu.Unlock()
		}))
	}

	bus.Publish(eventbus.Event{Type: "multi"})

	mu.Lock()
	if count != 3 {
		t.Errorf("expected 3 calls, got %d", count)
	}
	mu.Unlock()
}

func TestUnsubscribe(t *testing.T) {
	bus := eventbus.New()
	called := false
	id := bus.Subscribe("unsub", eventbus.SyncHandler(func(e eventbus.Event) {
		called = true
	}))

	bus.Unsubscribe("unsub", id)
	bus.Publish(eventbus.Event{Type: "unsub"})

	if called {
		t.Error("unsubscribed handler should not be called")
	}
}

func TestNoSubscribers(t *testing.T) {
	bus := eventbus.New()
	// Should not panic
	bus.Publish(eventbus.Event{Type: "no.subscribers"})
}

func TestPayloadPassthrough(t *testing.T) {
	bus := eventbus.New()
	var got interface{}
	bus.Subscribe("payload", eventbus.SyncHandler(func(e eventbus.Event) {
		got = e.Payload
	}))

	bus.Publish(eventbus.Event{Type: "payload", Payload: 42})

	if got != 42 {
		t.Errorf("expected payload 42, got %v", got)
	}
}
