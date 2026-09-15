package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	applicationuser "github.com/fidelis27/secretaria-backend/internal/application/user"
	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type identityRepository struct {
	users map[string]domainuser.User
}

func (repository identityRepository) Create(context.Context, domainuser.User) error {
	return nil
}

func (repository identityRepository) List(context.Context) ([]domainuser.User, error) {
	return nil, nil
}

func (repository identityRepository) FindByID(_ context.Context, id string) (domainuser.User, bool, error) {
	user, found := repository.users[id]
	return user, found, nil
}

func TestIdentityMiddlewareRequiresDemoUser(t *testing.T) {
	handler := identityMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), applicationuser.NewService(identityRepository{}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/institutions", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestIdentityMiddlewareRejectsUnknownAndInactiveUsers(t *testing.T) {
	users := identityRepository{users: map[string]domainuser.User{
		"inactive": {ID: "inactive", Status: "inactive"},
	}}
	handler := identityMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), applicationuser.NewService(users))

	for _, userID := range []string{"unknown", "inactive"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/institutions", nil)
		request.Header.Set("x-demo-user", userID)
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("user %q status = %d, want %d", userID, recorder.Code, http.StatusForbidden)
		}
	}
}

func TestIdentityMiddlewarePassesActiveUserAndPublicRoutes(t *testing.T) {
	users := identityRepository{users: map[string]domainuser.User{
		"active": {ID: "active", Status: "active"},
	}}
	called := false
	handler := identityMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		called = true
		if request.URL.Path != "/health" {
			if _, ok := userFromContext(request.Context()); !ok {
				t.Error("active user missing from context")
			}
		}
		writer.WriteHeader(http.StatusNoContent)
	}), applicationuser.NewService(users))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/institutions", nil)
	request.Header.Set("x-demo-user", "active")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent || !called {
		t.Fatalf("active request status = %d, called = %v", recorder.Code, called)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("health status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestIdentityMiddlewareProtectsUserCreation(t *testing.T) {
	handler := identityMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), applicationuser.NewService(identityRepository{}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/users", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("user creation status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
