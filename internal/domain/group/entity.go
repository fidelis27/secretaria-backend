package group

import (
	"errors"

	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
)

var ErrInstitutionIDRequired = errors.New("group institution id is required")
var ErrUserIDRequired = errors.New("membership user id is required")
var ErrGroupIDRequired = errors.New("membership group id is required")
var ErrInvalidRole = errors.New("membership role is invalid")

type Group struct {
	ID            string `json:"id"`
	InstitutionID string `json:"institutionId"`
}

type Membership struct {
	UserID        string                   `json:"userId"`
	GroupID       string                   `json:"groupId"`
	InstitutionID string                   `json:"institutionId"`
	Role          domainauthorization.Role `json:"role"`
}

func New(id string, institutionID string) (Group, error) {
	if institutionID == "" {
		return Group{}, ErrInstitutionIDRequired
	}
	return Group{ID: id, InstitutionID: institutionID}, nil
}

func NewMembership(userID string, groupID string, institutionID string, role domainauthorization.Role) (Membership, error) {
	if userID == "" {
		return Membership{}, ErrUserIDRequired
	}
	if groupID == "" {
		return Membership{}, ErrGroupIDRequired
	}
	if role != domainauthorization.RoleAdmin && role != domainauthorization.RoleMember {
		return Membership{}, ErrInvalidRole
	}
	return Membership{UserID: userID, GroupID: groupID, InstitutionID: institutionID, Role: role}, nil
}
