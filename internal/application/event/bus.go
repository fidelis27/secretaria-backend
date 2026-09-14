package event

import (
	"sync"

	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
)

type Bus struct {
	mutex       sync.RWMutex
	subscribers map[chan domainevent.Event]struct{}
}

func NewBus() *Bus {
	return &Bus{subscribers: make(map[chan domainevent.Event]struct{})}
}

func (bus *Bus) Publish(event domainevent.Event) {
	bus.mutex.RLock()
	defer bus.mutex.RUnlock()
	for subscriber := range bus.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (bus *Bus) Subscribe() (<-chan domainevent.Event, func()) {
	subscriber := make(chan domainevent.Event, 16)
	bus.mutex.Lock()
	bus.subscribers[subscriber] = struct{}{}
	bus.mutex.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			bus.mutex.Lock()
			delete(bus.subscribers, subscriber)
			close(subscriber)
			bus.mutex.Unlock()
		})
	}
	return subscriber, cancel
}
