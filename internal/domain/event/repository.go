package event

import "context"

type Repository interface {
	Save(ctx context.Context, event Event) error
	List(ctx context.Context, limit int) ([]Event, error)
	ListByInstitutionIDs(ctx context.Context, limit int, institutionIDs []string) ([]Event, error)
}

type Subscriber interface {
	Subscribe() (<-chan Event, func())
}
