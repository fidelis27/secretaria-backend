package enrollment

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, enrollment Enrollment) error
	List(ctx context.Context) ([]Enrollment, error)
	FindByID(ctx context.Context, studentID string, enrollmentID string) (Enrollment, bool, error)
	Transfer(ctx context.Context, studentID string, enrollmentID string, destination Enrollment) error
	Suspend(ctx context.Context, studentID string, enrollmentID string, reason string, suspendedAt time.Time) error
	Reopen(ctx context.Context, studentID string, enrollmentID string) error
}
