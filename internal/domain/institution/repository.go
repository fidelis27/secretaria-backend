package institution

import "context"

type Repository interface {
	Create(ctx context.Context, institution Institution) error
	List(ctx context.Context) ([]Institution, error)
	ListByIDs(ctx context.Context, ids []string) ([]Institution, error)
}
