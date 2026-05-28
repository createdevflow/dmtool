// Package db initialises the database connection using GORM and runs
// AutoMigrate for all v2 models in development mode.
package db

import (
	"log"
	"strings"

	"backend/internal/models"

	glebsqlite "github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Init opens the database connection, runs AutoMigrate for all models in dev,
// and returns the *gorm.DB instance for dependency injection.
func Init(databaseURL string, isDev bool) *gorm.DB {
	gormCfg := &gorm.Config{}
	if isDev {
		gormCfg.Logger = logger.Default.LogMode(logger.Info)
	} else {
		gormCfg.Logger = logger.Default.LogMode(logger.Error)
	}

	var db *gorm.DB
	var err error

	if strings.HasPrefix(databaseURL, "postgres://") || strings.HasPrefix(databaseURL, "postgresql://") {
		// PostgreSQL (production)
		db, err = gorm.Open(postgres.Open(databaseURL), gormCfg)
	} else {
		// SQLite (development) — uses pure-Go driver, no CGO required
		dsn := strings.TrimPrefix(databaseURL, "file:")
		if dsn == "" {
			dsn = "dmtool.db"
		}
		db, err = gorm.Open(glebsqlite.Open(dsn), gormCfg)
	}

	if err != nil {
		log.Fatalf("[db] failed to connect to database: %v", err)
	}

	// AutoMigrate all v2 models (dev only; prod uses SQL migration files)
	if isDev {
		if err := db.AutoMigrate(
			&models.User{},
			&models.RefreshToken{},
			&models.Project{},
			&models.OAuthCredential{},
			&models.Metric{},
			&models.SocialMetric{},
			&models.Insight{},
			&models.Task{},
			&models.SEOIssue{},
			&models.KeywordResult{},
		); err != nil {
			log.Fatalf("[db] AutoMigrate failed: %v", err)
		}
		log.Println("[db] AutoMigrate completed successfully")
	}

	log.Println("[db] Database connection established")
	return db
}

// Ping checks that the database is reachable (used by health check endpoint).
func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
