package mariadb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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
	transaction, err := repository.connection.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save event: %w", err)
	}
	defer transaction.Rollback()
	_, err = transaction.ExecContext(ctx,
		`INSERT INTO audit_events (event_id, event_type, version, source, correlation_id, occurred_at, payload)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.EventID, event.Type, event.Version, event.Source, event.CorrelationID, event.OccurredAt, event.Payload,
	)
	if err != nil {
		return fmt.Errorf("save event: %w", err)
	}
	for _, institutionID := range event.InstitutionIDs {
		if _, err := transaction.ExecContext(ctx,
			`INSERT INTO audit_event_institutions (event_id, institution_id) VALUES (?, ?)`, event.EventID, institutionID,
		); err != nil {
			return fmt.Errorf("save event institution scope: %w", err)
		}
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit event: %w", err)
	}
	return nil
}

func (repository EventRepository) List(ctx context.Context, limit int) ([]domainevent.Event, error) {
	if limit < 1 || limit > 100 {
		limit = 30
	}
	return repository.list(ctx, `SELECT ae.event_id, ae.event_type, ae.version, ae.source, ae.correlation_id, ae.occurred_at, ae.payload,
		CASE WHEN COUNT(aei.institution_id) = 0 THEN JSON_ARRAY() ELSE JSON_ARRAYAGG(aei.institution_id) END AS institution_ids
		FROM audit_events ae LEFT JOIN audit_event_institutions aei ON aei.event_id = ae.event_id
		GROUP BY ae.event_id, ae.event_type, ae.version, ae.source, ae.correlation_id, ae.occurred_at, ae.payload
		ORDER BY ae.occurred_at DESC LIMIT ?`, limit)
}

func (repository EventRepository) ListByInstitutionIDs(ctx context.Context, limit int, institutionIDs []string) ([]domainevent.Event, error) {
	if len(institutionIDs) == 0 {
		return []domainevent.Event{}, nil
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(institutionIDs)), ",")
	args := make([]any, 0, len(institutionIDs)+1)
	for _, institutionID := range institutionIDs {
		args = append(args, institutionID)
	}
	args = append(args, limit)
	return repository.list(ctx, `SELECT ae.event_id, ae.event_type, ae.version, ae.source, ae.correlation_id, ae.occurred_at, ae.payload,
		CASE WHEN COUNT(scope.institution_id) = 0 THEN JSON_ARRAY() ELSE JSON_ARRAYAGG(scope.institution_id) END AS institution_ids
		FROM audit_events ae
		INNER JOIN audit_event_institutions visible ON visible.event_id = ae.event_id
		LEFT JOIN audit_event_institutions scope ON scope.event_id = ae.event_id
		WHERE visible.institution_id IN (`+placeholders+`)
		GROUP BY ae.event_id, ae.event_type, ae.version, ae.source, ae.correlation_id, ae.occurred_at, ae.payload
		ORDER BY ae.occurred_at DESC LIMIT ?`, args...)
}

func (repository EventRepository) list(ctx context.Context, query string, args ...any) ([]domainevent.Event, error) {
	rows, err := repository.connection.database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()
	result := make([]domainevent.Event, 0)
	for rows.Next() {
		var event domainevent.Event
		var payload []byte
		var institutionIDs []byte
		var occurredAt time.Time
		if err := rows.Scan(&event.EventID, &event.Type, &event.Version, &event.Source, &event.CorrelationID, &occurredAt, &payload, &institutionIDs); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		if err := json.Unmarshal(institutionIDs, &event.InstitutionIDs); err != nil {
			return nil, fmt.Errorf("decode event institution scope: %w", err)
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
