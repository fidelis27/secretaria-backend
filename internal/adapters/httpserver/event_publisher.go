package httpserver

import (
	"context"

	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
)

func publishDomainEvent(ctx context.Context, service applicationevent.Service, eventType string, source string, correlationID string, institutionIDs []string, payload any) error {
	event, err := domainevent.NewForInstitutions(eventType, source, correlationID, institutionIDs, payload)
	if err != nil {
		return err
	}
	return service.Publish(ctx, event)
}
