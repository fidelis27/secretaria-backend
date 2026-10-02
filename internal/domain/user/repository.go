package user

import "context"

type Repository interface {
	Create(ctx context.Context, user User) error
	List(ctx context.Context) ([]User, error)
	FindByID(ctx context.Context, id string) (User, bool, error)
	FindByEmail(ctx context.Context, email string) (User, bool, error)
	FindByAuthSub(ctx context.Context, authSub string) (User, bool, error)
	LinkAuthSub(ctx context.Context, userID string, authSub string) error
}
