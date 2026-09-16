package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	applicationuser "github.com/fidelis27/secretaria-backend/internal/application/user"
	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

func userListHandler(users applicationuser.Service, policy domainauthorization.Policy) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !authorizeUserManagement(writer, request, policy) {
			return
		}
		result, err := users.List(request.Context())
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not list users")
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})
}

func userCreateHandler(users applicationuser.Service, policy domainauthorization.Policy) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !authorizeUserManagement(writer, request, policy) {
			return
		}
		var input struct {
			Name       string `json:"name"`
			Email      string `json:"email"`
			SuperAdmin bool   `json:"superAdmin"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		created, err := users.Create(request.Context(), input.Name, input.Email, input.SuperAdmin)
		if errors.Is(err, domainuser.ErrNameRequired) || errors.Is(err, domainuser.ErrEmailRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not create user")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
}

func authorizeUserManagement(writer http.ResponseWriter, request *http.Request, policy domainauthorization.Policy) bool {
	user, ok := userFromContext(request.Context())
	if !ok {
		writeError(writer, http.StatusUnauthorized, "unauthorized")
		return false
	}
	if !policy.CanManageUsers(user) {
		writeError(writer, http.StatusForbidden, "forbidden")
		return false
	}
	return true
}
