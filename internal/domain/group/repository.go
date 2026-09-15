package group

import "context"

type Repository interface {
	Create(ctx context.Context, entity Group) error
	List(ctx context.Context) ([]Group, error)
	ListByInstitutionIDs(ctx context.Context, institutionIDs []string) ([]Group, error)
	FindByID(ctx context.Context, id string) (Group, bool, error)
	AddMembership(ctx context.Context, membership Membership) error
	ListMemberships(ctx context.Context, groupID string) ([]Membership, error)
}
