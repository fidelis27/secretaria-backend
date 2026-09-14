package user

import (
	"errors"
	"strings"
)

var ErrNameRequired = errors.New("user name is required")
var ErrEmailRequired = errors.New("user email is required")

type User struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Status     string `json:"status"`
	SuperAdmin bool   `json:"superAdmin"`
}

func New(id string, name string, email string, superAdmin bool) (User, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return User{}, ErrNameRequired
	}

	trimmedEmail := strings.TrimSpace(email)
	if trimmedEmail == "" {
		return User{}, ErrEmailRequired
	}

	return User{
		ID:         id,
		Name:       trimmedName,
		Email:      strings.ToLower(trimmedEmail),
		Status:     "active",
		SuperAdmin: superAdmin,
	}, nil
}
