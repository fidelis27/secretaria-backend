package mariadb

import (
	"context"
	"fmt"

	domainuser "github.com/karol/secretaria-escolar-backend/internal/domain/user"
)

type UserRepository struct {
	connection *Connection
}

func NewUserRepository(connection *Connection) UserRepository {
	return UserRepository{connection: connection}
}

func (repository UserRepository) Create(ctx context.Context, entity domainuser.User) error {
	_, err := repository.connection.database.ExecContext(ctx,
		`INSERT INTO users (id, name, email, status, super_admin) VALUES (?, ?, ?, ?, ?)`,
		entity.ID, entity.Name, entity.Email, entity.Status, entity.SuperAdmin,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (repository UserRepository) List(ctx context.Context) ([]domainuser.User, error) {
	rows, err := repository.connection.database.QueryContext(ctx,
		`SELECT id, name, email, status, super_admin FROM users ORDER BY name, id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	result := make([]domainuser.User, 0)
	for rows.Next() {
		var entity domainuser.User
		if err := rows.Scan(&entity.ID, &entity.Name, &entity.Email, &entity.Status, &entity.SuperAdmin); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		result = append(result, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return result, nil
}
