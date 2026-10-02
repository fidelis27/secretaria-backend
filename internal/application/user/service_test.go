package user

import (
	"context"
	"testing"

	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type repositoryStub struct {
	created           []domainuser.User
	usersByID         map[string]domainuser.User
	usersByAuthSub    map[string]domainuser.User
	updated           []domainuser.User
	activeSuperAdmins int
	links             []struct {
		userID  string
		authSub string
	}
}

func (stub *repositoryStub) Create(_ context.Context, entity domainuser.User) error {
	stub.created = append(stub.created, entity)
	return nil
}

func (*repositoryStub) List(context.Context) ([]domainuser.User, error) {
	return nil, nil
}

func (stub *repositoryStub) FindByID(_ context.Context, id string) (domainuser.User, bool, error) {
	user, found := stub.usersByID[id]
	return user, found, nil
}

func (*repositoryStub) FindByEmail(context.Context, string) (domainuser.User, bool, error) {
	return domainuser.User{}, false, nil
}

func (stub *repositoryStub) FindByAuthSub(_ context.Context, authSub string) (domainuser.User, bool, error) {
	user, found := stub.usersByAuthSub[authSub]
	return user, found, nil
}

func (stub *repositoryStub) LinkAuthSub(_ context.Context, userID string, authSub string) error {
	stub.links = append(stub.links, struct {
		userID  string
		authSub string
	}{userID: userID, authSub: authSub})
	return nil
}

func (stub *repositoryStub) Update(_ context.Context, entity domainuser.User) error {
	stub.updated = append(stub.updated, entity)
	if stub.usersByID == nil {
		stub.usersByID = map[string]domainuser.User{}
	}
	stub.usersByID[entity.ID] = entity
	return nil
}

func (stub *repositoryStub) CountActiveSuperAdmins(context.Context) (int, error) {
	return stub.activeSuperAdmins, nil
}

func TestServiceCreatesActiveUser(t *testing.T) {
	repository := &repositoryStub{}
	created, err := NewService(repository).Create(context.Background(), "  Ana Souza  ", "  ana@example.com  ", true)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.Name != "Ana Souza" || created.Email != "ana@example.com" || created.Status != "active" || !created.SuperAdmin {
		t.Fatalf("unexpected user: %+v", created)
	}
	if len(repository.created) != 1 {
		t.Fatalf("created users = %d, want 1", len(repository.created))
	}
}

func TestServiceRejectsBlankName(t *testing.T) {
	_, err := NewService(&repositoryStub{}).Create(context.Background(), "  ", "ana@example.com", false)
	if err != domainuser.ErrNameRequired {
		t.Fatalf("error = %v, want %v", err, domainuser.ErrNameRequired)
	}
}

func TestServiceFindsAndLinksAuthSub(t *testing.T) {
	repository := &repositoryStub{
		usersByAuthSub: map[string]domainuser.User{
			"sub-1": {ID: "user-1", AuthSub: "sub-1"},
		},
	}
	service := NewService(repository)

	user, found, err := service.FindByAuthSub(context.Background(), " sub-1 ")
	if err != nil || !found || user.ID != "user-1" {
		t.Fatalf("FindByAuthSub() = %+v, %v, %v", user, found, err)
	}
	if err := service.LinkAuthSub(context.Background(), " user-1 ", " sub-2 "); err != nil {
		t.Fatalf("LinkAuthSub() error = %v", err)
	}
	if len(repository.links) != 1 || repository.links[0].userID != "user-1" || repository.links[0].authSub != "sub-2" {
		t.Fatalf("unexpected links: %+v", repository.links)
	}
}

func TestServiceUpdatesNameAndStatus(t *testing.T) {
	repository := &repositoryStub{
		usersByID: map[string]domainuser.User{
			"user-1": {ID: "user-1", Name: "Old", Email: "old@example.com", Status: "active", SuperAdmin: true},
		},
	}
	service := NewService(repository)
	name := "  Novo Nome  "
	status := "inactive"

	updated, err := service.Update(context.Background(), " user-1 ", &name, &status)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Novo Nome" || updated.Status != "inactive" {
		t.Fatalf("updated = %+v", updated)
	}
	if len(repository.updated) != 1 {
		t.Fatalf("updated calls = %d, want 1", len(repository.updated))
	}
}

func TestServiceRejectsInvalidStatus(t *testing.T) {
	repository := &repositoryStub{
		usersByID: map[string]domainuser.User{
			"user-1": {ID: "user-1", Name: "Old", Email: "old@example.com", Status: "active"},
		},
	}
	status := "paused"
	_, err := NewService(repository).Update(context.Background(), "user-1", nil, &status)
	if err != domainuser.ErrStatusInvalid {
		t.Fatalf("error = %v, want %v", err, domainuser.ErrStatusInvalid)
	}
}
