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

func (service Service) Create(ctx context.Context, institutionID string, name string) (domaingroup.Group, error) {
	entity, err := domaingroup.New(newID(), strings.TrimSpace(institutionID), strings.TrimSpace(name))
	if err != nil {
		return domaingroup.Group{}, err
	}
	if err := service.repository.Create(ctx, entity); err != nil {
		return domaingroup.Group{}, err
	}
	created, _, err := service.repository.FindByID(ctx, entity.ID)
	if err != nil {
		return domaingroup.Group{}, err
	}
	return created, nil
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
	created, err := service.repository.AddMembership(ctx, entity)
	if err != nil {
		return domaingroup.Membership{}, err
	}
	return created, nil
}

func (service Service) UpdateMembershipRole(ctx context.Context, groupID string, userID string, role domainauthorization.Role) (domaingroup.Membership, error) {
	if strings.TrimSpace(userID) == "" {
		return domaingroup.Membership{}, domaingroup.ErrUserIDRequired
	}
	if strings.TrimSpace(groupID) == "" {
		return domaingroup.Membership{}, domaingroup.ErrGroupIDRequired
	}
	if role != domainauthorization.RoleAdmin && role != domainauthorization.RoleMember {
		return domaingroup.Membership{}, domaingroup.ErrInvalidRole
	}
	return service.repository.UpdateMembershipRole(ctx, strings.TrimSpace(groupID), strings.TrimSpace(userID), role)
}

func (service Service) RemoveMembership(ctx context.Context, groupID string, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return domaingroup.ErrUserIDRequired
	}
	if strings.TrimSpace(groupID) == "" {
		return domaingroup.ErrGroupIDRequired
	}
	return service.repository.RemoveMembership(ctx, strings.TrimSpace(groupID), strings.TrimSpace(userID))
}

func (service Service) ListMemberships(ctx context.Context, groupID string) ([]domaingroup.Membership, error) {
	return service.repository.ListMemberships(ctx, groupID)
}

func (service Service) ListCandidates(ctx context.Context, groupID string) ([]domaingroup.Candidate, error) {
	if strings.TrimSpace(groupID) == "" {
		return nil, domaingroup.ErrGroupIDRequired
	}
	return service.repository.ListCandidates(ctx, strings.TrimSpace(groupID))
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
