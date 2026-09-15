package group

import (
	"context"
	"testing"

	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domaingroup "github.com/fidelis27/secretaria-backend/internal/domain/group"
)

type repositoryStub struct {
	createdGroup      domaingroup.Group
	createdMembership domaingroup.Membership
}

func (stub *repositoryStub) Create(_ context.Context, entity domaingroup.Group) error {
	stub.createdGroup = entity
	return nil
}

func (*repositoryStub) List(context.Context) ([]domaingroup.Group, error) { return nil, nil }

func (*repositoryStub) ListByInstitutionIDs(context.Context, []string) ([]domaingroup.Group, error) {
	return nil, nil
}

func (stub *repositoryStub) FindByID(_ context.Context, id string) (domaingroup.Group, bool, error) {
	return domaingroup.Group{ID: id, InstitutionID: "institution-1"}, true, nil
}

func (stub *repositoryStub) AddMembership(_ context.Context, entity domaingroup.Membership) error {
	stub.createdMembership = entity
	return nil
}

func (*repositoryStub) ListMemberships(context.Context, string) ([]domaingroup.Membership, error) {
	return nil, nil
}

func TestServiceCreatesGroupAndMembership(t *testing.T) {
	stub := &repositoryStub{}
	service := NewService(stub)

	createdGroup, err := service.Create(context.Background(), " institution-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if createdGroup.InstitutionID != "institution-1" || stub.createdGroup.ID == "" {
		t.Fatalf("group = %+v", createdGroup)
	}

	createdMembership, err := service.AddMembership(context.Background(), " user-1 ", createdGroup.ID, createdGroup.InstitutionID, domainauthorization.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if createdMembership.UserID != "user-1" || createdMembership.Role != domainauthorization.RoleAdmin {
		t.Fatalf("membership = %+v", createdMembership)
	}
}
