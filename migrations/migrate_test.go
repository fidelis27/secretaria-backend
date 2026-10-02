package migrations

import "testing"

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
