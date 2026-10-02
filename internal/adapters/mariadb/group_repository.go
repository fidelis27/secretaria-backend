package mariadb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"

	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
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
		"INSERT INTO "+groupsTable+" (id, institution_id, name) VALUES (?, ?, ?)", entity.ID, entity.InstitutionID, entity.Name,
	)
	if err != nil {
		return fmt.Errorf("create group: %w", err)
	}
	return nil
}

func (repository GroupRepository) List(ctx context.Context) ([]domaingroup.Group, error) {
	return repository.list(ctx, `SELECT g.id, g.institution_id, g.name, i.name
FROM `+groupsTable+` g
INNER JOIN institutions i ON i.id = g.institution_id
ORDER BY i.name, g.name, g.id`)
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
	return repository.list(ctx, `SELECT g.id, g.institution_id, g.name, i.name
FROM `+groupsTable+` g
INNER JOIN institutions i ON i.id = g.institution_id
WHERE g.institution_id IN (`+placeholders+`)
ORDER BY i.name, g.name, g.id`, args...)
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
		if err := rows.Scan(&entity.ID, &entity.InstitutionID, &entity.Name, &entity.InstitutionName); err != nil {
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
		`SELECT g.id, g.institution_id, g.name, i.name
FROM `+groupsTable+` g
INNER JOIN institutions i ON i.id = g.institution_id
WHERE g.id = ?`, id,
	).Scan(&entity.ID, &entity.InstitutionID, &entity.Name, &entity.InstitutionName)
	if err == sql.ErrNoRows {
		return domaingroup.Group{}, false, nil
	}
	if err != nil {
		return domaingroup.Group{}, false, fmt.Errorf("find group: %w", err)
	}
	return entity, true, nil
}

func (repository GroupRepository) AddMembership(ctx context.Context, membership domaingroup.Membership) (domaingroup.Membership, error) {
	_, err := repository.connection.database.ExecContext(ctx,
		`INSERT INTO member_groups (user_id, group_id, role) VALUES (?, ?, ?)`, membership.UserID, membership.GroupID, membership.Role,
	)
	if err != nil {
		return domaingroup.Membership{}, fmt.Errorf("create membership: %w", err)
	}
	created, err := repository.loadMembership(ctx, repository.connection.database, membership.GroupID, membership.UserID)
	if err != nil {
		return domaingroup.Membership{}, err
	}
	return created, nil
}

func (repository GroupRepository) ListMemberships(ctx context.Context, groupID string) ([]domaingroup.Membership, error) {
	query := `SELECT mg.user_id, mg.group_id, g.institution_id, mg.role, u.name, u.email
FROM member_groups mg
INNER JOIN ` + groupsTable + ` g ON g.id = mg.group_id
INNER JOIN users u ON u.id = mg.user_id
WHERE mg.group_id = ? ORDER BY u.name, mg.user_id`
	rows, err := repository.connection.database.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()

	result := make([]domaingroup.Membership, 0)
	for rows.Next() {
		var membership domaingroup.Membership
		if err := rows.Scan(&membership.UserID, &membership.GroupID, &membership.InstitutionID, &membership.Role, &membership.UserName, &membership.UserEmail); err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}
		result = append(result, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memberships: %w", err)
	}
	return result, nil
}

func (repository GroupRepository) UpdateMembershipRole(ctx context.Context, groupID string, userID string, role domainauthorization.Role) (domaingroup.Membership, error) {
	transaction, err := repository.connection.database.BeginTx(ctx, nil)
	if err != nil {
		return domaingroup.Membership{}, fmt.Errorf("begin update membership: %w", err)
	}
	defer transaction.Rollback()

	currentRole, err := repository.loadMembershipRole(ctx, transaction, groupID, userID)
	if err != nil {
		return domaingroup.Membership{}, err
	}
	if currentRole == domainauthorization.RoleAdmin && role != domainauthorization.RoleAdmin {
		ok, err := repository.hasAnotherAdmin(ctx, transaction, groupID, userID)
		if err != nil {
			return domaingroup.Membership{}, err
		}
		if !ok {
			return domaingroup.Membership{}, domaingroup.ErrLastAdminRequired
		}
	}

	if _, err := transaction.ExecContext(ctx,
		`UPDATE member_groups SET role = ? WHERE group_id = ? AND user_id = ?`,
		role, groupID, userID,
	); err != nil {
		return domaingroup.Membership{}, fmt.Errorf("update membership role: %w", err)
	}

	updated, err := repository.loadMembership(ctx, transaction, groupID, userID)
	if err != nil {
		return domaingroup.Membership{}, err
	}
	if err := transaction.Commit(); err != nil {
		return domaingroup.Membership{}, fmt.Errorf("commit membership role update: %w", err)
	}
	return updated, nil
}

func (repository GroupRepository) RemoveMembership(ctx context.Context, groupID string, userID string) error {
	transaction, err := repository.connection.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin remove membership: %w", err)
	}
	defer transaction.Rollback()

	currentRole, err := repository.loadMembershipRole(ctx, transaction, groupID, userID)
	if err != nil {
		return err
	}
	if currentRole == domainauthorization.RoleAdmin {
		ok, err := repository.hasAnotherAdmin(ctx, transaction, groupID, userID)
		if err != nil {
			return err
		}
		if !ok {
			return domaingroup.ErrLastAdminRequired
		}
	}

	result, err := transaction.ExecContext(ctx,
		`DELETE FROM member_groups WHERE group_id = ? AND user_id = ?`,
		groupID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete membership: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted membership: %w", err)
	}
	if rowsAffected != 1 {
		return domaingroup.ErrMembershipNotFound
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit membership removal: %w", err)
	}
	return nil
}

func (repository GroupRepository) ListCandidates(ctx context.Context, groupID string) ([]domaingroup.Candidate, error) {
	rows, err := repository.connection.database.QueryContext(ctx, `SELECT u.id, u.name, u.email
FROM users u
WHERE u.status = 'active'
AND NOT EXISTS (
	SELECT 1 FROM member_groups mg WHERE mg.group_id = ? AND mg.user_id = u.id
)
ORDER BY u.name, u.id`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group candidates: %w", err)
	}
	defer rows.Close()

	result := make([]domaingroup.Candidate, 0)
	for rows.Next() {
		var candidate domaingroup.Candidate
		if err := rows.Scan(&candidate.ID, &candidate.Name, &candidate.Email); err != nil {
			return nil, fmt.Errorf("scan group candidate: %w", err)
		}
		result = append(result, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate group candidates: %w", err)
	}
	return result, nil
}

type membershipQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (repository GroupRepository) loadMembership(ctx context.Context, querier membershipQuerier, groupID string, userID string) (domaingroup.Membership, error) {
	var membership domaingroup.Membership
	err := querier.QueryRowContext(ctx, `SELECT mg.user_id, mg.group_id, g.institution_id, mg.role, u.name, u.email
FROM member_groups mg
INNER JOIN `+groupsTable+` g ON g.id = mg.group_id
INNER JOIN users u ON u.id = mg.user_id
WHERE mg.group_id = ? AND mg.user_id = ?`, groupID, userID).
		Scan(&membership.UserID, &membership.GroupID, &membership.InstitutionID, &membership.Role, &membership.UserName, &membership.UserEmail)
	if err == sql.ErrNoRows {
		return domaingroup.Membership{}, domaingroup.ErrMembershipNotFound
	}
	if err != nil {
		return domaingroup.Membership{}, fmt.Errorf("load membership: %w", err)
	}
	return membership, nil
}

func (repository GroupRepository) loadMembershipRole(ctx context.Context, transaction *sql.Tx, groupID string, userID string) (domainauthorization.Role, error) {
	var role domainauthorization.Role
	err := transaction.QueryRowContext(ctx,
		`SELECT role FROM member_groups WHERE group_id = ? AND user_id = ? FOR UPDATE`,
		groupID, userID,
	).Scan(&role)
	if err == sql.ErrNoRows {
		return "", domaingroup.ErrMembershipNotFound
	}
	if err != nil {
		return "", fmt.Errorf("load membership role: %w", err)
	}
	return role, nil
}

func (repository GroupRepository) hasAnotherAdmin(ctx context.Context, transaction *sql.Tx, groupID string, excludedUserID string) (bool, error) {
	var count int
	if err := transaction.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM member_groups WHERE group_id = ? AND role = 'admin' AND user_id <> ?`,
		groupID, excludedUserID,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("count group admins: %w", err)
	}
	return count > 0, nil
}

func isDuplicateGroupName(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 && strings.Contains(strings.ToLower(err.Error()), "institution_id")
}

var _ domaingroup.Repository = GroupRepository{}
