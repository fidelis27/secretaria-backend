package institution

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/fidelis27/secretaria-backend/internal/domain/institution"
)

type Service struct {
	repository institution.Repository
}

func NewService(repository institution.Repository) Service {
	return Service{repository: repository}
}

func (service Service) Create(ctx context.Context, name string, cnpj string) (institution.Institution, error) {
	entity, err := institution.New(newID(), strings.TrimSpace(name), strings.TrimSpace(cnpj))
	if err != nil {
		return institution.Institution{}, err
	}
	if err := service.repository.Create(ctx, entity); err != nil {
		return institution.Institution{}, err
	}
	return entity, nil
}

func (service Service) List(ctx context.Context) ([]institution.Institution, error) {
	return service.repository.List(ctx)
}

func (service Service) ListByIDs(ctx context.Context, ids []string) ([]institution.Institution, error) {
	return service.repository.ListByIDs(ctx, ids)
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
