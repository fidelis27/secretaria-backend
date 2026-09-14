package institution

import (
	"context"
	"testing"

	"github.com/karol/secretaria-escolar-backend/internal/domain/institution"
)

type repositoryStub struct {
	created []institution.Institution
}

func (stub *repositoryStub) Create(_ context.Context, entity institution.Institution) error {
	stub.created = append(stub.created, entity)
	return nil
}

func (*repositoryStub) List(context.Context) ([]institution.Institution, error) {
	return nil, nil
}

func TestServiceCreatesActiveInstitution(t *testing.T) {
	repository := &repositoryStub{}
	created, err := NewService(repository).Create(context.Background(), "  Fatec Campinas  ", " 123 ")
	if err != nil {
		t.Fatalf("create institution: %v", err)
	}
	if created.Name != "Fatec Campinas" || created.CNPJ != "123" || created.Status != "active" {
		t.Fatalf("unexpected institution: %+v", created)
	}
	if len(repository.created) != 1 {
		t.Fatalf("created institutions = %d, want 1", len(repository.created))
	}
}

func TestServiceRejectsBlankName(t *testing.T) {
	_, err := NewService(&repositoryStub{}).Create(context.Background(), "  ", "")
	if err != institution.ErrNameRequired {
		t.Fatalf("error = %v, want %v", err, institution.ErrNameRequired)
	}
}
