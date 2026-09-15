package mariadb

import (
	"context"
	"fmt"
	"strings"

	domainstudent "github.com/fidelis27/secretaria-backend/internal/domain/student"
)

type StudentRepository struct {
	connection *Connection
}

func NewStudentRepository(connection *Connection) StudentRepository {
	return StudentRepository{connection: connection}
}

func (repository StudentRepository) Create(ctx context.Context, entity domainstudent.Student) error {
	_, err := repository.connection.database.ExecContext(ctx,
		`INSERT INTO students (id, name, institution_id, status) VALUES (?, ?, ?, ?)`,
		entity.ID, entity.Name, entity.InstitutionID, entity.Status,
	)
	if err != nil {
		return fmt.Errorf("create student: %w", err)
	}
	return nil
}

func (repository StudentRepository) List(ctx context.Context) ([]domainstudent.Student, error) {
	rows, err := repository.connection.database.QueryContext(ctx,
		`SELECT id, name, institution_id, status FROM students ORDER BY name, id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}
	defer rows.Close()

	result := make([]domainstudent.Student, 0)
	for rows.Next() {
		var entity domainstudent.Student
		if err := rows.Scan(&entity.ID, &entity.Name, &entity.InstitutionID, &entity.Status); err != nil {
			return nil, fmt.Errorf("scan student: %w", err)
		}
		result = append(result, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate students: %w", err)
	}
	return result, nil
}

func (repository StudentRepository) ListByInstitutionIDs(ctx context.Context, institutionIDs []string) ([]domainstudent.Student, error) {
	if len(institutionIDs) == 0 {
		return []domainstudent.Student{}, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(institutionIDs)), ",")
	args := make([]any, len(institutionIDs))
	for index, id := range institutionIDs {
		args[index] = id
	}
	rows, err := repository.connection.database.QueryContext(ctx,
		`SELECT id, name, institution_id, status FROM students WHERE institution_id IN (`+placeholders+`) ORDER BY name, id`, args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list students by scope: %w", err)
	}
	defer rows.Close()
	result := make([]domainstudent.Student, 0)
	for rows.Next() {
		var entity domainstudent.Student
		if err := rows.Scan(&entity.ID, &entity.Name, &entity.InstitutionID, &entity.Status); err != nil {
			return nil, fmt.Errorf("scan scoped student: %w", err)
		}
		result = append(result, entity)
	}
	return result, rows.Err()
}
