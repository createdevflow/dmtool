package db

import (
	"path/filepath"
	"testing"

	"backend/internal/models"

	glebsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestPhase4_DashboardModeColumnAdd simulates the production boot path:
// a database where `users` already exists WITHOUT the dashboard_mode
// column. After calling RunMigrations once via db.Init, the column
// must exist with the documented default.
//
// SQLite is used here for the test; Postgres uses the same GORM
// Migrator.HasColumn gate inside the same code path, so the assertion
// holds for both.
func TestPhase4_DashboardModeColumnAdd(t *testing.T) {
	tmp := t.TempDir()
	dsn := filepath.Join(tmp, "phase4.db")

	// First "boot": open the file with a stripped User shape so the
	// users table has no dashboard_mode column. This mirrors the
	// production data state right now (no live deploy has run phase 4 yet).
	bootstrap, err := gorm.Open(glebsqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open bootstrap: %v", err)
	}
	bootstrap.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TIMESTAMP,
		updated_at TIMESTAMP,
		deleted_at TIMESTAMP,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		phone TEXT,
		password_hash TEXT NOT NULL,
		role TEXT DEFAULT 'owner'
	)`)
	if !bootstrap.Migrator().HasTable("users") {
		t.Fatalf("bootstrap users table not created")
	}
	if bootstrap.Migrator().HasColumn(&models.User{}, "dashboard_mode") {
		t.Fatalf("dashboard_mode column was present already — test fixture is wrong")
	}
	closeDB(bootstrap)

	// Second "boot": real RunMigrations must add the column with default 'combined'.
	database := Init(dsn, true)
	t.Cleanup(func() { closeDB(database) })

	migrator := database.Migrator()
	if !migrator.HasColumn(&models.User{}, "dashboard_mode") {
		t.Fatalf("expected users.dashboard_mode after RunMigrations; not present")
	}

	// Verify the default is 'combined' on existing rows by inserting one
	// without specifying the column.
	database.Exec("INSERT INTO users (name, email, password_hash, role) VALUES ('x', 'x@x', 'x', 'owner')")
	var mode string
	database.Raw("SELECT dashboard_mode FROM users WHERE email = ?", "x@x").Scan(&mode)
	if mode != "combined" {
		t.Errorf("default value = %q, want 'combined'", mode)
	}
}

// TestPhase4_DashboardModeColumnAdd_Idempotent runs RunMigrations twice
// in a row and verifies the second call observes the column (no second
// ALTER). This guards against a regression where HasColumn fails to
// detect the column after creation and we end up issuing ALTER TABLE
// repeatedly, which on Postgres can lock the table.
func TestPhase4_DashboardModeColumnAdd_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	dsn := filepath.Join(tmp, "phase4_idem.db")

	// First boot — RunMigrations via db.Init creates the column.
	first := Init(dsn, true)
	closeDB(first)

	// Second call: column already exists, AddColumn path must be skipped.
	second := openPhase4DB(t, dsn)
	if err := RunMigrations(second); err != nil {
		t.Fatalf("RunMigrations (second): %v", err)
	}
	closeDB(second)

	// Re-open and verify the column still exists.
	database := Init(dsn, true)
	t.Cleanup(func() { closeDB(database) })
	migrator := database.Migrator()
	if !migrator.HasColumn(&models.User{}, "dashboard_mode") {
		t.Fatalf("dashboard_mode missing after idempotent re-run")
	}
}

// openPhase4DB is a one-shot opener used by the idempotency test so it
// doesn't compete with db.Init on the schema-creation path.
func openPhase4DB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	d, err := gorm.Open(glebsqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { closeDB(d) })
	return d
}

// closeDB does a best-effort close on the underlying *sql.DB so the
// SQLite file handle is released before t.TempDir cleanup tries to
// unlink the file. Without this, Windows holds the file open and the
// test's own TempDir cleanup fails.
func closeDB(database *gorm.DB) {
	if database == nil {
		return
	}
	sqlDB, err := database.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}
