package handlers

import (
	"path/filepath"
	"testing"
	"time"

	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

// TestPhase2_AutoSubscribeContract verifies that the registration flow
// the prod handler relies on (user + sub create) works against the
// same DB shape. We exercise the repos directly here — the auth
// handler itself is tested separately via integration. This keeps
// the test independent of JWT key generation.
func TestPhase2_AutoSubscribeContract(t *testing.T) {
	tmp := t.TempDir()
	dsn := filepath.Join(tmp, "phase2_register.db")

	database := db.Init(dsn, true)
	if database == nil {
		t.Fatal("db.Init returned nil")
	}
	t.Cleanup(func() {
		sqlDB, err := database.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	userRepo := repository.NewUserRepository(database)
	subRepo := repository.NewSubscriptionRepository(database)

	u := &models.User{
		Name:         "Alice",
		Email:        "alice@example.com",
		PasswordHash: "x",
		Role:         "owner",
	}
	if err := userRepo.Create(u); err != nil {
		t.Fatalf("userRepo.Create: %v", err)
	}

	// Mirror the exact subscription row Register creates.
	sub := &models.Subscription{
		UserID:   u.ID,
		PlanCode: "free",
		Status:   models.SubscriptionStatusActive,
		StartsAt: time.Now(),
	}
	if err := subRepo.Create(sub); err != nil {
		t.Fatalf("subRepo.Create: %v", err)
	}

	// Read it back. We expect exactly one row, scoped to this user.
	var subs []models.Subscription
	if err := database.Where("user_id = ?", u.ID).Find(&subs).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("got %d subs, want 1", len(subs))
	}
	if subs[0].PlanCode != "free" {
		t.Errorf("plan = %q, want free", subs[0].PlanCode)
	}
	if subs[0].Status != models.SubscriptionStatusActive {
		t.Errorf("status = %q, want active", subs[0].Status)
	}
	_ = gorm.ErrRecordNotFound // silence
}
