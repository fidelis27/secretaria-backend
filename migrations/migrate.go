package migrations

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"github.com/go-sql-driver/mysql"
)

//go:embed *.sql
var Files embed.FS

// alreadyExistsErrorCodes lists MySQL/MariaDB error numbers that indicate a
// DDL statement's target object already exists. MySQL implicitly commits DDL
// statements, so a migration that fails partway through leaves earlier
// CREATE TABLE/INDEX statements applied even though the surrounding
// transaction is rolled back and the migration is never recorded as applied.
// Re-running such a migration would otherwise fail forever, so these errors
// are treated as "already applied" for that statement and skipped.
var alreadyExistsErrorCodes = map[uint16]bool{
	1050: true, // ER_TABLE_EXISTS_ERROR
	1060: true, // ER_DUP_FIELDNAME
	1061: true, // ER_DUP_KEYNAME
	1068: true, // ER_MULTIPLE_PRI_KEY
	1826: true, // ER_FK_DUP_NAME
}

func Apply(ctx context.Context, database *sql.DB) error {
	if _, err := database.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) NOT NULL PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.Glob(Files, "*.sql")
	if err != nil {
		return fmt.Errorf("list migration files: %w", err)
	}
	sort.Strings(entries)

	for _, entry := range entries {
		var applied bool
		if err := database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)", entry).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", entry, err)
		}
		if applied {
			continue
		}

		contents, err := Files.ReadFile(entry)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry, err)
		}
		transaction, err := database.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry, err)
		}
		for _, statement := range splitStatements(string(contents)) {
			if _, err := transaction.ExecContext(ctx, statement); err != nil {
				var mysqlErr *mysql.MySQLError
				if errors.As(err, &mysqlErr) && alreadyExistsErrorCodes[mysqlErr.Number] {
					slog.Warn("skipping already-applied statement", "migration", entry, "error", err)
					continue
				}
				_ = transaction.Rollback()
				return fmt.Errorf("apply migration %s: %w", entry, err)
			}
		}
		if _, err := transaction.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", entry); err != nil {
			_ = transaction.Rollback()
			return fmt.Errorf("record migration %s: %w", entry, err)
		}
		if err := transaction.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry, err)
		}
	}
	return nil
}

func splitStatements(contents string) []string {
	parts := strings.Split(contents, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		if statement := strings.TrimSpace(part); statement != "" {
			statements = append(statements, statement)
		}
	}
	return statements
}
