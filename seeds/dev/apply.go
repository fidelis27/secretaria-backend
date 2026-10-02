package devseeds

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

const ProductionEnv = "production"

//go:embed *.sql
var Files embed.FS

func Apply(ctx context.Context, database *sql.DB, appEnv string) error {
	if strings.EqualFold(strings.TrimSpace(appEnv), ProductionEnv) {
		return fmt.Errorf("dev seeds are disabled in production")
	}

	entries, err := fs.Glob(Files, "*.sql")
	if err != nil {
		return fmt.Errorf("list dev seed files: %w", err)
	}
	sort.Strings(entries)

	for _, entry := range entries {
		contents, err := Files.ReadFile(entry)
		if err != nil {
			return fmt.Errorf("read dev seed %s: %w", entry, err)
		}
		for _, statement := range splitStatements(string(contents)) {
			if _, err := database.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("apply dev seed %s: %w", entry, err)
			}
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
