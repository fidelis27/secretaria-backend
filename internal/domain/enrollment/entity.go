package enrollment

import (
	"errors"
	"strings"
)

var ErrStudentIDRequired = errors.New("student id is required")
var ErrInstitutionIDRequired = errors.New("institution id is required")

type Enrollment struct {
	ID            string `json:"id"`
	StudentID     string `json:"studentId"`
	InstitutionID string `json:"institutionId"`
	Status        string `json:"status"`
}

func New(id string, studentID string, institutionID string) (Enrollment, error) {
	trimmedStudentID := strings.TrimSpace(studentID)
	if trimmedStudentID == "" {
		return Enrollment{}, ErrStudentIDRequired
	}
	trimmedInstitutionID := strings.TrimSpace(institutionID)
	if trimmedInstitutionID == "" {
		return Enrollment{}, ErrInstitutionIDRequired
	}

	return Enrollment{
		ID:            id,
		StudentID:     trimmedStudentID,
		InstitutionID: trimmedInstitutionID,
		Status:        "active",
	}, nil
}
