package db

import (
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
//   - Forward-only. No DROP. No ALTER.
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

	// Idempotent seed. Re-runs are no-ops because we use ON CONFLICT.
	return SeedPlans(database)
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
