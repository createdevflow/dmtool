package handlers

import (
	"crypto/rsa"

	"backend/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

// hashWithBcrypt hashes a plaintext password with bcrypt at cost 12
// (the same cost used at registration in auth_handlers.go). Centralised
// here so the admin reset-password path can't drift from the
// registration path.
func hashWithBcrypt(plaintext string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
	if err != nil {
		// bcrypt failure is fatal at any boundary; the registration
		// handler treats it the same way.
		panic(err)
	}
	return string(h)
}

// mintImpersonationToken is a thin wrapper over
// utils.GenerateImpersonationToken kept here so the AdminHandler
// doesn't need to import crypto/rsa. The caller (NewAdminHandler) is
// the only one that holds the *rsa.PrivateKey, and it passes it
// straight through.
func mintImpersonationToken(privKey any, adminID, targetUserID uint, targetEmail, targetRole string) (string, error) {
	rsaKey, ok := privKey.(*rsa.PrivateKey)
	if !ok || rsaKey == nil {
		// Fall back: pass nil; utils will return an error.
		rsaKey = nil
	}
	return utils.GenerateImpersonationToken(rsaKey, adminID, targetUserID, targetEmail, targetRole)
}
