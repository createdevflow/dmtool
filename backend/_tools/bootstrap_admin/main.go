// bootstrap_admin ensures a single admin user exists in the database
// from env-driven configuration. Idempotent: re-runs on an existing
// admin user are no-ops (we do NOT reset the password on re-run — that
// would be a silent rotation). Use it once during initial deployment
// or to recover from a lost admin.
//
// Inputs (env vars):
//
//	ADMIN_EMAIL            (required) the admin's email
//	ADMIN_PASSWORD_HASH    (required) bcrypt hash, generated offline
//	APP_ENV                (optional) read by config.Load; dev/prod
//	DATABASE_URL           (optional) defaults to file:dmtool.db
//
// Generate the password hash offline (the admin's plaintext never
// touches the codebase):
//
//	htpasswd -bnBC 12 "" "new-password" | tr -d ':\n' | sed 's/^\$2y/\$2a/'
//
// Or use the bcrypt CLI of choice. The cost factor must match the
// registration cost (12) in backend/internal/handlers/auth_handlers.go.
//
// Run from the backend/ directory:
//
//	go run ./_tools/bootstrap_admin/
//
// On success, the script prints the admin's user ID. On failure (no
// env, DB unreachable, etc.) it exits non-zero with a human-readable
// error.
//
// Idempotency:
//
//   - If a user with ADMIN_EMAIL already exists with role="admin":
//     no-op, prints "admin already exists"
//   - If the user exists with role != "admin": role is promoted to
//     "admin" (prints "promoted existing user")
//   - If no user with that email exists: creates one with the
//     provided hash, role=admin, dashboard_mode=combined
//
// SECURITY: this script does not log the password or the hash. It logs
// only the user ID and the chosen email.
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/models"

	"gorm.io/gorm"
)

func main() {
	log.SetFlags(0)

	email := strings.TrimSpace(strings.ToLower(os.Getenv("ADMIN_EMAIL")))
	hash := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD_HASH"))
	if email == "" {
		log.Fatal("ADMIN_EMAIL is required")
	}
	if hash == "" {
		log.Fatal("ADMIN_PASSWORD_HASH is required (bcrypt hash, generated offline)")
	}
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
		log.Fatal("ADMIN_PASSWORD_HASH does not look like a bcrypt hash (expected $2a$/$2b$/$2y$ prefix)")
	}

	cfg := config.Load()
	isDev := cfg.AppEnv == "development"
	dsn := cfg.DatabaseURL
	if strings.HasPrefix(dsn, "file:") {
		dsn = strings.TrimPrefix(dsn, "file:")
	}
	if dsn == "" {
		dsn = "dmtool.db"
	}
	// Resolve relative SQLite paths to the cwd, matching main.go.
	abs, err := filepath.Abs(dsn)
	if err == nil {
		dsn = abs
	}

	database := db.Init(cfg.DatabaseURL, isDev)
	sqlDB, _ := database.DB()
	defer sqlDB.Close()

	if err := run(database, email, hash); err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}
}

func run(database *gorm.DB, email, passwordHash string) error {
	var u models.User
	err := database.Where("email = ?", email).First(&u).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// No existing user — create.
		u = models.User{
			Email:         email,
			PasswordHash:  passwordHash,
			Role:          models.RoleAdmin,
			DashboardMode: "combined",
		}
		// ADMIN_EMAIL is required; ADMIN_NAME is optional.
		if name := strings.TrimSpace(os.Getenv("ADMIN_NAME")); name != "" {
			u.Name = name
		} else {
			u.Name = "Admin"
		}
		if err := database.Create(&u).Error; err != nil {
			return fmt.Errorf("create admin: %w", err)
		}
		fmt.Printf("created admin user id=%d email=%s\n", u.ID, u.Email)
		return nil
	case err != nil:
		return fmt.Errorf("lookup user: %w", err)
	default:
		// Existing user.
		if u.Role == models.RoleAdmin {
			fmt.Printf("admin already exists id=%d email=%s (no change)\n", u.ID, u.Email)
			return nil
		}
		// Promote.
		if err := database.Model(&u).Update("role", models.RoleAdmin).Error; err != nil {
			return fmt.Errorf("promote user: %w", err)
		}
		fmt.Printf("promoted existing user to admin id=%d email=%s (was role=%s)\n", u.ID, u.Email, u.Role)
		return nil
	}
}
