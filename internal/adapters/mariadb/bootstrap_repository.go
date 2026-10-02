package mariadb

import (
	"context"
	"database/sql"
	"fmt"

	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type BootstrapRepository struct {
	connection *Connection
}

func NewBootstrapRepository(connection *Connection) BootstrapRepository {
	return BootstrapRepository{connection: connection}
}

func (repository BootstrapRepository) CountSuperAdmins(ctx context.Context) (int, error) {
	var count int
	if err := repository.connection.database.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE super_admin = TRUE`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count superadmins: %w", err)
	}
	return count, nil
}

func (repository BootstrapRepository) FindByEmail(ctx context.Context, email string) (domainuser.User, bool, error) {
	return NewUserRepository(repository.connection).FindByEmail(ctx, email)
}

func (repository BootstrapRepository) Create(ctx context.Context, user domainuser.User) error {
	return NewUserRepository(repository.connection).Create(ctx, user)
}

func (repository BootstrapRepository) PromoteByEmail(ctx context.Context, email string) error {
	result, err := repository.connection.database.ExecContext(ctx,
		`UPDATE users SET super_admin = TRUE, status = 'active' WHERE email = ?`,
		email,
	)
	if err != nil {
		return fmt.Errorf("promote bootstrap superadmin: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check bootstrap promotion: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
