package user

import (
	"context"
	"testing"

	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type repositoryStub struct {
	created []domainuser.User
}

func (stub *repositoryStub) Create(_ context.Context, entity domainuser.User) error {
	stub.created = append(stub.created, entity)
	return nil
}

func (*repositoryStub) List(context.Context) ([]domainuser.User, error) {
	return nil, nil
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
