package event

import "context"

type Repository interface {
	Save(ctx context.Context, event Event) error
}

type Subscriber interface {
	Subscribe() (<-chan Event, func())
}
