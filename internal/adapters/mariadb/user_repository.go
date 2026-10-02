package mariadb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type UserRepository struct {
	connection *Connection
}

func NewUserRepository(connection *Connection) UserRepository {
	return UserRepository{connection: connection}
}

func (repository UserRepository) Create(ctx context.Context, entity domainuser.User) error {
	_, err := repository.connection.database.ExecContext(ctx,
		`INSERT INTO users (id, name, email, auth_sub, status, super_admin) VALUES (?, ?, ?, ?, ?, ?)`,
		entity.ID, entity.Name, entity.Email, nullableString(entity.AuthSub), entity.Status, entity.SuperAdmin,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (repository UserRepository) List(ctx context.Context) ([]domainuser.User, error) {
	rows, err := repository.connection.database.QueryContext(ctx,
		`SELECT id, name, email, auth_sub, status, super_admin FROM users ORDER BY name, id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	result := make([]domainuser.User, 0)
	for rows.Next() {
		var entity domainuser.User
		var authSub sql.NullString
		if err := rows.Scan(&entity.ID, &entity.Name, &entity.Email, &authSub, &entity.Status, &entity.SuperAdmin); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		entity.AuthSub = authSub.String
		result = append(result, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return result, nil
}

func (repository UserRepository) FindByID(ctx context.Context, id string) (domainuser.User, bool, error) {
	var entity domainuser.User
	var authSub sql.NullString
	err := repository.connection.database.QueryRowContext(ctx,
		`SELECT id, name, email, auth_sub, status, super_admin FROM users WHERE id = ?`, id,
	).Scan(&entity.ID, &entity.Name, &entity.Email, &authSub, &entity.Status, &entity.SuperAdmin)
	if err == sql.ErrNoRows {
		return domainuser.User{}, false, nil
	}
	if err != nil {
		return domainuser.User{}, false, fmt.Errorf("find user by id: %w", err)
	}
	entity.AuthSub = authSub.String
	return entity, true, nil
}

func (repository UserRepository) FindByEmail(ctx context.Context, email string) (domainuser.User, bool, error) {
	var entity domainuser.User
	var authSub sql.NullString
	err := repository.connection.database.QueryRowContext(ctx,
		`SELECT id, name, email, auth_sub, status, super_admin FROM users WHERE email = ?`, email,
	).Scan(&entity.ID, &entity.Name, &entity.Email, &authSub, &entity.Status, &entity.SuperAdmin)
	if err == sql.ErrNoRows {
		return domainuser.User{}, false, nil
	}
	if err != nil {
		return domainuser.User{}, false, fmt.Errorf("find user by email: %w", err)
	}
	entity.AuthSub = authSub.String
	return entity, true, nil
}

func (repository UserRepository) FindByAuthSub(ctx context.Context, authSub string) (domainuser.User, bool, error) {
	var entity domainuser.User
	var authValue sql.NullString
	err := repository.connection.database.QueryRowContext(ctx,
		`SELECT id, name, email, auth_sub, status, super_admin FROM users WHERE auth_sub = ?`, authSub,
	).Scan(&entity.ID, &entity.Name, &entity.Email, &authValue, &entity.Status, &entity.SuperAdmin)
	if err == sql.ErrNoRows {
		return domainuser.User{}, false, nil
	}
	if err != nil {
		return domainuser.User{}, false, fmt.Errorf("find user by auth sub: %w", err)
	}
	entity.AuthSub = authValue.String
	return entity, true, nil
}

func (repository UserRepository) LinkAuthSub(ctx context.Context, userID string, authSub string) error {
	_, err := repository.connection.database.ExecContext(ctx,
		`UPDATE users SET auth_sub = ? WHERE id = ? AND (auth_sub IS NULL OR auth_sub = '')`,
		authSub, userID,
	)
	if err != nil {
		return fmt.Errorf("link user auth sub: %w", err)
	}
	return nil
}

func nullableString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
