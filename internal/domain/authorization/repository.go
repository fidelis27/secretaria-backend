package authorization

import "context"

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type Membership struct {
	UserID        string
	GroupID       string
	InstitutionID string
	Role          Role
}

type MembershipRepository interface {
	ListInstitutionIDsByUser(ctx context.Context, userID string) ([]string, error)
	FindByUserAndInstitution(ctx context.Context, userID string, institutionID string) ([]Membership, error)
	FindByUserAndGroup(ctx context.Context, userID string, groupID string) (Membership, bool, error)
}
