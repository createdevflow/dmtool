package models

import "time"

// RefreshToken stores an opaque 256-bit refresh token (SHA-256 hashed) per
// device session.  It is rotated on every use and immediately invalidated
// when consumed.  Expired / revoked tokens are purged by a background cron.
type RefreshToken struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	TokenHash  string    `gorm:"uniqueIndex;not null" json:"-"` // SHA-256 of the raw token
	ExpiresAt  time.Time `gorm:"not null" json:"expires_at"`
	Revoked    bool      `gorm:"default:false" json:"revoked"`
	DeviceInfo string    `json:"device_info"` // User-Agent fingerprint
}
