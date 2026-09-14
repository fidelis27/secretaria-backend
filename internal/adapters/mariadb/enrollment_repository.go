package mariadb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	domainenrollment "github.com/fidelis27/secretaria-backend/internal/domain/enrollment"
)

type EnrollmentRepository struct {
	connection *Connection
}

func NewEnrollmentRepository(connection *Connection) EnrollmentRepository {
	return EnrollmentRepository{connection: connection}
}

func (repository EnrollmentRepository) Create(ctx context.Context, entity domainenrollment.Enrollment) error {
	_, err := repository.connection.database.ExecContext(ctx,
		`INSERT INTO enrollments (id, student_id, institution_id, status) VALUES (?, ?, ?, ?)`,
		entity.ID, entity.StudentID, entity.InstitutionID, entity.Status,
	)
	if err != nil {
		return fmt.Errorf("create enrollment: %w", err)
	}
	return nil
}

func (repository EnrollmentRepository) List(ctx context.Context) ([]domainenrollment.Enrollment, error) {
	rows, err := repository.connection.database.QueryContext(ctx,
		`SELECT id, student_id, institution_id, status, suspension_reason, suspended_at FROM enrollments ORDER BY student_id, id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list enrollments: %w", err)
	}
	defer rows.Close()

	result := make([]domainenrollment.Enrollment, 0)
	for rows.Next() {
		var entity domainenrollment.Enrollment
		var suspensionReason sql.NullString
		var suspendedAt *time.Time
		if err := rows.Scan(&entity.ID, &entity.StudentID, &entity.InstitutionID, &entity.Status, &suspensionReason, &suspendedAt); err != nil {
			return nil, fmt.Errorf("scan enrollment: %w", err)
		}
		if suspensionReason.Valid {
			entity.SuspensionReason = suspensionReason.String
		}
		entity.SuspendedAt = suspendedAt
		result = append(result, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enrollments: %w", err)
	}
	return result, nil
}

func (repository EnrollmentRepository) FindByID(ctx context.Context, studentID string, enrollmentID string) (domainenrollment.Enrollment, bool, error) {
	var entity domainenrollment.Enrollment
	var suspensionReason sql.NullString
	var suspendedAt *time.Time
	err := repository.connection.database.QueryRowContext(ctx,
		`SELECT id, student_id, institution_id, status, suspension_reason, suspended_at
		 FROM enrollments WHERE id = ? AND student_id = ?`, enrollmentID, studentID,
	).Scan(&entity.ID, &entity.StudentID, &entity.InstitutionID, &entity.Status, &suspensionReason, &suspendedAt)
	if err == sql.ErrNoRows {
		return domainenrollment.Enrollment{}, false, nil
	}
	if err != nil {
		return domainenrollment.Enrollment{}, false, fmt.Errorf("find enrollment by id: %w", err)
	}
	if suspensionReason.Valid {
		entity.SuspensionReason = suspensionReason.String
	}
	entity.SuspendedAt = suspendedAt
	return entity, true, nil
}

func (repository EnrollmentRepository) Suspend(ctx context.Context, studentID string, enrollmentID string, reason string, suspendedAt time.Time) error {
	result, err := repository.connection.database.ExecContext(ctx,
		`UPDATE enrollments SET status = 'suspended', suspension_reason = ?, suspended_at = ? WHERE id = ? AND student_id = ? AND status = 'active'`,
		reason, suspendedAt, enrollmentID, studentID,
	)
	if err != nil {
		return fmt.Errorf("suspend enrollment: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check suspended enrollment: %w", err)
	}
	if rowsAffected != 1 {
		return domainenrollment.ErrEnrollmentNotActive
	}
	return nil
}

func (repository EnrollmentRepository) Reopen(ctx context.Context, studentID string, enrollmentID string) error {
	result, err := repository.connection.database.ExecContext(ctx,
		`UPDATE enrollments SET status = 'active' WHERE id = ? AND student_id = ? AND status = 'suspended'`,
		enrollmentID, studentID,
	)
	if err != nil {
		return fmt.Errorf("reopen enrollment: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check reopened enrollment: %w", err)
	}
	if rowsAffected != 1 {
		return domainenrollment.ErrEnrollmentNotSuspended
	}
	return nil
}

func (repository EnrollmentRepository) Transfer(ctx context.Context, studentID string, enrollmentID string, destination domainenrollment.Enrollment) error {
	transaction, err := repository.connection.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin enrollment transfer: %w", err)
	}
	defer transaction.Rollback()

	result, err := transaction.ExecContext(ctx,
		`UPDATE enrollments SET status = 'transferred' WHERE id = ? AND student_id = ? AND status = 'active'`,
		enrollmentID, studentID,
	)
	if err != nil {
		return fmt.Errorf("close source enrollment: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check source enrollment: %w", err)
	}
	if rowsAffected != 1 {
		return domainenrollment.ErrSourceEnrollmentNotActive
	}

	if _, err := transaction.ExecContext(ctx,
		`INSERT INTO enrollments (id, student_id, institution_id, status) VALUES (?, ?, ?, ?)`,
		destination.ID, destination.StudentID, destination.InstitutionID, destination.Status,
	); err != nil {
		return fmt.Errorf("create destination enrollment: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit enrollment transfer: %w", err)
	}
	return nil
}
