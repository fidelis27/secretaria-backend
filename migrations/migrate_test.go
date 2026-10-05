package migrations

import (
	"io/fs"
	"regexp"
	"testing"
)

var emailLiteralPattern = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)

func TestSkipsHistoricalSeedMigrationsInProductionPath(t *testing.T) {
	for _, version := range []string{
		"004_seed_demo_super_admin.sql",
		"006_seed_demo_institution.sql",
		"007_promote_owner_super_admin.sql",
	} {
		if !skippedProductionMigrations[version] {
			t.Fatalf("version %q should be skipped from production migrations", version)
		}

	}
	if skippedProductionMigrations["008_cleanup_legacy_dev_seeds.sql"] {
		t.Fatal("cleanup migration must remain in the production path")
	}
}

func TestNewMigrationsDoNotContainEmailLiterals(t *testing.T) {
	legacyEmailMigrations := map[string]bool{
		"004_seed_demo_super_admin.sql":     true,
		"007_promote_owner_super_admin.sql": true,
		"008_cleanup_legacy_dev_seeds.sql":  true,
	}
	entries, err := fs.Glob(Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range entries {
		if legacyEmailMigrations[name] {
			continue
		}
		contents, err := Files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if match := emailLiteralPattern.Find(contents); match != nil {
			t.Errorf("migration %s contains a literal email address", name)
		}
	}
}
