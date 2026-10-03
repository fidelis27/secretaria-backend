package httpserver

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
)

type failingEventRepository struct {
	err error
}

func (repository failingEventRepository) Save(context.Context, domainevent.Event) error {
	return repository.err
}

func (failingEventRepository) List(context.Context, int) ([]domainevent.Event, error) {
	return nil, nil
}

func (failingEventRepository) ListByInstitutionIDs(context.Context, int, []string) ([]domainevent.Event, error) {
	return nil, nil
}

func TestPublishDomainEventBestEffortLogsFailure(t *testing.T) {
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	service := applicationevent.NewService(failingEventRepository{err: errors.New("database unavailable")}, applicationevent.NewBus())
	if published := publishDomainEventBestEffort(context.Background(), service, "STUDENT_CREATED", "backend.student", "correlation-1", []string{"institution-1"}, map[string]string{"id": "student-1"}); published {
		t.Fatal("publishDomainEventBestEffort() reported success after repository failure")
	}
	if !strings.Contains(logs.String(), "failed to publish domain event after successful write") {
		t.Fatalf("expected a non-fatal publication error log, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "STUDENT_CREATED") {
		t.Fatalf("expected the event type in the log, got %q", logs.String())
	}
}

func TestMutationAuditPayloadIncludesActorTargetAndBeforeAfter(t *testing.T) {
	payload := mutationAuditPayload(
		"actor-1",
		"user-2",
		map[string]any{"role": "member"},
		map[string]any{"role": "admin"},
	)
	if payload["actorId"] != "actor-1" || payload["targetId"] != "user-2" {
		t.Fatalf("audit actor/target missing: %#v", payload)
	}
	if payload["before"].(map[string]any)["role"] != "member" || payload["after"].(map[string]any)["role"] != "admin" {
		t.Fatalf("audit before/after values missing: %#v", payload)
	}
}
