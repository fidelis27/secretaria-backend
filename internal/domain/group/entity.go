package group

import (
	"errors"
	"strings"

	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
)

var ErrInstitutionIDRequired = errors.New("group institution id is required")
var ErrNameRequired = errors.New("group name is required")
var ErrUserIDRequired = errors.New("membership user id is required")
var ErrGroupIDRequired = errors.New("membership group id is required")
var ErrInvalidRole = errors.New("membership role is invalid")
var ErrMembershipNotFound = errors.New("membership not found")
var ErrLastAdminRequired = errors.New("o grupo deve manter ao menos um administrador")

type Group struct {
	ID              string `json:"id"`
	InstitutionID   string `json:"institutionId"`
	Name            string `json:"name"`
	InstitutionName string `json:"institutionName,omitempty"`
}

type Membership struct {
	UserID        string                   `json:"userId"`
	GroupID       string                   `json:"groupId"`
	InstitutionID string                   `json:"institutionId"`
	Role          domainauthorization.Role `json:"role"`
	UserName      string                   `json:"userName,omitempty"`
	UserEmail     string                   `json:"userEmail,omitempty"`
}

type Candidate struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func New(id string, institutionID string, name string) (Group, error) {
	if institutionID == "" {
		return Group{}, ErrInstitutionIDRequired
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Group{}, ErrNameRequired
	}
	return Group{ID: id, InstitutionID: institutionID, Name: name}, nil
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
