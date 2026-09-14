package httpserver

import (
	"net/http"

	"github.com/gorilla/websocket"

	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
)

var eventUpgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

func eventHandler(bus *applicationevent.Bus) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := eventUpgrader.Upgrade(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.Close()

		subscriber, cancel := bus.Subscribe()
		defer cancel()
		for {
			select {
			case event, open := <-subscriber:
				if !open {
					return
				}
				if err := connection.WriteJSON(event); err != nil {
					return
				}
			case <-request.Context().Done():
				return
			}
		}
	})
}
