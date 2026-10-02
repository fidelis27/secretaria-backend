package bootstrap

import (
	"context"
	"testing"

	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type repositoryStub struct {
	superAdminCount int
	usersByEmail    map[string]domainuser.User
	created         []domainuser.User
	promotedEmail   string
}

func (stub *repositoryStub) CountSuperAdmins(context.Context) (int, error) {
	return stub.superAdminCount, nil
}

func (stub *repositoryStub) FindByEmail(_ context.Context, email string) (domainuser.User, bool, error) {
	user, found := stub.usersByEmail[email]
	return user, found, nil
}

func (stub *repositoryStub) Create(_ context.Context, user domainuser.User) error {
	stub.created = append(stub.created, user)
	return nil
}

func (stub *repositoryStub) PromoteByEmail(_ context.Context, email string) error {
	stub.promotedEmail = email
	return nil
}

func TestEnsureFirstSuperAdminCreatesMissingUser(t *testing.T) {
	repository := &repositoryStub{usersByEmail: map[string]domainuser.User{}}

	created, err := NewService(repository).EnsureFirstSuperAdmin(context.Background(), "  admin@example.com  ")
	if err != nil {
		t.Fatalf("EnsureFirstSuperAdmin() error = %v", err)
	}
	if !created {
		t.Fatal("expected bootstrap to create a user")
	}
	if len(repository.created) != 1 {
		t.Fatalf("created users = %d, want 1", len(repository.created))
	}
	if repository.created[0].Email != "admin@example.com" || !repository.created[0].SuperAdmin {
		t.Fatalf("unexpected user: %+v", repository.created[0])
	}
}

func TestEnsureFirstSuperAdminPromotesExistingUser(t *testing.T) {
	repository := &repositoryStub{
		usersByEmail: map[string]domainuser.User{
			"admin@example.com": {ID: "user-1", Email: "admin@example.com", Status: "inactive"},
		},
	}

	created, err := NewService(repository).EnsureFirstSuperAdmin(context.Background(), "admin@example.com")
	if err != nil {
		t.Fatalf("EnsureFirstSuperAdmin() error = %v", err)
	}
	if !created {
		t.Fatal("expected bootstrap to promote the existing user")
	}
	if repository.promotedEmail != "admin@example.com" {
		t.Fatalf("promoted email = %q", repository.promotedEmail)
	}
	if len(repository.created) != 0 {
		t.Fatal("existing user should not be recreated")
	}
}

func TestEnsureFirstSuperAdminSkipsWhenAlreadyPresent(t *testing.T) {
	repository := &repositoryStub{
		superAdminCount: 1,
		usersByEmail:    map[string]domainuser.User{},
	}

	created, err := NewService(repository).EnsureFirstSuperAdmin(context.Background(), "admin@example.com")
	if err != nil {
		t.Fatalf("EnsureFirstSuperAdmin() error = %v", err)
	}
	if created {
		t.Fatal("bootstrap should be skipped when a superadmin already exists")
	}
	if repository.promotedEmail != "" || len(repository.created) != 0 {
		t.Fatal("bootstrap should not modify users when a superadmin already exists")
	}
}
