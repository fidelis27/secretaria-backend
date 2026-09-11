package health

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Check() Status {
	return Status{OK: true}
}

type Status struct {
	OK bool `json:"ok"`
}
