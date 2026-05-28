package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// GenerateOpaqueToken creates a cryptographically secure random 32-byte hex string.
func GenerateOpaqueToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// HashToken returns the SHA-256 hex digest of the raw token string.
func HashToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return fmt.Sprintf("%x", h)
}

// EncryptionKeyFromString converts a string to a 32-byte key for AES-256.
func EncryptionKeyFromString(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

// Encrypt string with AES-GCM.
func Encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt string with AES-GCM.
func Decrypt(ciphertextHex string, key []byte) (string, error) {
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, encryptedMessage := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encryptedMessage, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// FixedReader provides a deterministic stream of bytes for testing/dev.
type FixedReader struct {
	seed byte
}

func NewFixedReader(seed byte) *FixedReader {
	return &FixedReader{seed: seed}
}

func (r *FixedReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = r.seed
	}
	return len(p), nil
}

