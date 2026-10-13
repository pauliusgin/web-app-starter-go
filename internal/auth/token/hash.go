package token

import (
	"crypto/sha256"
	"encoding/hex"
)

// Hash returns the hex-encoded SHA-256 of a token, the only form ever stored.
func Hash(plaintext string) string {
	hash := sha256.Sum256([]byte(plaintext))

	return hex.EncodeToString(hash[:])
}
