package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type authorizationMemberships struct{}

func (authorizationMemberships) ListInstitutionIDsByUser(context.Context, string) ([]string, error) {
	return nil, nil
}

func (authorizationMemberships) FindByUserAndInstitution(context.Context, string, string) ([]domainauthorization.Membership, error) {
	return []domainauthorization.Membership{{Role: domainauthorization.RoleMember}}, nil
}

func (authorizationMemberships) FindByUserAndGroup(context.Context, string, string) (domainauthorization.Membership, bool, error) {
	return domainauthorization.Membership{}, false, nil
}

func TestAuthorizeInstitutionEditRejectsMember(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/students", nil)
	request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, domainuser.User{
		ID:     "member",
		Status: "active",
	}))
	recorder := httptest.NewRecorder()
	allowed := authorizeInstitutionEdit(recorder, request, domainauthorization.NewPolicy(authorizationMemberships{}), "institution-a")

	if allowed {
		t.Fatal("member should not be allowed to edit an institution")
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}
