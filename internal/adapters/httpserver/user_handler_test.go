package httpserver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	applicationuser "github.com/fidelis27/secretaria-backend/internal/application/user"
	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type userHandlerRepository struct {
	created           []domainuser.User
	usersByID         map[string]domainuser.User
	updated           []domainuser.User
	activeSuperAdmins int
}

func (repository *userHandlerRepository) Create(_ context.Context, user domainuser.User) error {
	repository.created = append(repository.created, user)
	return nil
}

func (repository *userHandlerRepository) List(context.Context) ([]domainuser.User, error) {
	return []domainuser.User{{ID: "user-1", Name: "User", Email: "user@example.com", Status: "active"}}, nil
}

func (repository *userHandlerRepository) FindByID(_ context.Context, id string) (domainuser.User, bool, error) {
	user, found := repository.usersByID[id]
	return user, found, nil
}

func (*userHandlerRepository) FindByEmail(context.Context, string) (domainuser.User, bool, error) {
	return domainuser.User{}, false, nil
}

func (*userHandlerRepository) FindByAuthSub(context.Context, string) (domainuser.User, bool, error) {
	return domainuser.User{}, false, nil
}

func (*userHandlerRepository) LinkAuthSub(context.Context, string, string) error {
	return nil
}

func (repository *userHandlerRepository) Update(_ context.Context, user domainuser.User) error {
	repository.updated = append(repository.updated, user)
	if repository.usersByID == nil {
		repository.usersByID = map[string]domainuser.User{}
	}
	repository.usersByID[user.ID] = user
	return nil
}

func (repository *userHandlerRepository) CountActiveSuperAdmins(context.Context) (int, error) {
	return repository.activeSuperAdmins, nil
}

type userHandlerMemberships struct{}

type eventRepositoryStub struct{}

func (eventRepositoryStub) Save(context.Context, domainevent.Event) error { return nil }
func (eventRepositoryStub) List(context.Context, int) ([]domainevent.Event, error) {
	return nil, nil
}
func (eventRepositoryStub) ListByInstitutionIDs(context.Context, int, []string) ([]domainevent.Event, error) {
	return nil, nil
}

func (userHandlerMemberships) ListInstitutionIDsByUser(context.Context, string) ([]string, error) {
	return nil, nil
}

func (userHandlerMemberships) FindByUserAndInstitution(context.Context, string, string) ([]domainauthorization.Membership, error) {
	return nil, nil
}

func (userHandlerMemberships) FindByUserAndGroup(context.Context, string, string) (domainauthorization.Membership, bool, error) {
	return domainauthorization.Membership{}, false, nil
}

func requestWithUser(method string, user domainuser.User, body io.Reader) *http.Request {
	request := httptest.NewRequest(method, "/users", body)
	return request.WithContext(context.WithValue(request.Context(), identityContextKey{}, user))
}

func zeroEventService() applicationevent.Service {
	return applicationevent.NewService(eventRepositoryStub{}, applicationevent.NewBus())
}

func TestUserListHandlerAllowsSuperAdmin(t *testing.T) {
	users := applicationuser.NewService(&userHandlerRepository{})
	policy := domainauthorization.NewPolicy(userHandlerMemberships{})
	recorder := httptest.NewRecorder()

	userListHandler(users, policy).ServeHTTP(recorder, requestWithUser(http.MethodGet, domainuser.User{
		ID: "super", Status: "active", SuperAdmin: true,
	}, nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestUserHandlersRejectRegularUserForListAndCreate(t *testing.T) {
	repository := &userHandlerRepository{}
	users := applicationuser.NewService(repository)
	policy := domainauthorization.NewPolicy(userHandlerMemberships{})
	regularUser := domainuser.User{ID: "member", Status: "active"}

	listRecorder := httptest.NewRecorder()
	userListHandler(users, policy).ServeHTTP(listRecorder, requestWithUser(http.MethodGet, regularUser, nil))
	if listRecorder.Code != http.StatusForbidden {
		t.Fatalf("GET status = %d, want %d", listRecorder.Code, http.StatusForbidden)
	}

	createRecorder := httptest.NewRecorder()
	body := strings.NewReader(`{"name":"Attacker","email":"attacker@example.com","superAdmin":true}`)
	userCreateHandler(users, policy).ServeHTTP(createRecorder, requestWithUser(http.MethodPost, regularUser, body))
	if createRecorder.Code != http.StatusForbidden {
		t.Fatalf("POST status = %d, want %d", createRecorder.Code, http.StatusForbidden)
	}
	if len(repository.created) != 0 {
		t.Fatal("regular user must not create a user")
	}
}

func TestUserCreateHandlerAllowsSuperAdmin(t *testing.T) {
	repository := &userHandlerRepository{}
	users := applicationuser.NewService(repository)
	policy := domainauthorization.NewPolicy(userHandlerMemberships{})
	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"name":"Admin","email":"admin@example.com","superAdmin":true}`)

	userCreateHandler(users, policy).ServeHTTP(recorder, requestWithUser(http.MethodPost, domainuser.User{
		ID: "super", Status: "active", SuperAdmin: true,
	}, body))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if len(repository.created) != 1 || !repository.created[0].SuperAdmin {
		t.Fatal("superadmin should be able to create a superadmin user")
	}
}

func TestUserUpdateHandlerRejectsSelfDeactivation(t *testing.T) {
	repository := &userHandlerRepository{
		usersByID: map[string]domainuser.User{
			"super": {ID: "super", Name: "Admin", Email: "admin@example.com", Status: "active", SuperAdmin: true},
		},
		activeSuperAdmins: 2,
	}
	users := applicationuser.NewService(repository)
	policy := domainauthorization.NewPolicy(userHandlerMemberships{})
	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"status":"inactive"}`)
	request := httptest.NewRequest(http.MethodPatch, "/users/super", body)
	request.SetPathValue("userId", "super")
	request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, domainuser.User{
		ID: "super", Status: "active", SuperAdmin: true,
	}))

	userUpdateHandler(users, policy, zeroEventService()).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestUserUpdateHandlerRejectsLastSuperAdminDeactivation(t *testing.T) {
	repository := &userHandlerRepository{
		usersByID: map[string]domainuser.User{
			"target": {ID: "target", Name: "Admin", Email: "admin@example.com", Status: "active", SuperAdmin: true},
		},
		activeSuperAdmins: 1,
	}
	users := applicationuser.NewService(repository)
	policy := domainauthorization.NewPolicy(userHandlerMemberships{})
	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"status":"inactive"}`)
	request := httptest.NewRequest(http.MethodPatch, "/users/target", body)
	request.SetPathValue("userId", "target")
	request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, domainuser.User{
		ID: "super", Status: "active", SuperAdmin: true,
	}))

	userUpdateHandler(users, policy, zeroEventService()).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
}

func TestUserUpdateHandlerAllowsSuperAdmin(t *testing.T) {
	repository := &userHandlerRepository{
		usersByID: map[string]domainuser.User{
			"user-1": {ID: "user-1", Name: "User", Email: "user@example.com", Status: "active"},
		},
		activeSuperAdmins: 2,
	}
	users := applicationuser.NewService(repository)
	policy := domainauthorization.NewPolicy(userHandlerMemberships{})
	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"name":"Updated User","status":"inactive"}`)
	request := httptest.NewRequest(http.MethodPatch, "/users/user-1", body)
	request.SetPathValue("userId", "user-1")
	request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, domainuser.User{
		ID: "super", Status: "active", SuperAdmin: true,
	}))

	userUpdateHandler(users, policy, zeroEventService()).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if len(repository.updated) != 1 || repository.updated[0].Name != "Updated User" || repository.updated[0].Status != "inactive" {
		t.Fatalf("updated users = %+v", repository.updated)
	}
}
