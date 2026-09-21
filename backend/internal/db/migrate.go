package db

import (
	"errors"
	"fmt"
	"log"

	"backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RunMigrations creates any production-required tables that AutoMigrate is
// not allowed to create. AutoMigrate handles dev (per db.go); this path
// runs in every environment and is the only schema change that ever
// happens in production, gating on GORM's Migrator.CreateTable which is
// idempotent ("CREATE TABLE IF NOT EXISTS" on both PostgreSQL and SQLite).
//
// Scope rules:
//   - Forward-only. No DROP. No ALTER (except idempotent column adds
//     gated on HasColumn, see addUserDashboardModeColumn below).
//   - Idempotent. Re-running is a no-op.
//   - Called from db.Init unconditionally — not gated by APP_ENV.
//
// Adding a new table: append its model below in dependency order (FK
// targets must exist first). Keep the model definition in lockstep with
// this list — see internal/models/*.
func RunMigrations(database *gorm.DB) error {
	if database == nil {
		log.Println("[migrate] nil db; skipping")
		return nil
	}

	// Order: independent tables first, then tables with FKs to them.
	targets := []any{
		&models.Plan{},
		&models.UserPreference{},
		&models.Subscription{},
		&models.AdminAuditLog{},
		&models.UserActivity{},
		&models.SystemHealth{},
		&models.Permission{},
		&models.Role{},
		&models.RolePermission{},
	}

	migrator := database.Migrator()
	created := 0
	for _, t := range targets {
		if !migrator.HasTable(t) {
			if err := migrator.CreateTable(t); err != nil {
				return err
			}
			created++
		}
	}
	log.Printf("[migrate] created %d tables (skipped %d already present)\n",
		created, len(targets)-created)

	// Forward-only column adds for existing tables. Phase 4: add
	// User.DashboardMode. Idempotent — Migrator.HasColumn gates each
	// addition, so re-running is a no-op.
	if err := addUserDashboardModeColumn(migrator); err != nil {
		return err
	}
	// Phase 1: admin user management columns.
	if err := addAdminUserColumns(migrator); err != nil {
		return err
	}

	// Idempotent seed. Re-runs are no-ops because we use ON CONFLICT.
	if err := SeedPlans(database); err != nil {
		return err
	}
	if err := SeedPermissions(database); err != nil {
		return err
	}
	return SeedSuperAdmin(database)
}

// addUserDashboardModeColumn adds users.dashboard_mode if missing. SQLite
// (dev) and PostgreSQL (prod) both support ALTER TABLE ADD COLUMN with
// a default value, and both treat this as a no-op when the column
// already exists. The migrator's HasColumn check makes that explicit.
func addUserDashboardModeColumn(migrator gorm.Migrator) error {
	if !migrator.HasTable("users") || migrator.HasColumn(&models.User{}, "dashboard_mode") {
		return nil
	}
	if err := migrator.AddColumn(&models.User{}, "DashboardMode"); err != nil {
		return err
	}
	log.Println("[migrate] added column users.dashboard_mode (default 'combined')")
	return nil
}

// addAdminUserColumns adds Phase 1 admin columns to users if missing:
// disabled_at, disabled_reason, last_login_at, login_count.
func addAdminUserColumns(migrator gorm.Migrator) error {
	if !migrator.HasTable("users") {
		return nil
	}
	cols := []struct {
		field string
		col   string
	}{
		{"DisabledAt", "disabled_at"},
		{"DisabledReason", "disabled_reason"},
		{"LastLoginAt", "last_login_at"},
		{"LoginCount", "login_count"},
	}
	for _, c := range cols {
		if !migrator.HasColumn(&models.User{}, c.col) {
			if err := migrator.AddColumn(&models.User{}, c.field); err != nil {
				return err
			}
			log.Printf("[migrate] added column users.%s\n", c.col)
		}
	}
	return nil
}

// SeedPlans inserts the three baseline plan rows if absent. Idempotent.
// Uses GORM's clause.OnConflict{DoNothing: true} — PostgreSQL emits
// "ON CONFLICT DO NOTHING", SQLite emits "OR IGNORE".
func SeedPlans(database *gorm.DB) error {
	type planSeed struct {
		Code         string
		Name         string
		Description  string
		TierRank     int
		MonthlyCents int
		YearlyCents  int
		MaxSites     int
	}
	seeds := []planSeed{
		{
			Code:         "free",
			Name:         "Free",
			Description:  "Get started with one connected website.",
			TierRank:     0,
			MonthlyCents: 0,
			YearlyCents:  0,
			MaxSites:     1,
		},
		{
			Code:         "pro_monthly",
			Name:         "Pro · Monthly",
			Description:  "Up to 3 websites, AI insights, and full audit history.",
			TierRank:     10,
			MonthlyCents: 2900,
			YearlyCents:  0,
			MaxSites:     3,
		},
		{
			Code:         "pro_yearly",
			Name:         "Pro · Yearly",
			Description:  "Up to 3 websites, AI insights, and full audit history — billed yearly.",
			TierRank:     10,
			MonthlyCents: 0,
			YearlyCents:  29000,
			MaxSites:     3,
		},
	}

	for _, s := range seeds {
		row := &models.Plan{
			Code:         s.Code,
			Name:         s.Name,
			Description:  s.Description,
			TierRank:     s.TierRank,
			MonthlyCents: s.MonthlyCents,
			YearlyCents:  s.YearlyCents,
			MaxSites:     s.MaxSites,
			IsActive:     true,
		}
		stmt := database.Clauses(clause.OnConflict{DoNothing: true}).Create(row)
		if stmt.Error != nil {
			return stmt.Error
		}
	}
	log.Printf("[migrate] seeded %d plan rows (no-op on re-run)\n", len(seeds))
	return nil
}

// SeedPermissions inserts the staff permission catalog if absent.
// Idempotent on code. Does not update name/description on re-run —
// changing copy is a later migration if we need it.
func SeedPermissions(database *gorm.DB) error {
	catalog := models.PermissionCatalog()
	for i := range catalog {
		row := catalog[i]
		stmt := database.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "code"}},
			DoNothing: true,
		}).Create(&row)
		if stmt.Error != nil {
			return stmt.Error
		}
	}
	log.Printf("[migrate] seeded %d permission rows (no-op on re-run)\n", len(catalog))
	return nil
}

// SeedSuperAdmin ensures the system Super Admin role (code=admin) exists
// and has every catalog permission. Users with users.role=admin resolve
// to this row by code. Idempotent.
func SeedSuperAdmin(database *gorm.DB) error {
	var role models.Role
	err := database.Where("code = ?", models.RoleCodeSuperAdmin).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		role = models.Role{
			Code:        models.RoleCodeSuperAdmin,
			Name:        "Super Admin",
			Description: "Full staff access. System role; last holder cannot be demoted.",
			IsSystem:    true,
		}
		if err := database.Create(&role).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if err := database.Model(&role).Updates(map[string]any{
			"name":        "Super Admin",
			"description": "Full staff access. System role; last holder cannot be demoted.",
			"is_system":   true,
		}).Error; err != nil {
			return err
		}
	}

	catalog := models.PermissionCatalog()
	for _, p := range catalog {
		var perm models.Permission
		if err := database.Where("code = ?", p.Code).First(&perm).Error; err != nil {
			return err
		}
		join := models.RolePermission{RoleID: role.ID, PermissionID: perm.ID}
		if err := database.Clauses(clause.OnConflict{DoNothing: true}).Create(&join).Error; err != nil {
			return err
		}
	}
	log.Printf("[migrate] super admin role id=%d granted %d catalog perms\n", role.ID, len(catalog))
	return nil
}

// SuperAdminCatalogReady is the Step 5 hard gate. Every catalog code
// must be on the Super Admin role, and at least one user.role=admin
// exists (they resolve by matching roles.code).
func SuperAdminCatalogReady(database *gorm.DB) error {
	if database == nil {
		return errGate("nil db")
	}
	catalog := models.PermissionCatalog()
	var role models.Role
	if err := database.Where("code = ?", models.RoleCodeSuperAdmin).First(&role).Error; err != nil {
		return errGate("super admin role %q missing: %v", models.RoleCodeSuperAdmin, err)
	}
	var codes []string
	if err := database.Table("permissions").
		Select("permissions.code").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", role.ID).
		Pluck("code", &codes).Error; err != nil {
		return errGate("list super admin perms: %v", err)
	}
	have := map[string]bool{}
	for _, c := range codes {
		have[c] = true
	}
	var missing []string
	for _, p := range catalog {
		if !have[p.Code] {
			missing = append(missing, p.Code)
		}
	}
	if len(missing) > 0 {
		return errGate("super admin missing permissions: %v", missing)
	}
	var n int64
	if err := database.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&n).Error; err != nil {
		return errGate("count admin users: %v", err)
	}
	if n == 0 {
		return errGate("no users with role=%s to resolve to super admin", models.RoleAdmin)
	}
	return nil
}

func errGate(format string, args ...any) error {
	return fmt.Errorf("super-admin hard gate: "+format, args...)
}
