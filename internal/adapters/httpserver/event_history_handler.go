package httpserver

import (
	"net/http"
	"strconv"

	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
)

func eventHistoryHandler(events applicationevent.Service, policy domainauthorization.Policy) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		limit, _ := strconv.Atoi(request.URL.Query().Get("limit"))
		user, _ := userFromContext(request.Context())
		institutionIDs, err := policy.VisibleInstitutionIDs(request.Context(), user)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not resolve event scope")
			return
		}
		var result []domainevent.Event
		if user.SuperAdmin {
			result, err = events.List(request.Context(), limit)
		} else {
			result, err = events.ListByInstitutionIDs(request.Context(), limit, institutionIDs)
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not list events")
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})
}
