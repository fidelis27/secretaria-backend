package enrollment

import (
	"context"
	"errors"
	"testing"

	domainenrollment "github.com/fidelis27/secretaria-backend/internal/domain/enrollment"
)

type transferRepository struct {
	created     []domainenrollment.Enrollment
	transferErr error
}

func (repository *transferRepository) Create(_ context.Context, entity domainenrollment.Enrollment) error {
	repository.created = append(repository.created, entity)
	return nil
}

func (repository *transferRepository) List(_ context.Context) ([]domainenrollment.Enrollment, error) {
	return nil, nil
}

func (repository *transferRepository) Transfer(_ context.Context, studentID string, enrollmentID string, destination domainenrollment.Enrollment) error {
	if repository.transferErr != nil {
		return repository.transferErr
	}
	repository.created = append(repository.created, destination)
	if studentID == "" || enrollmentID == "" {
		return errors.New("unexpected empty transfer identifier")
	}
	return nil
}

func TestServiceTransferCreatesDestinationEnrollment(t *testing.T) {
	repository := &transferRepository{}
	service := NewService(repository)

	destination, err := service.Transfer(context.Background(), "student-1", "enrollment-1", "institution-2")
	if err != nil {
		t.Fatalf("transfer returned error: %v", err)
	}
	if destination.StudentID != "student-1" || destination.InstitutionID != "institution-2" || destination.Status != "active" {
		t.Fatalf("unexpected destination enrollment: %+v", destination)
	}
}

func TestServiceTransferRejectsMissingSourceEnrollment(t *testing.T) {
	service := NewService(&transferRepository{})

	_, err := service.Transfer(context.Background(), "student-1", "", "institution-2")
	if !errors.Is(err, domainenrollment.ErrEnrollmentIDRequired) {
		t.Fatalf("expected missing enrollment error, got %v", err)
	}
}

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

func (*repositoryStub) Transfer(context.Context, string, string, domainenrollment.Enrollment) error {
	return nil
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
