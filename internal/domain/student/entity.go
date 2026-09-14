package student

import (
	"errors"
	"strings"
)

var ErrNameRequired = errors.New("student name is required")
var ErrInstitutionIDRequired = errors.New("student institution id is required")

type Student struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	InstitutionID string `json:"institutionId"`
	Status        string `json:"status"`
}

func New(id string, name string, institutionID string) (Student, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Student{}, ErrNameRequired
	}
	trimmedInstitutionID := strings.TrimSpace(institutionID)
	if trimmedInstitutionID == "" {
		return Student{}, ErrInstitutionIDRequired
	}

	return Student{
		ID:            id,
		Name:          trimmedName,
		InstitutionID: trimmedInstitutionID,
		Status:        "active",
	}, nil
}
