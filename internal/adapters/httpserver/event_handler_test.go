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
	t.Setenv("APP_ENV", "development")
	bus := applicationevent.NewBus()
	policy := domainauthorization.NewPolicy(userHandlerMemberships{})
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, domainuser.User{
			ID: "super", Status: "active", SuperAdmin: true,
		}))
		eventHandler(bus, policy, newEventConnections()).ServeHTTP(writer, request)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	dialer := websocket.DefaultDialer
	connection, _, err := dialer.Dial("ws"+server.URL[len("http"):], websocketHeader("http://localhost:4173"))
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

func TestEventHandlerNegotiatesBearerSubprotocol(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	bus := applicationevent.NewBus()
	policy := domainauthorization.NewPolicy(userHandlerMemberships{})
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, domainuser.User{
			ID: "super", Status: "active", SuperAdmin: true,
		}))
		eventHandler(bus, policy, newEventConnections()).ServeHTTP(writer, request)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	const bearerProtocol = "bearer.test-access-token"
	dialer := websocket.Dialer{Subprotocols: []string{bearerProtocol}}
	connection, _, err := dialer.Dial("ws"+server.URL[len("http"):], websocketHeader("http://localhost:4173"))
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer connection.Close()

	if got := connection.Subprotocol(); got != bearerProtocol {
		t.Fatalf("subprotocol = %q, want %q", got, bearerProtocol)
	}
}

func TestEventHandlerRejectsDisallowedOrigin(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("CORS_ORIGINS", "https://app.example.com")
	bus := applicationevent.NewBus()
	policy := domainauthorization.NewPolicy(userHandlerMemberships{})
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, domainuser.User{
			ID: "super", Status: "active", SuperAdmin: true,
		}))
		eventHandler(bus, policy, newEventConnections()).ServeHTTP(writer, request)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	_, response, err := websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):], websocketHeader("https://untrusted.example"))
	if err == nil {
		t.Fatal("expected websocket handshake to reject the untrusted origin")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("response = %v, want HTTP 403", response)
	}
}

func TestEventConnectionsCloseRegisteredConnectionsOnShutdown(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	bus := applicationevent.NewBus()
	policy := domainauthorization.NewPolicy(userHandlerMemberships{})
	connections := newEventConnections()
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, domainuser.User{
			ID: "super", Status: "active", SuperAdmin: true,
		}))
		eventHandler(bus, policy, connections).ServeHTTP(writer, request)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	connection, _, err := websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):], websocketHeader("http://localhost:4173"))
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for {
		connections.mutex.Lock()
		activeConnections := len(connections.connections)
		connections.mutex.Unlock()
		if activeConnections == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("websocket connection was not registered")
		}
		time.Sleep(time.Millisecond)
	}

	connections.closeAll()
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := connection.ReadMessage(); err == nil {
		t.Fatal("websocket remained open after shutdown")
	}
	_ = connection.Close()
}

func websocketHeader(origin string) http.Header {
	return http.Header{"Origin": []string{origin}}
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
