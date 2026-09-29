package health

import (
	"context"

	apperrors "github.com/fidelis27/secretaria-backend/internal/platform/errors"
)

type Checker interface {
	Check(ctx context.Context) error
}

type Service struct {
	checker Checker
}

func NewService(checker Checker) Service {
	return Service{checker: checker}
}

func (service Service) Check(ctx context.Context) Status {
	if err := service.checker.Check(ctx); err != nil {
		return Status{OK: false, Message: apperrors.MessageFor(err)}
	}
	return Status{OK: true, Message: "database available"}
}

type Status struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}
