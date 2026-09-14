package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	domainuser "github.com/karol/secretaria-escolar-backend/internal/domain/user"
)

type Service struct {
	repository domainuser.Repository
}

func NewService(repository domainuser.Repository) Service {
	return Service{repository: repository}
}

func (service Service) Create(ctx context.Context, name string, email string, superAdmin bool) (domainuser.User, error) {
	entity, err := domainuser.New(newID(), strings.TrimSpace(name), strings.TrimSpace(email), superAdmin)
	if err != nil {
		return domainuser.User{}, err
	}
	if err := service.repository.Create(ctx, entity); err != nil {
		return domainuser.User{}, err
	}
	return entity, nil
}

func (service Service) List(ctx context.Context) ([]domainuser.User, error) {
	return service.repository.List(ctx)
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
