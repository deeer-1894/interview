package service

import (
	"sync"

	"mockinterview/internal/protocol"
)

type EventBroker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan protocol.Event]struct{}
}

func NewEventBroker() *EventBroker {
	return &EventBroker{
		subscribers: make(map[string]map[chan protocol.Event]struct{}),
	}
}

func (b *EventBroker) Subscribe(runID string) (<-chan protocol.Event, func()) {
	ch := make(chan protocol.Event, 256)

	b.mu.Lock()
	if _, ok := b.subscribers[runID]; !ok {
		b.subscribers[runID] = make(map[chan protocol.Event]struct{})
	}
	b.subscribers[runID][ch] = struct{}{}
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		subscribers, ok := b.subscribers[runID]
		if !ok {
			return
		}
		if _, exists := subscribers[ch]; exists {
			delete(subscribers, ch)
			close(ch)
		}
		if len(subscribers) == 0 {
			delete(b.subscribers, runID)
		}
	}

	return ch, cancel
}

func (b *EventBroker) Publish(event protocol.Event) {
	b.mu.RLock()
	subscribers := b.subscribers[event.RunID]
	channels := make([]chan protocol.Event, 0, len(subscribers))
	for ch := range subscribers {
		channels = append(channels, ch)
	}
	b.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
		}
	}
}
