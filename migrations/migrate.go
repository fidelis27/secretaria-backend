package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed *.sql
var Files embed.FS

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
		if _, err := transaction.ExecContext(ctx, string(contents)); err != nil {
			_ = transaction.Rollback()
			return fmt.Errorf("apply migration %s: %w", entry, err)
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
