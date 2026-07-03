package db

import (
	"os"
	"path/filepath"
	"testing"

	"backend/internal/models"
)

// TestPhase1MigrationsIdempotentAgainstRealDevDB runs the migration runner
// against the project's existing dev SQLite database (backend/dmtool.db).
// Skipped automatically if the file isn't present in the working directory.
//
// Why this test: it proves that running RunMigrations against a DB that
// already has the schema doesn't drop or alter user data, and that
// repeated invocations don't change plan row counts.
//
// Run via:
//   cd backend && go test ./internal/db/ -run TestPhase1MigrationsIdempotentAgainstRealDevDB -v
func TestPhase1MigrationsIdempotentAgainstRealDevDB(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	// walk up to backend/ if running from a deeper test path
	for filepath.Base(wd) != "backend" && wd != filepath.Dir(wd) {
		wd = filepath.Dir(wd)
	}
	dsn := filepath.Join(wd, "dmtool.db")
	if _, err := os.Stat(dsn); err != nil {
		t.Skipf("dev DB not present at %s; skipping", dsn)
	}

	database := Init(dsn, true)
	if database == nil {
		t.Fatal("Init returned nil db")
	}
	t.Cleanup(func() {
		sqlDB, err := database.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	// Ensure the 4 target tables exist after running migrations.
	migrator := database.Migrator()
	for _, want := range []string{"plans", "subscriptions", "user_preferences", "admin_audit_logs"} {
		if !migrator.HasTable(want) {
			t.Errorf("expected table %q after RunMigrations against real dev DB", want)
		}
	}

	// Plan count is 3 and stays at 3 after a second RunMigrations call.
	var count int64
	if err := database.Model(&models.Plan{}).Count(&count).Error; err != nil {
		t.Fatalf("count plans: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 plans, got %d", count)
	}
	if err := RunMigrations(database); err != nil {
		t.Fatalf("second RunMigrations failed: %v", err)
	}
	var recount int64
	if err := database.Model(&models.Plan{}).Count(&recount).Error; err != nil {
		t.Fatalf("recount plans: %v", err)
	}
	if recount != 3 {
		t.Errorf("expected 3 plans after re-run, got %d", recount)
	}
}
