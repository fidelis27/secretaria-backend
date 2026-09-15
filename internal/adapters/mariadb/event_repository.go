package mariadb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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

func (repository EventRepository) List(ctx context.Context, limit int) ([]domainevent.Event, error) {
	if limit < 1 || limit > 100 {
		limit = 30
	}
	rows, err := repository.connection.database.QueryContext(ctx,
		`SELECT event_id, event_type, version, source, correlation_id, occurred_at, payload
		 FROM audit_events ORDER BY occurred_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()
	result := make([]domainevent.Event, 0)
	for rows.Next() {
		var event domainevent.Event
		var payload []byte
		var occurredAt time.Time
		if err := rows.Scan(&event.EventID, &event.Type, &event.Version, &event.Source, &event.CorrelationID, &occurredAt, &payload); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		event.OccurredAt = occurredAt
		event.Payload = json.RawMessage(payload)
		result = append(result, event)
	}
	if err := rows.Err(); err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return result, nil
}
