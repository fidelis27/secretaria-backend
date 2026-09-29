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

type staticIdentityVerifier struct {
	claims IdentityClaims
	err    error
}

func (verifier staticIdentityVerifier) Verify(context.Context, string) (IdentityClaims, error) {
	return verifier.claims, verifier.err
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

func (repository identityRepository) FindByEmail(_ context.Context, email string) (domainuser.User, bool, error) {
	for _, user := range repository.users {
		if user.Email == email {
			return user, true, nil
		}
	}
	return domainuser.User{}, false, nil
}

func TestIdentityMiddlewareRequiresDemoUser(t *testing.T) {
	handler := identityMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), applicationuser.NewService(identityRepository{}), staticIdentityVerifier{})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/institutions", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestIdentityMiddlewareRejectsUnknownAndInactiveUsers(t *testing.T) {
	users := identityRepository{users: map[string]domainuser.User{
		"inactive": {ID: "inactive", Email: "inactive@example.com", Status: "inactive"},
	}}
	handler := identityMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), applicationuser.NewService(users), staticIdentityVerifier{claims: IdentityClaims{Email: "unknown@example.com", EmailVerified: true}})

	for _, email := range []string{"unknown@example.com", "inactive@example.com"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/institutions", nil)
		request.Header.Set("Authorization", "Bearer test-token")
		if email == "inactive@example.com" {
			handler = identityMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), applicationuser.NewService(users), staticIdentityVerifier{claims: IdentityClaims{Email: email, EmailVerified: true}})
		}
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("email %q status = %d, want %d", email, recorder.Code, http.StatusForbidden)
		}
	}
}

func TestIdentityMiddlewarePassesActiveUserAndPublicRoutes(t *testing.T) {
	users := identityRepository{users: map[string]domainuser.User{
		"active": {ID: "active", Email: "active@example.com", Status: "active", SuperAdmin: true},
	}}
	called := false
	handler := identityMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		called = true
		if request.URL.Path != "/health" {
			if _, ok := userFromContext(request.Context()); !ok {
				t.Error("active user missing from context")
			}
			if _, ok := identityClaimsFromContext(request.Context()); !ok {
				t.Error("identity claims missing from context")
			}
		}
		writer.WriteHeader(http.StatusNoContent)
	}), applicationuser.NewService(users), staticIdentityVerifier{claims: IdentityClaims{Subject: "subject", Email: "active@example.com", EmailVerified: true, Roles: []string{"super_admin"}}})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/institutions", nil)
	request.Header.Set("Authorization", "Bearer test-token")
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
	handler := identityMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), applicationuser.NewService(identityRepository{}), staticIdentityVerifier{})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/users", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("user creation status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
