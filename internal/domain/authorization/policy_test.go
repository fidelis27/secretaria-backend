package authorization

import (
	"context"
	"testing"

	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type membershipRepository struct {
	memberships []Membership
}

func (repository membershipRepository) ListInstitutionIDsByUser(_ context.Context, userID string) ([]string, error) {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, membership := range repository.memberships {
		if membership.UserID == userID && !seen[membership.InstitutionID] {
			seen[membership.InstitutionID] = true
			result = append(result, membership.InstitutionID)
		}
	}
	return result, nil
}

func (repository membershipRepository) FindByUserAndInstitution(_ context.Context, userID string, institutionID string) ([]Membership, error) {
	result := make([]Membership, 0)
	for _, membership := range repository.memberships {
		if membership.UserID == userID && membership.InstitutionID == institutionID {
			result = append(result, membership)
		}
	}
	return result, nil
}

func (repository membershipRepository) FindByUserAndGroup(_ context.Context, userID string, groupID string) (Membership, bool, error) {
	for _, membership := range repository.memberships {
		if membership.UserID == userID && membership.GroupID == groupID {
			return membership, true, nil
		}
	}
	return Membership{}, false, nil
}

func TestPolicyScopesAccessByRole(t *testing.T) {
	policy := NewPolicy(membershipRepository{memberships: []Membership{
		{UserID: "admin", GroupID: "group-a", InstitutionID: "institution-a", Role: RoleAdmin},
		{UserID: "member", GroupID: "group-a", InstitutionID: "institution-a", Role: RoleMember},
	}})
	admin := domainuser.User{ID: "admin", Status: "active"}
	member := domainuser.User{ID: "member", Status: "active"}
	outOfScope := domainuser.User{ID: "outside", Status: "active"}

	canEdit, err := policy.CanEditInstitution(context.Background(), admin, "institution-a")
	if err != nil || !canEdit {
		t.Fatalf("admin edit access = %v, %v", canEdit, err)
	}
	canRead, err := policy.CanReadInstitution(context.Background(), member, "institution-a")
	if err != nil || !canRead {
		t.Fatalf("member read access = %v, %v", canRead, err)
	}
	canEdit, err = policy.CanEditInstitution(context.Background(), member, "institution-a")
	if err != nil || canEdit {
		t.Fatalf("member edit access = %v, %v", canEdit, err)
	}
	canRead, err = policy.CanReadInstitution(context.Background(), outOfScope, "institution-a")
	if err != nil || canRead {
		t.Fatalf("out-of-scope read access = %v, %v", canRead, err)
	}
}

func TestPolicySuperAdminBypassesScope(t *testing.T) {
	policy := NewPolicy(membershipRepository{})
	superAdmin := domainuser.User{ID: "super", Status: "active", SuperAdmin: true}

	canEdit, err := policy.CanEditInstitution(context.Background(), superAdmin, "institution-z")
	if err != nil || !canEdit || !policy.CanCreateGroup(superAdmin) || !policy.CanManageUsers(superAdmin) {
		t.Fatalf("super-admin permissions = edit:%v create:%v users:%v err:%v", canEdit, policy.CanCreateGroup(superAdmin), policy.CanManageUsers(superAdmin), err)
	}
}

func TestPolicyGroupManagementRequiresAdminRole(t *testing.T) {
	policy := NewPolicy(membershipRepository{memberships: []Membership{
		{UserID: "member", GroupID: "group-a", Role: RoleMember},
		{UserID: "admin", GroupID: "group-a", Role: RoleAdmin},
	}})

	canManage, err := policy.CanManageGroup(context.Background(), domainuser.User{ID: "member", Status: "active"}, "group-a")
	if err != nil || canManage {
		t.Fatalf("member group management = %v, %v", canManage, err)
	}
	canManage, err = policy.CanManageGroup(context.Background(), domainuser.User{ID: "admin", Status: "active"}, "group-a")
	if err != nil || !canManage {
		t.Fatalf("admin group management = %v, %v", canManage, err)
	}
}

func TestPolicyDeniesInactiveSuperAdmin(t *testing.T) {
	policy := NewPolicy(membershipRepository{})
	inactive := domainuser.User{ID: "super", Status: "inactive", SuperAdmin: true}

	canRead, err := policy.CanReadInstitution(context.Background(), inactive, "institution-a")
	if err != nil || canRead || policy.CanCreateGroup(inactive) {
		t.Fatalf("inactive permissions = read:%v create:%v err:%v", canRead, policy.CanCreateGroup(inactive), err)
	}
}

func TestPolicyListsVisibleInstitutions(t *testing.T) {
	policy := NewPolicy(membershipRepository{memberships: []Membership{
		{UserID: "member", InstitutionID: "institution-a"},
		{UserID: "member", InstitutionID: "institution-a"},
		{UserID: "member", InstitutionID: "institution-b"},
	}})

	ids, err := policy.VisibleInstitutionIDs(context.Background(), domainuser.User{ID: "member", Status: "active"})
	if err != nil || len(ids) != 2 {
		t.Fatalf("visible institution ids = %v, err = %v", ids, err)
	}
}
