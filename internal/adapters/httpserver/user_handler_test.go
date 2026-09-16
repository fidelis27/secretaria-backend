package httpserver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	applicationuser "github.com/fidelis27/secretaria-backend/internal/application/user"
	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type userHandlerRepository struct {
	created []domainuser.User
}

func (repository *userHandlerRepository) Create(_ context.Context, user domainuser.User) error {
	repository.created = append(repository.created, user)
	return nil
}

func (repository *userHandlerRepository) List(context.Context) ([]domainuser.User, error) {
	return []domainuser.User{{ID: "user-1", Name: "User", Email: "user@example.com", Status: "active"}}, nil
}

func (*userHandlerRepository) FindByID(context.Context, string) (domainuser.User, bool, error) {
	return domainuser.User{}, false, nil
}

type userHandlerMemberships struct{}

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
