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
	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inBacktickQuote := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(contents); i++ {
		ch := contents[i]
		next := byte(0)
		if i+1 < len(contents) {
			next = contents[i+1]
		}

		switch {
		case inLineComment:
			if ch == '\n' {
				inLineComment = false
			}
		case inBlockComment:
			if ch == '*' && next == '/' {
				i++
				inBlockComment = false
			}
		case inSingleQuote:
			current.WriteByte(ch)
			if ch == '\'' {
				if next == '\'' {
					current.WriteByte(next)
					i++
				} else {
					inSingleQuote = false
				}
			}
		case inDoubleQuote:
			current.WriteByte(ch)
			if ch == '"' {
				if next == '"' {
					current.WriteByte(next)
					i++
				} else {
					inDoubleQuote = false
				}
			}
		case inBacktickQuote:
			current.WriteByte(ch)
			if ch == '`' {
				if next == '`' {
					current.WriteByte(next)
					i++
				} else {
					inBacktickQuote = false
				}
			}
		case ch == '-' && next == '-':
			i++
			inLineComment = true
		case ch == '/' && next == '*':
			i++
			inBlockComment = true
		case ch == '#':
			inLineComment = true
		case ch == '\'':
			current.WriteByte(ch)
			inSingleQuote = true
		case ch == '"':
			current.WriteByte(ch)
			inDoubleQuote = true
		case ch == '`':
			current.WriteByte(ch)
			inBacktickQuote = true
		case ch == ';':
			if statement := strings.TrimSpace(current.String()); statement != "" {
				statements = append(statements, statement)
			}
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}

	if statement := strings.TrimSpace(current.String()); statement != "" {
		statements = append(statements, statement)
	}
	return statements
}
