package models

import "time"

// OAuthCredential stores encrypted OAuth tokens for Google and Meta.
// AccessToken and RefreshToken fields are AES-256-GCM encrypted before
// being written to the DB and decrypted after reading.
type OAuthCredential struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID    uint   `gorm:"index" json:"user_id"`
	ProjectID uint   `gorm:"index" json:"project_id"`
	Provider  string `gorm:"not null" json:"provider"` // google | meta

	// AES-256-GCM encrypted ciphertext stored as base64
	AccessTokenEnc  string    `gorm:"column:access_token_enc" json:"-"`
	RefreshTokenEnc string    `gorm:"column:refresh_token_enc" json:"-"`
	ExpiresAt       time.Time `json:"expires_at"`

	// Sync state
	LastSyncedAt *time.Time `json:"last_synced_at"`
	SyncError    string     `json:"sync_error"`
	IsExpired    bool       `gorm:"default:false" json:"is_expired"`
}
