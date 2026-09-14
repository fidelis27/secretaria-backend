package enrollment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	domainenrollment "github.com/karol/secretaria-escolar-backend/internal/domain/enrollment"
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

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
