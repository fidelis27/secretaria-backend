package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

const bootstrapSuperAdminName = "Bootstrap Super Admin"

type Repository interface {
	CountSuperAdmins(ctx context.Context) (int, error)
	FindByEmail(ctx context.Context, email string) (domainuser.User, bool, error)
	Create(ctx context.Context, user domainuser.User) error
	PromoteByEmail(ctx context.Context, email string) error
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return Service{repository: repository}
}

func (service Service) EnsureFirstSuperAdmin(ctx context.Context, email string) (bool, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return false, nil
	}

	count, err := service.repository.CountSuperAdmins(ctx)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}

	if _, found, err := service.repository.FindByEmail(ctx, email); err != nil {
		return false, err
	} else if found {
		if err := service.repository.PromoteByEmail(ctx, email); err != nil {
			return false, err
		}
		return true, nil
	}

	user, err := domainuser.New(newID(), bootstrapSuperAdminName, email, true)
	if err != nil {
		return false, err
	}
	if err := service.repository.Create(ctx, user); err != nil {
		return false, err
	}
	return true, nil
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
