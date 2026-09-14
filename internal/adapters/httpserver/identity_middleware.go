package httpserver

import (
	"context"
	"net/http"
	"strings"

	applicationuser "github.com/fidelis27/secretaria-backend/internal/application/user"
	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type identityContextKey struct{}

func identityMiddleware(next http.Handler, users applicationuser.Service) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/health" || (request.URL.Path == "/users" && request.Method == http.MethodPost) {
			next.ServeHTTP(writer, request)
			return
		}

		userID := strings.TrimSpace(request.Header.Get("x-demo-user"))
		if userID == "" {
			writeError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}
		user, found, err := users.FindByID(request.Context(), userID)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not load user")
			return
		}
		if !found || user.Status != "active" {
			writeError(writer, http.StatusForbidden, "forbidden")
			return
		}

		request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, user))
		next.ServeHTTP(writer, request)
	})
}

func userFromContext(ctx context.Context) (domainuser.User, bool) {
	user, ok := ctx.Value(identityContextKey{}).(domainuser.User)
	return user, ok
}
