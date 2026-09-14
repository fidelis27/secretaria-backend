package student

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	domainstudent "github.com/fidelis27/secretaria-backend/internal/domain/student"
)

type Service struct {
	repository domainstudent.Repository
}

func NewService(repository domainstudent.Repository) Service {
	return Service{repository: repository}
}

func (service Service) Create(ctx context.Context, name string, institutionID string) (domainstudent.Student, error) {
	entity, err := domainstudent.New(newID(), strings.TrimSpace(name), strings.TrimSpace(institutionID))
	if err != nil {
		return domainstudent.Student{}, err
	}
	if err := service.repository.Create(ctx, entity); err != nil {
		return domainstudent.Student{}, err
	}
	return entity, nil
}

func (service Service) List(ctx context.Context) ([]domainstudent.Student, error) {
	return service.repository.List(ctx)
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
