package authorization

import (
	"context"

	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type Policy struct {
	memberships MembershipRepository
}

func NewPolicy(memberships MembershipRepository) Policy {
	return Policy{memberships: memberships}
}

func (policy Policy) CanReadInstitution(ctx context.Context, user domainuser.User, institutionID string) (bool, error) {
	if user.Status != "active" {
		return false, nil
	}
	if user.SuperAdmin {
		return true, nil
	}
	memberships, err := policy.memberships.FindByUserAndInstitution(ctx, user.ID, institutionID)
	if err != nil {
		return false, err
	}
	return len(memberships) > 0, nil
}

func (policy Policy) CanEditInstitution(ctx context.Context, user domainuser.User, institutionID string) (bool, error) {
	if user.Status != "active" {
		return false, nil
	}
	if user.SuperAdmin {
		return true, nil
	}
	memberships, err := policy.memberships.FindByUserAndInstitution(ctx, user.ID, institutionID)
	if err != nil {
		return false, err
	}
	for _, membership := range memberships {
		if membership.Role == RoleAdmin {
			return true, nil
		}
	}
	return false, nil
}

func (policy Policy) CanCreateGroup(user domainuser.User) bool {
	return user.Status == "active" && user.SuperAdmin
}

func (policy Policy) CanManageUsers(user domainuser.User) bool {
	return user.Status == "active" && user.SuperAdmin
}

func (policy Policy) CanManageGroup(ctx context.Context, user domainuser.User, groupID string) (bool, error) {
	if user.Status != "active" {
		return false, nil
	}
	if user.SuperAdmin {
		return true, nil
	}
	membership, found, err := policy.memberships.FindByUserAndGroup(ctx, user.ID, groupID)
	if err != nil {
		return false, err
	}
	return found && membership.Role == RoleAdmin, nil
}
