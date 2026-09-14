package user

import "context"

type Repository interface {
	Create(ctx context.Context, user User) error
	List(ctx context.Context) ([]User, error)
	FindByID(ctx context.Context, id string) (User, bool, error)
}
