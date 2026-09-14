package mariadb

import (
	"context"
	"fmt"

	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
)

type EventRepository struct {
	connection *Connection
}

func NewEventRepository(connection *Connection) EventRepository {
	return EventRepository{connection: connection}
}

func (repository EventRepository) Save(ctx context.Context, event domainevent.Event) error {
	_, err := repository.connection.database.ExecContext(ctx,
		`INSERT INTO audit_events (event_id, event_type, version, source, correlation_id, occurred_at, payload)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.EventID, event.Type, event.Version, event.Source, event.CorrelationID, event.OccurredAt, event.Payload,
	)
	if err != nil {
		return fmt.Errorf("save event: %w", err)
	}
	return nil
}
