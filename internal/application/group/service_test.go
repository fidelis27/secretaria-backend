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
	candidates        []domaingroup.Candidate
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
	return domaingroup.Group{ID: id, InstitutionID: "institution-1", Name: "Grupo A", InstitutionName: "Instituicao A"}, true, nil
}

func (stub *repositoryStub) AddMembership(_ context.Context, entity domaingroup.Membership) (domaingroup.Membership, error) {
	stub.createdMembership = entity
	entity.UserName = "User 1"
	entity.UserEmail = "user1@example.com"
	return entity, nil
}

func (*repositoryStub) UpdateMembershipRole(_ context.Context, groupID string, userID string, role domainauthorization.Role) (domaingroup.Membership, error) {
	return domaingroup.Membership{UserID: userID, GroupID: groupID, InstitutionID: "institution-1", Role: role}, nil
}

func (*repositoryStub) RemoveMembership(context.Context, string, string) error {
	return nil
}

func (*repositoryStub) ListMemberships(context.Context, string) ([]domaingroup.Membership, error) {
	return nil, nil
}

func (stub *repositoryStub) ListCandidates(context.Context, string) ([]domaingroup.Candidate, error) {
	return stub.candidates, nil
}

func TestServiceCreatesGroupAndMembership(t *testing.T) {
	stub := &repositoryStub{}
	service := NewService(stub)

	createdGroup, err := service.Create(context.Background(), " institution-1 ", "  Grupo A  ")
	if err != nil {
		t.Fatal(err)
	}
	if createdGroup.InstitutionID != "institution-1" || createdGroup.Name != "Grupo A" || stub.createdGroup.ID == "" {
		t.Fatalf("group = %+v", createdGroup)
	}

	createdMembership, err := service.AddMembership(context.Background(), " user-1 ", createdGroup.ID, createdGroup.InstitutionID, domainauthorization.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if createdMembership.UserID != "user-1" || createdMembership.Role != domainauthorization.RoleAdmin || createdMembership.UserEmail != "user1@example.com" {
		t.Fatalf("membership = %+v", createdMembership)
	}
}

func TestServiceListsCandidates(t *testing.T) {
	stub := &repositoryStub{
		candidates: []domaingroup.Candidate{{ID: "user-2", Name: "User 2", Email: "user2@example.com"}},
	}
	candidates, err := NewService(stub).ListCandidates(context.Background(), " group-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].ID != "user-2" {
		t.Fatalf("candidates = %+v", candidates)
	}
}
