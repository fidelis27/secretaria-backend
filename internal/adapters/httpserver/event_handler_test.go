package httpserver

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
)

func TestEventHandlerDeliversPublishedEvent(t *testing.T) {
	bus := applicationevent.NewBus()
	server := httptest.NewServer(eventHandler(bus))
	defer server.Close()

	connection, _, err := websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):], nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer connection.Close()

	want := domainevent.Event{EventID: "event-1", Type: "STUDENT_CREATED", Version: 1, OccurredAt: time.Now().UTC()}
	bus.Publish(want)

	var got domainevent.Event
	if err := connection.ReadJSON(&got); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if got.EventID != want.EventID || got.Type != want.Type || got.Version != want.Version {
		t.Fatalf("event = %+v, want %+v", got, want)
	}
}
