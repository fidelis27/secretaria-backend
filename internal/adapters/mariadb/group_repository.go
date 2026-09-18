package mariadb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	domaingroup "github.com/fidelis27/secretaria-backend/internal/domain/group"
)

type GroupRepository struct {
	connection *Connection
}

const groupsTable = "`groups`"

func NewGroupRepository(connection *Connection) GroupRepository {
	return GroupRepository{connection: connection}
}

func (repository GroupRepository) Create(ctx context.Context, entity domaingroup.Group) error {
	_, err := repository.connection.database.ExecContext(ctx,
		"INSERT INTO "+groupsTable+" (id, institution_id) VALUES (?, ?)", entity.ID, entity.InstitutionID,
	)
	if err != nil {
		return fmt.Errorf("create group: %w", err)
	}
	return nil
}

func (repository GroupRepository) List(ctx context.Context) ([]domaingroup.Group, error) {
	return repository.list(ctx, "SELECT id, institution_id FROM "+groupsTable+" ORDER BY id")
}

func (repository GroupRepository) ListByInstitutionIDs(ctx context.Context, institutionIDs []string) ([]domaingroup.Group, error) {
	if len(institutionIDs) == 0 {
		return []domaingroup.Group{}, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(institutionIDs)), ",")
	args := make([]any, len(institutionIDs))
	for index, institutionID := range institutionIDs {
		args[index] = institutionID
	}
	return repository.list(ctx, "SELECT id, institution_id FROM "+groupsTable+" WHERE institution_id IN ("+placeholders+") ORDER BY id", args...)
}

func (repository GroupRepository) list(ctx context.Context, query string, args ...any) ([]domaingroup.Group, error) {
	rows, err := repository.connection.database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	result := make([]domaingroup.Group, 0)
	for rows.Next() {
		var entity domaingroup.Group
		if err := rows.Scan(&entity.ID, &entity.InstitutionID); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		result = append(result, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate groups: %w", err)
	}
	return result, nil
}

func (repository GroupRepository) FindByID(ctx context.Context, id string) (domaingroup.Group, bool, error) {
	var entity domaingroup.Group
	err := repository.connection.database.QueryRowContext(ctx,
		"SELECT id, institution_id FROM "+groupsTable+" WHERE id = ?", id,
	).Scan(&entity.ID, &entity.InstitutionID)
	if err == sql.ErrNoRows {
		return domaingroup.Group{}, false, nil
	}
	if err != nil {
		return domaingroup.Group{}, false, fmt.Errorf("find group: %w", err)
	}
	return entity, true, nil
}

func (repository GroupRepository) AddMembership(ctx context.Context, membership domaingroup.Membership) error {
	_, err := repository.connection.database.ExecContext(ctx,
		`INSERT INTO member_groups (user_id, group_id, role) VALUES (?, ?, ?)`, membership.UserID, membership.GroupID, membership.Role,
	)
	if err != nil {
		return fmt.Errorf("create membership: %w", err)
	}
	return nil
}

func (repository GroupRepository) ListMemberships(ctx context.Context, groupID string) ([]domaingroup.Membership, error) {
	query := `SELECT mg.user_id, mg.group_id, g.institution_id, mg.role
FROM member_groups mg INNER JOIN ` + groupsTable + ` g ON g.id = mg.group_id
WHERE mg.group_id = ? ORDER BY mg.user_id`
	rows, err := repository.connection.database.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()

	result := make([]domaingroup.Membership, 0)
	for rows.Next() {
		var membership domaingroup.Membership
		if err := rows.Scan(&membership.UserID, &membership.GroupID, &membership.InstitutionID, &membership.Role); err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}
		result = append(result, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memberships: %w", err)
	}
	return result, nil
}

var _ domaingroup.Repository = GroupRepository{}
