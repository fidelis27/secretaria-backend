package student

import "context"

type Repository interface {
	Create(ctx context.Context, student Student) error
	List(ctx context.Context) ([]Student, error)
	ListByInstitutionIDs(ctx context.Context, institutionIDs []string) ([]Student, error)
}
