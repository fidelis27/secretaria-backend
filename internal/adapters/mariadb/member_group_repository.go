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

func (repository MemberGroupRepository) FindByUserAndInstitution(ctx context.Context, userID string, institutionID string) ([]domainauthorization.Membership, error) {
	rows, err := repository.connection.database.QueryContext(ctx,
		`SELECT mg.user_id, mg.group_id, g.institution_id, mg.role
		 FROM member_groups mg
		 INNER JOIN groups g ON g.id = mg.group_id
		 WHERE mg.user_id = ? AND g.institution_id = ?`, userID, institutionID,
	)
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
	err := repository.connection.database.QueryRowContext(ctx,
		`SELECT mg.user_id, mg.group_id, g.institution_id, mg.role
		 FROM member_groups mg
		 INNER JOIN groups g ON g.id = mg.group_id
		 WHERE mg.user_id = ? AND mg.group_id = ?`, userID, groupID,
	).Scan(&membership.UserID, &membership.GroupID, &membership.InstitutionID, &membership.Role)
	if err == sql.ErrNoRows {
		return domainauthorization.Membership{}, false, nil
	}
	if err != nil {
		return domainauthorization.Membership{}, false, fmt.Errorf("find membership by group: %w", err)
	}
	return membership, true, nil
}
