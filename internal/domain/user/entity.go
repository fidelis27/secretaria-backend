package user

import (
	"errors"
	"net/mail"
	"strings"
)

var ErrNameRequired = errors.New("user name is required")
var ErrEmailRequired = errors.New("user email is required")
var ErrEmailInvalid = errors.New("user email is invalid")

type User struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	AuthSub    string `json:"-"`
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
	parsedEmail, err := mail.ParseAddress(trimmedEmail)
	if err != nil || parsedEmail.Address != trimmedEmail {
		return User{}, ErrEmailInvalid
	}

	return User{
		ID:         id,
		Name:       trimmedName,
		Email:      strings.ToLower(parsedEmail.Address),
		Status:     "active",
		SuperAdmin: superAdmin,
	}, nil
}
