package httpserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/fidelis27/secretaria-backend/internal/platform/errors"
	"github.com/go-sql-driver/mysql"
)

func TestWriteServiceErrorMapsDatabaseFailures(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "duplicate email",
			err:         fmt.Errorf("create user: %w", &mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'ana@example.com' for key 'users.email'"}),
			wantStatus:  http.StatusConflict,
			wantMessage: apperrors.ErrDuplicateEmail.Error(),
		},
		{
			name:        "duplicate membership",
			err:         fmt.Errorf("create membership: %w", &mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'user-1-group-1' for key 'PRIMARY'"}),
			wantStatus:  http.StatusConflict,
			wantMessage: apperrors.ErrDuplicateMembership.Error(),
		},
		{
			name:        "foreign key",
			err:         fmt.Errorf("create enrollment: %w", &mysql.MySQLError{Number: 1452, Message: "Cannot add or update a child row"}),
			wantStatus:  http.StatusUnprocessableEntity,
			wantMessage: apperrors.ErrForeignKeyViolation.Error(),
		},
		{
			name:        "database unavailable",
			err:         fmt.Errorf("ping MariaDB: %w", apperrors.ErrDatabaseUnavailable),
			wantStatus:  http.StatusFailedDependency,
			wantMessage: apperrors.ErrDatabaseUnavailable.Error(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			writeServiceError(recorder, test.err, "fallback")

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if recorder.Body.String() != "{\"error\":\""+test.wantMessage+"\"}\n" {
				t.Fatalf("body = %q", recorder.Body.String())
			}
		})
	}
}
