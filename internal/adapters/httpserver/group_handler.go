package httpserver

import (
	"errors"
	"net/http"

	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	applicationgroup "github.com/fidelis27/secretaria-backend/internal/application/group"
	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domaingroup "github.com/fidelis27/secretaria-backend/internal/domain/group"
)

func groupListHandler(groups applicationgroup.Service, policy domainauthorization.Policy) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, _ := userFromContext(request.Context())
		institutionIDs, err := policy.VisibleInstitutionIDs(request.Context(), user)
		if err != nil {
			writeServiceError(writer, err, "could not resolve group scope")
			return
		}
		var result []domaingroup.Group
		if user.SuperAdmin {
			result, err = groups.List(request.Context())
		} else {
			result, err = groups.ListByInstitutionIDs(request.Context(), institutionIDs)
		}
		if err != nil {
			writeServiceError(writer, err, "could not list groups")
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})
}

func groupCreateHandler(groups applicationgroup.Service, policy domainauthorization.Policy) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, _ := userFromContext(request.Context())
		if !policy.CanCreateGroup(user) {
			writeError(writer, http.StatusForbidden, "forbidden")
			return
		}
		var input struct {
			InstitutionID string `json:"institutionId"`
			Name          string `json:"name"`
		}
		if err := decodeJSON(writer, request, &input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		created, err := groups.Create(request.Context(), input.InstitutionID, input.Name)
		if errors.Is(err, domaingroup.ErrInstitutionIDRequired) || errors.Is(err, domaingroup.ErrNameRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeServiceError(writer, err, "could not create group")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
}

func groupMembershipListHandler(groups applicationgroup.Service, policy domainauthorization.Policy) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		group, ok := authorizeGroupManagement(writer, request, groups, policy)
		if !ok {
			return
		}
		memberships, err := groups.ListMemberships(request.Context(), group.ID)
		if err != nil {
			writeServiceError(writer, err, "could not list memberships")
			return
		}
		writeJSON(writer, http.StatusOK, memberships)
	})
}

func groupMembershipCreateHandler(groups applicationgroup.Service, policy domainauthorization.Policy) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		group, ok := authorizeGroupManagement(writer, request, groups, policy)
		if !ok {
			return
		}
		var input struct {
			UserID string                   `json:"userId"`
			Role   domainauthorization.Role `json:"role"`
		}
		if err := decodeJSON(writer, request, &input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		created, err := groups.AddMembership(request.Context(), input.UserID, group.ID, group.InstitutionID, input.Role)
		if errors.Is(err, domaingroup.ErrUserIDRequired) || errors.Is(err, domaingroup.ErrGroupIDRequired) || errors.Is(err, domaingroup.ErrInvalidRole) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domaingroup.ErrMembershipNotFound) {
			writeError(writer, http.StatusNotFound, err.Error())
			return
		}
		if err != nil {
			writeServiceError(writer, err, "could not create membership")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
}

func groupMembershipUpdateHandler(groups applicationgroup.Service, policy domainauthorization.Policy, events applicationevent.Service) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		group, ok := authorizeGroupManagement(writer, request, groups, policy)
		if !ok {
			return
		}
		var input struct {
			Role domainauthorization.Role `json:"role"`
		}
		if err := decodeJSON(writer, request, &input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		targetID := request.PathValue("userId")
		memberships, err := groups.ListMemberships(request.Context(), group.ID)
		if err != nil {
			writeServiceError(writer, err, "could not load membership before update")
			return
		}
		var beforeRole domainauthorization.Role
		for _, membership := range memberships {
			if membership.UserID == targetID {
				beforeRole = membership.Role
				break
			}
		}
		updated, err := groups.UpdateMembershipRole(request.Context(), group.ID, targetID, input.Role)
		if errors.Is(err, domaingroup.ErrUserIDRequired) || errors.Is(err, domaingroup.ErrGroupIDRequired) || errors.Is(err, domaingroup.ErrInvalidRole) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domaingroup.ErrMembershipNotFound) {
			writeError(writer, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domaingroup.ErrLastAdminRequired) {
			writeError(writer, http.StatusConflict, err.Error())
			return
		}
		if err != nil {
			writeServiceError(writer, err, "could not update membership")
			return
		}
		actor, _ := userFromContext(request.Context())
		publishDomainEventBestEffort(request.Context(), events, "GROUP_MEMBERSHIP_UPDATED", "backend.group", request.Header.Get("x-correlation-id"), []string{group.InstitutionID}, mutationAuditPayload(actor.ID, targetID, map[string]any{"role": beforeRole}, map[string]any{"role": updated.Role}))
		writeJSON(writer, http.StatusOK, updated)
	})
}

func groupMembershipDeleteHandler(groups applicationgroup.Service, policy domainauthorization.Policy, events applicationevent.Service) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		group, ok := authorizeGroupManagement(writer, request, groups, policy)
		if !ok {
			return
		}
		userID := request.PathValue("userId")
		memberships, err := groups.ListMemberships(request.Context(), group.ID)
		if err != nil {
			writeServiceError(writer, err, "could not load membership before removal")
			return
		}
		var beforeRole domainauthorization.Role
		for _, membership := range memberships {
			if membership.UserID == userID {
				beforeRole = membership.Role
				break
			}
		}
		if err := groups.RemoveMembership(request.Context(), group.ID, userID); err != nil {
			if errors.Is(err, domaingroup.ErrUserIDRequired) || errors.Is(err, domaingroup.ErrGroupIDRequired) {
				writeError(writer, http.StatusBadRequest, err.Error())
				return
			}
			if errors.Is(err, domaingroup.ErrMembershipNotFound) {
				writeError(writer, http.StatusNotFound, err.Error())
				return
			}
			if errors.Is(err, domaingroup.ErrLastAdminRequired) {
				writeError(writer, http.StatusConflict, err.Error())
				return
			}
			writeServiceError(writer, err, "could not remove membership")
			return
		}
		actor, _ := userFromContext(request.Context())
		publishDomainEventBestEffort(request.Context(), events, "GROUP_MEMBERSHIP_REMOVED", "backend.group", request.Header.Get("x-correlation-id"), []string{group.InstitutionID}, mutationAuditPayload(actor.ID, userID, map[string]any{"role": beforeRole}, nil))
		writer.WriteHeader(http.StatusNoContent)
	})
}

func groupCandidatesHandler(groups applicationgroup.Service, policy domainauthorization.Policy) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		group, ok := authorizeGroupManagement(writer, request, groups, policy)
		if !ok {
			return
		}
		candidates, err := groups.ListCandidates(request.Context(), group.ID)
		if errors.Is(err, domaingroup.ErrGroupIDRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeServiceError(writer, err, "could not list group candidates")
			return
		}
		writeJSON(writer, http.StatusOK, candidates)
	})
}

func authorizeGroupManagement(writer http.ResponseWriter, request *http.Request, groups applicationgroup.Service, policy domainauthorization.Policy) (domaingroup.Group, bool) {
	group, found, err := groups.FindByID(request.Context(), request.PathValue("groupId"))
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "could not load group")
		return domaingroup.Group{}, false
	}
	if !found {
		writeError(writer, http.StatusNotFound, "group not found")
		return domaingroup.Group{}, false
	}
	user, _ := userFromContext(request.Context())
	allowed, err := policy.CanManageGroup(request.Context(), user, group.ID)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "could not verify group authorization")
		return domaingroup.Group{}, false
	}
	if !allowed {
		writeError(writer, http.StatusForbidden, "forbidden")
		return domaingroup.Group{}, false
	}
	return group, true
}
