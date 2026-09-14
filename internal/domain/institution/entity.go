package institution

import "errors"

var ErrNameRequired = errors.New("institution name is required")

type Institution struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	CNPJ   string `json:"cnpj,omitempty"`
	Status string `json:"status"`
}

func New(id string, name string, cnpj string) (Institution, error) {
	if name == "" {
		return Institution{}, ErrNameRequired
	}

	return Institution{ID: id, Name: name, CNPJ: cnpj, Status: "active"}, nil
}
