package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
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

func (service Service) FindByID(ctx context.Context, id string) (domainuser.User, bool, error) {
	return service.repository.FindByID(ctx, id)
}

func (service Service) FindByEmail(ctx context.Context, email string) (domainuser.User, bool, error) {
	return service.repository.FindByEmail(ctx, strings.TrimSpace(email))
}

func (service Service) FindByAuthSub(ctx context.Context, authSub string) (domainuser.User, bool, error) {
	return service.repository.FindByAuthSub(ctx, strings.TrimSpace(authSub))
}

func (service Service) LinkAuthSub(ctx context.Context, userID string, authSub string) error {
	return service.repository.LinkAuthSub(ctx, strings.TrimSpace(userID), strings.TrimSpace(authSub))
}

func (service Service) Update(ctx context.Context, userID string, name *string, status *string) (domainuser.User, error) {
	entity, found, err := service.repository.FindByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		return domainuser.User{}, err
	}
	if !found {
		return domainuser.User{}, nil
	}
	if name != nil {
		normalizedName, err := domainuser.NormalizeName(*name)
		if err != nil {
			return domainuser.User{}, err
		}
		entity.Name = normalizedName
	}
	if status != nil {
		normalizedStatus, err := domainuser.NormalizeStatus(*status)
		if err != nil {
			return domainuser.User{}, err
		}
		entity.Status = normalizedStatus
	}
	if err := service.repository.Update(ctx, entity); err != nil {
		return domainuser.User{}, err
	}
	return entity, nil
}

func (service Service) CountActiveSuperAdmins(ctx context.Context) (int, error) {
	return service.repository.CountActiveSuperAdmins(ctx)
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
