package event

import (
	"testing"
	"time"

	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
)

func TestBusPublishesAndCancelsSubscription(t *testing.T) {
	bus := NewBus()
	subscriber, cancel := bus.Subscribe()
	want := domainevent.Event{EventID: "event-1", Type: "STUDENT_CREATED", OccurredAt: time.Now()}
	bus.Publish(want)

	select {
	case got := <-subscriber:
		if got.EventID != want.EventID || got.Type != want.Type {
			t.Fatalf("event = %+v, want %+v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	cancel()
	select {
	case _, open := <-subscriber:
		if open {
			t.Fatal("subscriber channel remains open")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for subscription cancellation")
	}
}
