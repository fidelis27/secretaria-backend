package mariadb

import (
	"context"
	"database/sql"
	"fmt"

	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
)

type MemberGroupRepository struct {
	connection *Connection
}

func NewMemberGroupRepository(connection *Connection) MemberGroupRepository {
	return MemberGroupRepository{connection: connection}
}

func (repository MemberGroupRepository) ListInstitutionIDsByUser(ctx context.Context, userID string) ([]string, error) {
	query := `SELECT DISTINCT g.institution_id FROM member_groups mg INNER JOIN ` + groupsTable + ` g ON g.id = mg.group_id WHERE mg.user_id = ?`
	rows, err := repository.connection.database.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list visible institutions: %w", err)
	}
	defer rows.Close()
	result := make([]string, 0)
	for rows.Next() {
		var institutionID string
		if err := rows.Scan(&institutionID); err != nil {
			return nil, fmt.Errorf("scan visible institution: %w", err)
		}
		result = append(result, institutionID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate visible institutions: %w", err)
	}
	return result, nil
}

func (repository MemberGroupRepository) FindByUserAndInstitution(ctx context.Context, userID string, institutionID string) ([]domainauthorization.Membership, error) {
	query := `SELECT mg.user_id, mg.group_id, g.institution_id, mg.role
FROM member_groups mg INNER JOIN ` + groupsTable + ` g ON g.id = mg.group_id
WHERE mg.user_id = ? AND g.institution_id = ?`
	rows, err := repository.connection.database.QueryContext(ctx, query, userID, institutionID)
	if err != nil {
		return nil, fmt.Errorf("find memberships by institution: %w", err)
	}
	defer rows.Close()

	result := make([]domainauthorization.Membership, 0)
	for rows.Next() {
		var membership domainauthorization.Membership
		if err := rows.Scan(&membership.UserID, &membership.GroupID, &membership.InstitutionID, &membership.Role); err != nil {
			return nil, fmt.Errorf("scan institution membership: %w", err)
		}
		result = append(result, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate institution memberships: %w", err)
	}
	return result, nil
}

func (repository MemberGroupRepository) FindByUserAndGroup(ctx context.Context, userID string, groupID string) (domainauthorization.Membership, bool, error) {
	var membership domainauthorization.Membership
	query := `SELECT mg.user_id, mg.group_id, g.institution_id, mg.role
FROM member_groups mg INNER JOIN ` + groupsTable + ` g ON g.id = mg.group_id
WHERE mg.user_id = ? AND mg.group_id = ?`
	err := repository.connection.database.QueryRowContext(ctx, query, userID, groupID).
		Scan(&membership.UserID, &membership.GroupID, &membership.InstitutionID, &membership.Role)
	if err == sql.ErrNoRows {
		return domainauthorization.Membership{}, false, nil
	}
	if err != nil {
		return domainauthorization.Membership{}, false, fmt.Errorf("find membership by group: %w", err)
	}
	return membership, true, nil
}
