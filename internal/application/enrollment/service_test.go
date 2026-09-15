package enrollment

import (
	"context"
	"errors"
	"testing"
	"time"

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

func (*transferRepository) FindByID(context.Context, string, string) (domainenrollment.Enrollment, bool, error) {
	return domainenrollment.Enrollment{}, false, nil
}

func (*transferRepository) ListByInstitutionIDs(context.Context, []string) ([]domainenrollment.Enrollment, error) {
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

func (*transferRepository) Suspend(context.Context, string, string, string, time.Time) error {
	return nil
}

func (*transferRepository) Reopen(context.Context, string, string) error {
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

func TestServiceSuspendRequiresReason(t *testing.T) {
	service := NewService(&transferRepository{})

	err := service.Suspend(context.Background(), "student-1", "enrollment-1", "  ", time.Now())
	if !errors.Is(err, domainenrollment.ErrSuspensionReasonRequired) {
		t.Fatalf("expected suspension reason error, got %v", err)
	}
}

func TestServiceSuspendRequiresDate(t *testing.T) {
	service := NewService(&transferRepository{})

	err := service.Suspend(context.Background(), "student-1", "enrollment-1", "leave", time.Time{})
	if !errors.Is(err, domainenrollment.ErrSuspensionDateRequired) {
		t.Fatalf("expected suspension date error, got %v", err)
	}
}

func TestServiceSuspendAndReopen(t *testing.T) {
	service := NewService(&transferRepository{})

	if err := service.Suspend(context.Background(), "student-1", "enrollment-1", "leave", time.Now()); err != nil {
		t.Fatalf("suspend returned error: %v", err)
	}
	if err := service.Reopen(context.Background(), "student-1", "enrollment-1"); err != nil {
		t.Fatalf("reopen returned error: %v", err)
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

func (*repositoryStub) FindByID(context.Context, string, string) (domainenrollment.Enrollment, bool, error) {
	return domainenrollment.Enrollment{}, false, nil
}

func (*repositoryStub) ListByInstitutionIDs(context.Context, []string) ([]domainenrollment.Enrollment, error) {
	return nil, nil
}

func (*repositoryStub) Transfer(context.Context, string, string, domainenrollment.Enrollment) error {
	return nil
}

func (*repositoryStub) Suspend(context.Context, string, string, string, time.Time) error {
	return nil
}

func (*repositoryStub) Reopen(context.Context, string, string) error {
	return nil
}

type lookupRepository struct {
	repositoryStub
	stored domainenrollment.Enrollment
}

func (repository *lookupRepository) FindByID(_ context.Context, studentID string, enrollmentID string) (domainenrollment.Enrollment, bool, error) {
	if repository.stored.StudentID == studentID && repository.stored.ID == enrollmentID {
		return repository.stored, true, nil
	}
	return domainenrollment.Enrollment{}, false, nil
}

func TestServiceFindByIDReturnsMatchingEnrollment(t *testing.T) {
	repository := &lookupRepository{stored: domainenrollment.Enrollment{ID: "enrollment-1", StudentID: "student-1", InstitutionID: "institution-1", Status: "active"}}
	found, ok, err := NewService(repository).FindByID(context.Background(), "student-1", "enrollment-1")
	if err != nil || !ok || found.InstitutionID != "institution-1" {
		t.Fatalf("unexpected lookup result: found=%v ok=%v err=%v", found, ok, err)
	}
}

func TestServiceFindByIDReturnsFalseWhenMissing(t *testing.T) {
	_, ok, err := NewService(&lookupRepository{}).FindByID(context.Background(), "student-1", "enrollment-1")
	if err != nil || ok {
		t.Fatalf("expected missing enrollment, ok=%v err=%v", ok, err)
	}
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
