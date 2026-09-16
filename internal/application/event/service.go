package event

import (
	"context"

	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
)

type Publisher interface {
	Publish(ctx context.Context, event domainevent.Event) error
}

type Service struct {
	repository  domainevent.Repository
	broadcaster *Bus
}

func NewService(repository domainevent.Repository, broadcaster *Bus) Service {
	return Service{repository: repository, broadcaster: broadcaster}
}

func (service Service) Publish(ctx context.Context, event domainevent.Event) error {
	if err := service.repository.Save(ctx, event); err != nil {
		return err
	}
	service.broadcaster.Publish(event)
	return nil
}

func (service Service) Subscribe() (<-chan domainevent.Event, func()) {
	return service.broadcaster.Subscribe()
}

func (service Service) List(ctx context.Context, limit int) ([]domainevent.Event, error) {
	return service.repository.List(ctx, limit)
}

func (service Service) ListByInstitutionIDs(ctx context.Context, limit int, institutionIDs []string) ([]domainevent.Event, error) {
	return service.repository.ListByInstitutionIDs(ctx, limit, institutionIDs)
}
