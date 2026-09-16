package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

func TestEventHandlerDeliversPublishedEvent(t *testing.T) {
	bus := applicationevent.NewBus()
	policy := domainauthorization.NewPolicy(userHandlerMemberships{})
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, domainuser.User{
			ID: "super", Status: "active", SuperAdmin: true,
		}))
		eventHandler(bus, policy).ServeHTTP(writer, request)
	})
	server := httptest.NewServer(handler)
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

func TestEventVisibleToInstitutionsFiltersOtherScopes(t *testing.T) {
	visible := domainevent.Event{InstitutionIDs: []string{"institution-a", "institution-b"}}
	other := domainevent.Event{InstitutionIDs: []string{"institution-c"}}

	if !eventVisibleToInstitutions(visible, []string{"institution-b"}) {
		t.Fatal("event should be visible to a matching institution")
	}
	if eventVisibleToInstitutions(other, []string{"institution-b"}) {
		t.Fatal("event should not be visible outside the institution scope")
	}
}
