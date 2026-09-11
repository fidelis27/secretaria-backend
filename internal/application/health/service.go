package health

import "context"

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
		return Status{OK: false}
	}
	return Status{OK: true}
}

type Status struct {
	OK bool `json:"ok"`
}
