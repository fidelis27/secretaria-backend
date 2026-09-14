package student

import (
	"context"
	"testing"

	domainstudent "github.com/fidelis27/secretaria-backend/internal/domain/student"
)

type repositoryStub struct {
	created []domainstudent.Student
}

func (stub *repositoryStub) Create(_ context.Context, entity domainstudent.Student) error {
	stub.created = append(stub.created, entity)
	return nil
}

func (*repositoryStub) List(context.Context) ([]domainstudent.Student, error) {
	return nil, nil
}

func TestServiceCreatesActiveStudent(t *testing.T) {
	repository := &repositoryStub{}
	created, err := NewService(repository).Create(context.Background(), "  Maria Silva  ", "  inst-123  ")
	if err != nil {
		t.Fatalf("create student: %v", err)
	}
	if created.Name != "Maria Silva" || created.InstitutionID != "inst-123" || created.Status != "active" {
		t.Fatalf("unexpected student: %+v", created)
	}
	if len(repository.created) != 1 {
		t.Fatalf("created students = %d, want 1", len(repository.created))
	}
}

func TestServiceRejectsBlankName(t *testing.T) {
	_, err := NewService(&repositoryStub{}).Create(context.Background(), "  ", "inst-123")
	if err != domainstudent.ErrNameRequired {
		t.Fatalf("error = %v, want %v", err, domainstudent.ErrNameRequired)
	}
}
