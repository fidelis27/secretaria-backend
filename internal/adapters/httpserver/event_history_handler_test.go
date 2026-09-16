package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type eventHistoryRepository struct {
	all     []domainevent.Event
	visible []domainevent.Event
}

func (repository eventHistoryRepository) Save(context.Context, domainevent.Event) error { return nil }

func (repository eventHistoryRepository) List(context.Context, int) ([]domainevent.Event, error) {
	return repository.all, nil
}

func (repository eventHistoryRepository) ListByInstitutionIDs(context.Context, int, []string) ([]domainevent.Event, error) {
	return repository.visible, nil
}

type eventHistoryMemberships struct{}

func (eventHistoryMemberships) ListInstitutionIDsByUser(context.Context, string) ([]string, error) {
	return []string{"institution-a"}, nil
}

func (eventHistoryMemberships) FindByUserAndInstitution(context.Context, string, string) ([]domainauthorization.Membership, error) {
	return nil, nil
}

func (eventHistoryMemberships) FindByUserAndGroup(context.Context, string, string) (domainauthorization.Membership, bool, error) {
	return domainauthorization.Membership{}, false, nil
}

func TestEventHistoryHandlerDoesNotLeakCrossInstitutionEvents(t *testing.T) {
	visibleEvent := domainevent.Event{EventID: "event-a", InstitutionIDs: []string{"institution-a"}}
	otherEvent := domainevent.Event{EventID: "event-b", InstitutionIDs: []string{"institution-b"}}
	repository := eventHistoryRepository{all: []domainevent.Event{visibleEvent, otherEvent}, visible: []domainevent.Event{visibleEvent}}
	service := applicationevent.NewService(repository, applicationevent.NewBus())
	policy := domainauthorization.NewPolicy(eventHistoryMemberships{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/events/history?limit=30", nil)
	request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, domainuser.User{
		ID: "member", Status: "active",
	}))

	eventHistoryHandler(service, policy).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var events []domainevent.Event
	if err := json.NewDecoder(recorder.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventID != visibleEvent.EventID {
		t.Fatalf("events = %+v, want only institution-a event", events)
	}
}

func TestEventHistoryHandlerKeepsSuperAdminGlobalView(t *testing.T) {
	events := []domainevent.Event{
		{EventID: "event-a", InstitutionIDs: []string{"institution-a"}},
		{EventID: "event-b", InstitutionIDs: []string{"institution-b"}},
	}
	repository := eventHistoryRepository{all: events}
	service := applicationevent.NewService(repository, applicationevent.NewBus())
	policy := domainauthorization.NewPolicy(eventHistoryMemberships{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/events/history", nil)
	request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, domainuser.User{
		ID: "super", Status: "active", SuperAdmin: true,
	}))

	eventHistoryHandler(service, policy).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var result []domainevent.Event
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result) != len(events) {
		t.Fatalf("events = %+v, want global view", result)
	}
}
