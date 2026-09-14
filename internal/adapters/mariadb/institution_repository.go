package mariadb

import (
	"context"
	"fmt"

	"github.com/karol/secretaria-escolar-backend/internal/domain/institution"
)

type InstitutionRepository struct {
	connection *Connection
}

func NewInstitutionRepository(connection *Connection) InstitutionRepository {
	return InstitutionRepository{connection: connection}
}

func (repository InstitutionRepository) Create(ctx context.Context, entity institution.Institution) error {
	_, err := repository.connection.database.ExecContext(ctx,
		`INSERT INTO institutions (id, name, cnpj, status) VALUES (?, ?, NULLIF(?, ''), ?)`,
		entity.ID, entity.Name, entity.CNPJ, entity.Status,
	)
	if err != nil {
		return fmt.Errorf("create institution: %w", err)
	}
	return nil
}

func (repository InstitutionRepository) List(ctx context.Context) ([]institution.Institution, error) {
	rows, err := repository.connection.database.QueryContext(ctx,
		`SELECT id, name, COALESCE(cnpj, ''), status FROM institutions ORDER BY name, id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list institutions: %w", err)
	}
	defer rows.Close()

	result := make([]institution.Institution, 0)
	for rows.Next() {
		var entity institution.Institution
		if err := rows.Scan(&entity.ID, &entity.Name, &entity.CNPJ, &entity.Status); err != nil {
			return nil, fmt.Errorf("scan institution: %w", err)
		}
		result = append(result, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate institutions: %w", err)
	}
	return result, nil
}
