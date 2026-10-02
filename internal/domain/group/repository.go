package group

import (
	"context"

	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
)

type Repository interface {
	Create(ctx context.Context, entity Group) error
	List(ctx context.Context) ([]Group, error)
	ListByInstitutionIDs(ctx context.Context, institutionIDs []string) ([]Group, error)
	FindByID(ctx context.Context, id string) (Group, bool, error)
	AddMembership(ctx context.Context, membership Membership) (Membership, error)
	UpdateMembershipRole(ctx context.Context, groupID string, userID string, role domainauthorization.Role) (Membership, error)
	RemoveMembership(ctx context.Context, groupID string, userID string) error
	ListMemberships(ctx context.Context, groupID string) ([]Membership, error)
	ListCandidates(ctx context.Context, groupID string) ([]Candidate, error)
}
