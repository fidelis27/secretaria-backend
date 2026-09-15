package group

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domaingroup "github.com/fidelis27/secretaria-backend/internal/domain/group"
)

type Service struct {
	repository domaingroup.Repository
}

func NewService(repository domaingroup.Repository) Service {
	return Service{repository: repository}
}

func (service Service) Create(ctx context.Context, institutionID string) (domaingroup.Group, error) {
	entity, err := domaingroup.New(newID(), strings.TrimSpace(institutionID))
	if err != nil {
		return domaingroup.Group{}, err
	}
	if err := service.repository.Create(ctx, entity); err != nil {
		return domaingroup.Group{}, err
	}
	return entity, nil
}

func (service Service) List(ctx context.Context) ([]domaingroup.Group, error) {
	return service.repository.List(ctx)
}

func (service Service) ListByInstitutionIDs(ctx context.Context, institutionIDs []string) ([]domaingroup.Group, error) {
	return service.repository.ListByInstitutionIDs(ctx, institutionIDs)
}

func (service Service) FindByID(ctx context.Context, id string) (domaingroup.Group, bool, error) {
	return service.repository.FindByID(ctx, id)
}

func (service Service) AddMembership(ctx context.Context, userID string, groupID string, institutionID string, role domainauthorization.Role) (domaingroup.Membership, error) {
	entity, err := domaingroup.NewMembership(strings.TrimSpace(userID), strings.TrimSpace(groupID), strings.TrimSpace(institutionID), role)
	if err != nil {
		return domaingroup.Membership{}, err
	}
	if err := service.repository.AddMembership(ctx, entity); err != nil {
		return domaingroup.Membership{}, err
	}
	return entity, nil
}

func (service Service) ListMemberships(ctx context.Context, groupID string) ([]domaingroup.Membership, error) {
	return service.repository.ListMemberships(ctx, groupID)
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
