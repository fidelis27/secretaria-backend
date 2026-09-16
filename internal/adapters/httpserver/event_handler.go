package httpserver

import (
	"net/http"

	"github.com/gorilla/websocket"

	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
)

var eventUpgrader = websocket.Upgrader{
	CheckOrigin: func(request *http.Request) bool {
		origin := request.Header.Get("Origin")
		return origin == "" || isLocalFrontendOrigin(origin)
	},
}

func eventHandler(bus *applicationevent.Bus, policy domainauthorization.Policy) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, ok := userFromContext(request.Context())
		if !ok {
			writeError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}
		visibleIDs, err := policy.VisibleInstitutionIDs(request.Context(), user)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not resolve event scope")
			return
		}
		subscriber, cancel := bus.Subscribe()
		defer cancel()
		connection, err := eventUpgrader.Upgrade(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.Close()

		for {
			select {
			case event, open := <-subscriber:
				if !open {
					return
				}
				if !user.SuperAdmin && !eventVisibleToInstitutions(event, visibleIDs) {
					continue
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
