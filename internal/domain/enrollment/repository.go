package enrollment

import "context"

type Repository interface {
	Create(ctx context.Context, enrollment Enrollment) error
	List(ctx context.Context) ([]Enrollment, error)
}
