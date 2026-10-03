package httpserver

import (
	"context"
	"log/slog"

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

func publishDomainEventBestEffort(ctx context.Context, service applicationevent.Service, eventType string, source string, correlationID string, institutionIDs []string, payload any) bool {
	if err := publishDomainEvent(ctx, service, eventType, source, correlationID, institutionIDs, payload); err != nil {
		slog.ErrorContext(ctx, "failed to publish domain event after successful write", "event_type", eventType, "error", err)
		return false
	}
	return true
}
