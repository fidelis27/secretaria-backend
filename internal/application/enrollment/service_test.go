package enrollment

import (
	"context"
	"testing"

	domainenrollment "github.com/fidelis27/secretaria-backend/internal/domain/enrollment"
)

type repositoryStub struct {
	created []domainenrollment.Enrollment
}

func (stub *repositoryStub) Create(_ context.Context, entity domainenrollment.Enrollment) error {
	stub.created = append(stub.created, entity)
	return nil
}

func (*repositoryStub) List(context.Context) ([]domainenrollment.Enrollment, error) {
	return nil, nil
}

func TestServiceCreatesActiveEnrollment(t *testing.T) {
	repository := &repositoryStub{}
	created, err := NewService(repository).Create(context.Background(), "student-1", "institution-1")
	if err != nil {
		t.Fatalf("create enrollment: %v", err)
	}
	if created.StudentID != "student-1" || created.InstitutionID != "institution-1" || created.Status != "active" {
		t.Fatalf("unexpected enrollment: %+v", created)
	}
	if len(repository.created) != 1 {
		t.Fatalf("created enrollments = %d, want 1", len(repository.created))
	}
}

func TestServiceRejectsBlankStudentID(t *testing.T) {
	_, err := NewService(&repositoryStub{}).Create(context.Background(), "  ", "institution-1")
	if err != domainenrollment.ErrStudentIDRequired {
		t.Fatalf("error = %v, want %v", err, domainenrollment.ErrStudentIDRequired)
	}
}
