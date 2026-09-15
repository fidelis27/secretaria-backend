package mariadb

import (
	"context"
	"os"
	"testing"
	"time"

	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
	"github.com/fidelis27/secretaria-backend/internal/platform/config"
)

func integrationConnection(t *testing.T) *Connection {
	t.Helper()
	if os.Getenv("BACKEND_INTEGRATION") != "1" {
		t.Skip("set BACKEND_INTEGRATION=1 to run MariaDB integration tests")
	}
	connection, err := Open(config.Config{
		DBHost: "127.0.0.1",
		DBPort: 3306,
		DBName: "test",
		DBUser: "root",
	})
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	return connection
}

func TestEventRepositoryPersistsAndListsEvents(t *testing.T) {
	connection := integrationConnection(t)
	repository := NewEventRepository(connection)
	event := domainevent.Event{
		EventID:       "integration-event-" + time.Now().UTC().Format("20060102150405.000000000"),
		Type:          "INTEGRATION_TEST",
		Version:       1,
		Source:        "integration",
		CorrelationID: "integration-correlation",
		OccurredAt:    time.Now().UTC(),
		Payload:       []byte(`{"ok":true}`),
	}

	if err := repository.Save(context.Background(), event); err != nil {
		t.Fatalf("save event: %v", err)
	}
	listed, err := repository.List(context.Background(), 1)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(listed) != 1 || listed[0].EventID != event.EventID || listed[0].Type != event.Type {
		t.Fatalf("unexpected listed events: %+v", listed)
	}
}
