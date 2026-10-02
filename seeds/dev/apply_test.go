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
