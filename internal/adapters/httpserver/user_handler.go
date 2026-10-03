package httpserver

import (
	"errors"
	"net/http"
	"strings"

	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
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
			writeServiceError(writer, err, "could not list users")
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
		if err := decodeJSON(writer, request, &input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		created, err := users.Create(request.Context(), input.Name, input.Email, input.SuperAdmin)
		if errors.Is(err, domainuser.ErrNameRequired) || errors.Is(err, domainuser.ErrEmailRequired) || errors.Is(err, domainuser.ErrEmailInvalid) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeServiceError(writer, err, "could not create user")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
}

func userUpdateHandler(users applicationuser.Service, policy domainauthorization.Policy, events applicationevent.Service) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !authorizeUserManagement(writer, request, policy) {
			return
		}

		var input struct {
			Name   *string `json:"name"`
			Status *string `json:"status"`
		}
		if err := decodeJSON(writer, request, &input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}

		currentUser, _ := userFromContext(request.Context())
		targetUser, found, err := users.FindByID(request.Context(), request.PathValue("userId"))
		if err != nil {
			writeServiceError(writer, err, "could not load user")
			return
		}
		if !found {
			writeError(writer, http.StatusNotFound, "user not found")
			return
		}
		normalizedStatus := ""
		if input.Status != nil {
			normalizedStatus = strings.TrimSpace(*input.Status)
		}
		if normalizedStatus == "inactive" && targetUser.ID == currentUser.ID {
			writeError(writer, http.StatusForbidden, "voce nao pode desativar o proprio usuario")
			return
		}
		if normalizedStatus == "inactive" && targetUser.SuperAdmin && targetUser.Status == "active" {
			activeSuperAdmins, err := users.CountActiveSuperAdmins(request.Context())
			if err != nil {
				writeServiceError(writer, err, "could not validate superadmin status")
				return
			}
			if activeSuperAdmins <= 1 {
				writeError(writer, http.StatusConflict, "deve permanecer ao menos um superadmin ativo")
				return
			}
		}

		updated, err := users.Update(request.Context(), targetUser.ID, input.Name, input.Status)
		if errors.Is(err, domainuser.ErrNameRequired) || errors.Is(err, domainuser.ErrStatusInvalid) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeServiceError(writer, err, "could not update user")
			return
		}
		publishDomainEventBestEffort(request.Context(), events, "USER_UPDATED", "backend.user", request.Header.Get("x-correlation-id"), nil, mutationAuditPayload(
			currentUser.ID,
			targetUser.ID,
			map[string]any{"name": targetUser.Name, "status": targetUser.Status, "superAdmin": targetUser.SuperAdmin},
			map[string]any{"name": updated.Name, "status": updated.Status, "superAdmin": updated.SuperAdmin},
		))
		writeJSON(writer, http.StatusOK, updated)
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
