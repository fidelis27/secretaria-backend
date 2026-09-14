package mariadb

import (
	"context"
	"fmt"

	domainenrollment "github.com/karol/secretaria-escolar-backend/internal/domain/enrollment"
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
		`SELECT id, student_id, institution_id, status FROM enrollments ORDER BY student_id, id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list enrollments: %w", err)
	}
	defer rows.Close()

	result := make([]domainenrollment.Enrollment, 0)
	for rows.Next() {
		var entity domainenrollment.Enrollment
		if err := rows.Scan(&entity.ID, &entity.StudentID, &entity.InstitutionID, &entity.Status); err != nil {
			return nil, fmt.Errorf("scan enrollment: %w", err)
		}
		result = append(result, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enrollments: %w", err)
	}
	return result, nil
}
