package httpserver

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
)

var eventUpgrader = websocket.Upgrader{
	CheckOrigin: func(request *http.Request) bool {
		origin := request.Header.Get("Origin")
		return isLocalFrontendOrigin(origin)
	},
}

const (
	websocketWriteWait = 10 * time.Second
	websocketPongWait  = 60 * time.Second
	websocketPingEvery = (websocketPongWait * 9) / 10
	websocketReadLimit = 1024
)

type eventConnections struct {
	mutex       sync.Mutex
	connections map[*websocket.Conn]struct{}
	closed      bool
}

func newEventConnections() *eventConnections {
	return &eventConnections{connections: make(map[*websocket.Conn]struct{})}
}

func (connections *eventConnections) register(connection *websocket.Conn) bool {
	connections.mutex.Lock()
	defer connections.mutex.Unlock()
	if connections.closed {
		_ = connection.Close()
		return false
	}
	connections.connections[connection] = struct{}{}
	return true
}

func (connections *eventConnections) unregister(connection *websocket.Conn) {
	connections.mutex.Lock()
	delete(connections.connections, connection)
	connections.mutex.Unlock()
}

func (connections *eventConnections) closeAll() {
	connections.mutex.Lock()
	connections.closed = true
	active := make([]*websocket.Conn, 0, len(connections.connections))
	for connection := range connections.connections {
		active = append(active, connection)
	}
	connections.mutex.Unlock()

	for _, connection := range active {
		_ = connection.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down"), time.Now().Add(websocketWriteWait))
		_ = connection.Close()
	}
}

func eventHandler(bus *applicationevent.Bus, policy domainauthorization.Policy, connections *eventConnections) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, ok := userFromContext(request.Context())
		if !ok {
			writeError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}
		visibleIDs, err := policy.VisibleInstitutionIDs(request.Context(), user)
		if err != nil {
			writeServiceError(writer, err, "could not resolve event scope")
			return
		}
		subscriber, cancel := bus.Subscribe()
		defer cancel()
		responseHeaders := http.Header{}
		if protocol := websocketBearerProtocol(request); protocol != "" {
			responseHeaders.Set("Sec-WebSocket-Protocol", protocol)
		}
		connection, err := eventUpgrader.Upgrade(writer, request, responseHeaders)
		if err != nil {
			return
		}
		if !connections.register(connection) {
			return
		}
		defer connections.unregister(connection)
		defer connection.Close()
		connection.SetReadLimit(websocketReadLimit)
		_ = connection.SetReadDeadline(time.Now().Add(websocketPongWait))
		connection.SetPongHandler(func(string) error {
			return connection.SetReadDeadline(time.Now().Add(websocketPongWait))
		})

		readErrors := make(chan error, 1)
		go func() {
			for {
				if _, _, err := connection.ReadMessage(); err != nil {
					readErrors <- err
					return
				}
			}
		}()
		pingTicker := time.NewTicker(websocketPingEvery)
		defer pingTicker.Stop()

		for {
			select {
			case event, open := <-subscriber:
				if !open {
					return
				}
				if !user.SuperAdmin && !eventVisibleToInstitutions(event, visibleIDs) {
					continue
				}
				_ = connection.SetWriteDeadline(time.Now().Add(websocketWriteWait))
				if err := connection.WriteJSON(event); err != nil {
					return
				}
			case <-pingTicker.C:
				if err := connection.WriteControl(websocket.PingMessage, nil, time.Now().Add(websocketWriteWait)); err != nil {
					return
				}
			case <-readErrors:
				return
			case <-request.Context().Done():
				return
			}
		}
	})
}

func eventVisibleToInstitutions(event domainevent.Event, visibleIDs []string) bool {
	visible := make(map[string]struct{}, len(visibleIDs))
	for _, institutionID := range visibleIDs {
		visible[institutionID] = struct{}{}
	}
	for _, institutionID := range event.InstitutionIDs {
		if _, ok := visible[institutionID]; ok {
			return true
		}
	}
	return false
}
