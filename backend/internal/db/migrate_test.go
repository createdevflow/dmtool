package db

import (
	"path/filepath"
	"testing"

	"backend/internal/models"
)

// TestPhase1MigrationsSQLite exercises the full migration + seed path
// against a fresh SQLite database. This mirrors what would happen on the
// Render Postgres instance in production, but uses SQLite because that's
// what we can run locally without a Postgres dependency.
//
// What it asserts:
//   - RunMigrations on a fresh DB creates exactly the 4 target tables.
//   - Re-running RunMigrations is a no-op (no error, no duplicate inserts).
//   - SeedPlans inserts 3 baseline plan rows.
//   - SeedPlans is idempotent: re-running keeps the row count at 3.
//   - The three seeded plan codes have the expected names and max_sites.
func TestPhase1MigrationsSQLite(t *testing.T) {
	tmp := t.TempDir()
	dsn := filepath.Join(tmp, "phase1.db")

	database := Init(dsn, true)
	if database == nil {
		t.Fatal("Init returned nil db")
	}
	t.Cleanup(func() {
		// Windows can't always unlink the temp SQLite file while a
		// connection is still open. Force-close before tempdir cleanup.
		sqlDB, err := database.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	migrator := database.Migrator()

	for _, want := range []string{"plans", "subscriptions", "user_preferences", "admin_audit_logs"} {
		if !migrator.HasTable(want) {
			t.Errorf("expected table %q to exist after RunMigrations", want)
		}
	}

	// Run again — must be a no-op.
	if err := RunMigrations(database); err != nil {
		t.Fatalf("second RunMigrations failed: %v", err)
	}

	// Plans table contents.
	var plans []models.Plan
	if err := database.Find(&plans).Error; err != nil {
		t.Fatalf("query plans failed: %v", err)
	}
	if len(plans) != 3 {
		t.Fatalf("expected 3 seeded plan rows, got %d", len(plans))
	}

	wantByCode := map[string]struct {
		Name     string
		MaxSites int
	}{
		"free":        {"Free", 1},
		"pro_monthly": {"Pro · Monthly", 3},
		"pro_yearly":  {"Pro · Yearly", 3},
	}
	for _, p := range plans {
		want, ok := wantByCode[p.Code]
		if !ok {
			t.Errorf("unexpected seeded plan code %q", p.Code)
			continue
		}
		if p.Name != want.Name {
			t.Errorf("plan %q name = %q, want %q", p.Code, p.Name, want.Name)
		}
		if p.MaxSites != want.MaxSites {
			t.Errorf("plan %q max_sites = %d, want %d", p.Code, p.MaxSites, want.MaxSites)
		}
	}

	// Re-running SeedPlans via RunMigrations keeps count at 3.
	if err := RunMigrations(database); err != nil {
		t.Fatalf("third RunMigrations failed: %v", err)
	}
	var recount int64
	if err := database.Model(&models.Plan{}).Count(&recount).Error; err != nil {
		t.Fatalf("count plans failed: %v", err)
	}
	if recount != 3 {
		t.Fatalf("expected plan count to remain 3 after re-seed, got %d", recount)
	}
}
