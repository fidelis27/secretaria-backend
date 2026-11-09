package devseeds

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

func TestApplyRejectsProduction(t *testing.T) {
	err := Apply(context.Background(), &sql.DB{}, "production")
	if err == nil {
		t.Fatal("Apply() should reject production")
	}
	if !strings.Contains(err.Error(), "disabled in production") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSplitStatementsIgnoresSemicolonsInsideLiteralsAndComments(t *testing.T) {
	contents := `INSERT INTO audit_events (payload) VALUES ('{"note":"semicolon; inside string"}');
-- ignore semicolon ; in comments
INSERT INTO users (name) VALUES ('O''Brien; still one field');
/* block comment ; with semicolon */
SELECT 1;`

	statements := splitStatements(contents)
	want := []string{
		"INSERT INTO audit_events (payload) VALUES ('{\"note\":\"semicolon; inside string\"}')",
		"INSERT INTO users (name) VALUES ('O''Brien; still one field')",
		"SELECT 1",
	}
	if len(statements) != len(want) {
		t.Fatalf("splitStatements() returned %d statements, want %d: %#v", len(statements), len(want), statements)
	}
	for i := range want {
		if statements[i] != want[i] {
			t.Fatalf("statement %d = %q, want %q", i, statements[i], want[i])
		}
	}
}
