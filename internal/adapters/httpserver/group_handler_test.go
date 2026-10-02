package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	applicationgroup "github.com/fidelis27/secretaria-backend/internal/application/group"
	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domaingroup "github.com/fidelis27/secretaria-backend/internal/domain/group"
	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type groupHandlerRepository struct {
	groups       map[string]domaingroup.Group
	memberships  map[string][]domaingroup.Membership
	candidates   map[string][]domaingroup.Candidate
	createdGroup domaingroup.Group
}

func (repository *groupHandlerRepository) Create(_ context.Context, entity domaingroup.Group) error {
	repository.createdGroup = entity
	if repository.groups == nil {
		repository.groups = map[string]domaingroup.Group{}
	}
	entity.InstitutionName = "Instituicao A"
	repository.groups[entity.ID] = entity
	return nil
}

func (repository *groupHandlerRepository) List(context.Context) ([]domaingroup.Group, error) {
	result := make([]domaingroup.Group, 0, len(repository.groups))
	for _, group := range repository.groups {
		result = append(result, group)
	}
	return result, nil
}

func (repository *groupHandlerRepository) ListByInstitutionIDs(context.Context, []string) ([]domaingroup.Group, error) {
	return repository.List(context.Background())
}

func (repository *groupHandlerRepository) FindByID(_ context.Context, id string) (domaingroup.Group, bool, error) {
	group, found := repository.groups[id]
	return group, found, nil
}

func (repository *groupHandlerRepository) AddMembership(_ context.Context, membership domaingroup.Membership) (domaingroup.Membership, error) {
	membership.UserName = "User"
	membership.UserEmail = "user@example.com"
	repository.memberships[membership.GroupID] = append(repository.memberships[membership.GroupID], membership)
	return membership, nil
}

func (repository *groupHandlerRepository) UpdateMembershipRole(_ context.Context, groupID string, userID string, role domainauthorization.Role) (domaingroup.Membership, error) {
	memberships := repository.memberships[groupID]
	for index, membership := range memberships {
		if membership.UserID != userID {
			continue
		}
		if membership.Role == domainauthorization.RoleAdmin && role != domainauthorization.RoleAdmin {
			admins := 0
			for _, existing := range memberships {
				if existing.Role == domainauthorization.RoleAdmin && existing.UserID != userID {
					admins++
				}
			}
			if admins == 0 {
				return domaingroup.Membership{}, domaingroup.ErrLastAdminRequired
			}
		}
		membership.Role = role
		repository.memberships[groupID][index] = membership
		return membership, nil
	}
	return domaingroup.Membership{}, domaingroup.ErrMembershipNotFound
}

func (repository *groupHandlerRepository) RemoveMembership(_ context.Context, groupID string, userID string) error {
	memberships := repository.memberships[groupID]
	for index, membership := range memberships {
		if membership.UserID != userID {
			continue
		}
		if membership.Role == domainauthorization.RoleAdmin {
			admins := 0
			for _, existing := range memberships {
				if existing.Role == domainauthorization.RoleAdmin && existing.UserID != userID {
					admins++
				}
			}
			if admins == 0 {
				return domaingroup.ErrLastAdminRequired
			}
		}
		repository.memberships[groupID] = append(memberships[:index], memberships[index+1:]...)
		return nil
	}
	return domaingroup.ErrMembershipNotFound
}

func (repository *groupHandlerRepository) ListMemberships(_ context.Context, groupID string) ([]domaingroup.Membership, error) {
	return repository.memberships[groupID], nil
}

func (repository *groupHandlerRepository) ListCandidates(_ context.Context, groupID string) ([]domaingroup.Candidate, error) {
	return repository.candidates[groupID], nil
}

func groupRequest(method string, path string, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	return request.WithContext(context.WithValue(request.Context(), identityContextKey{}, domainuser.User{
		ID: "super", Status: "active", SuperAdmin: true,
	}))
}

func TestGroupListHandlerIncludesNameAndInstitutionName(t *testing.T) {
	repository := &groupHandlerRepository{
		groups: map[string]domaingroup.Group{
			"group-1": {ID: "group-1", InstitutionID: "inst-1", Name: "Grupo A", InstitutionName: "Instituicao A"},
		},
		memberships: map[string][]domaingroup.Membership{},
		candidates:  map[string][]domaingroup.Candidate{},
	}
	recorder := httptest.NewRecorder()

	groupListHandler(applicationgroup.NewService(repository), domainauthorization.NewPolicy(userHandlerMemberships{})).
		ServeHTTP(recorder, groupRequest(http.MethodGet, "/groups", ""))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"name":"Grupo A"`) || !strings.Contains(body, `"institutionName":"Instituicao A"`) {
		t.Fatalf("body = %s", body)
	}
}

func TestGroupCreateHandlerRequiresName(t *testing.T) {
	repository := &groupHandlerRepository{
		groups:      map[string]domaingroup.Group{},
		memberships: map[string][]domaingroup.Membership{},
		candidates:  map[string][]domaingroup.Candidate{},
	}
	recorder := httptest.NewRecorder()

	groupCreateHandler(applicationgroup.NewService(repository), domainauthorization.NewPolicy(userHandlerMemberships{})).
		ServeHTTP(recorder, groupRequest(http.MethodPost, "/groups", `{"institutionId":"inst-1","name":"  "}`))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestGroupMembershipListIncludesUserInfo(t *testing.T) {
	repository := &groupHandlerRepository{
		groups: map[string]domaingroup.Group{
			"group-1": {ID: "group-1", InstitutionID: "inst-1", Name: "Grupo A", InstitutionName: "Instituicao A"},
		},
		memberships: map[string][]domaingroup.Membership{
			"group-1": {{UserID: "user-1", GroupID: "group-1", InstitutionID: "inst-1", Role: domainauthorization.RoleAdmin, UserName: "User 1", UserEmail: "user1@example.com"}},
		},
		candidates: map[string][]domaingroup.Candidate{},
	}
	recorder := httptest.NewRecorder()
	request := groupRequest(http.MethodGet, "/groups/group-1/members", "")
	request.SetPathValue("groupId", "group-1")

	groupMembershipListHandler(applicationgroup.NewService(repository), domainauthorization.NewPolicy(userHandlerMemberships{})).
		ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"userName":"User 1"`) || !strings.Contains(body, `"userEmail":"user1@example.com"`) {
		t.Fatalf("body = %s", body)
	}
}

func TestGroupMembershipUpdateRejectsDemotingLastAdmin(t *testing.T) {
	repository := &groupHandlerRepository{
		groups: map[string]domaingroup.Group{
			"group-1": {ID: "group-1", InstitutionID: "inst-1", Name: "Grupo A", InstitutionName: "Instituicao A"},
		},
		memberships: map[string][]domaingroup.Membership{
			"group-1": {{UserID: "user-1", GroupID: "group-1", InstitutionID: "inst-1", Role: domainauthorization.RoleAdmin, UserName: "User 1", UserEmail: "user1@example.com"}},
		},
		candidates: map[string][]domaingroup.Candidate{},
	}
	recorder := httptest.NewRecorder()
	request := groupRequest(http.MethodPatch, "/groups/group-1/members/user-1", `{"role":"member"}`)
	request.SetPathValue("groupId", "group-1")
	request.SetPathValue("userId", "user-1")

	groupMembershipUpdateHandler(applicationgroup.NewService(repository), domainauthorization.NewPolicy(userHandlerMemberships{}), zeroEventService()).
		ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
}

func TestGroupCandidatesHandlerReturnsCandidates(t *testing.T) {
	repository := &groupHandlerRepository{
		groups: map[string]domaingroup.Group{
			"group-1": {ID: "group-1", InstitutionID: "inst-1", Name: "Grupo A", InstitutionName: "Instituicao A"},
		},
		memberships: map[string][]domaingroup.Membership{},
		candidates: map[string][]domaingroup.Candidate{
			"group-1": {{ID: "user-2", Name: "User 2", Email: "user2@example.com"}},
		},
	}
	recorder := httptest.NewRecorder()
	request := groupRequest(http.MethodGet, "/groups/group-1/candidates", "")
	request.SetPathValue("groupId", "group-1")

	groupCandidatesHandler(applicationgroup.NewService(repository), domainauthorization.NewPolicy(userHandlerMemberships{})).
		ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"email":"user2@example.com"`) || strings.Contains(body, `"status"`) {
		t.Fatalf("body = %s", body)
	}
}
