package enrollment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	domainenrollment "github.com/fidelis27/secretaria-backend/internal/domain/enrollment"
)

type Service struct {
	repository domainenrollment.Repository
}

func NewService(repository domainenrollment.Repository) Service {
	return Service{repository: repository}
}

func (service Service) Create(ctx context.Context, studentID string, institutionID string) (domainenrollment.Enrollment, error) {
	entity, err := domainenrollment.New(newID(), strings.TrimSpace(studentID), strings.TrimSpace(institutionID))
	if err != nil {
		return domainenrollment.Enrollment{}, err
	}
	if err := service.repository.Create(ctx, entity); err != nil {
		return domainenrollment.Enrollment{}, err
	}
	return entity, nil
}

func (service Service) List(ctx context.Context) ([]domainenrollment.Enrollment, error) {
	return service.repository.List(ctx)
}

func (service Service) FindByID(ctx context.Context, studentID string, enrollmentID string) (domainenrollment.Enrollment, bool, error) {
	return service.repository.FindByID(ctx, studentID, enrollmentID)
}

func (service Service) Transfer(ctx context.Context, studentID string, enrollmentID string, institutionID string) (domainenrollment.Enrollment, error) {
	if err := domainenrollment.ValidateTransfer(studentID, enrollmentID, institutionID); err != nil {
		return domainenrollment.Enrollment{}, err
	}

	destination, err := domainenrollment.New(newID(), studentID, institutionID)
	if err != nil {
		return domainenrollment.Enrollment{}, err
	}
	if err := service.repository.Transfer(ctx, studentID, enrollmentID, destination); err != nil {
		return domainenrollment.Enrollment{}, err
	}
	return destination, nil
}

func (service Service) Suspend(ctx context.Context, studentID string, enrollmentID string, reason string, suspendedAt time.Time) error {
	if err := domainenrollment.ValidateSuspension(studentID, enrollmentID, reason); err != nil {
		return err
	}
	if suspendedAt.IsZero() {
		return domainenrollment.ErrSuspensionDateRequired
	}
	return service.repository.Suspend(ctx, studentID, enrollmentID, strings.TrimSpace(reason), suspendedAt)
}

func (service Service) Reopen(ctx context.Context, studentID string, enrollmentID string) error {
	if strings.TrimSpace(studentID) == "" {
		return domainenrollment.ErrStudentIDRequired
	}
	if strings.TrimSpace(enrollmentID) == "" {
		return domainenrollment.ErrEnrollmentIDRequired
	}
	return service.repository.Reopen(ctx, studentID, enrollmentID)
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
