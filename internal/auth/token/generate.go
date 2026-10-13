// Package token mints the opaque secrets other packages hand out, and hashes them for storage.
package token

import (
	"crypto/rand"
	"encoding/base64"
)

const tokenByteLength = 32

// Generated is a freshly minted token, in plain text and in its hash.
type Generated struct {
	Plaintext string
	Hash      string
}

// Generate returns a new token drawn from the operating system's crypto generator.
func Generate() Generated {
	secret := [tokenByteLength]byte{}
	rand.Read(secret[:])

	return generatedFromSecret(secret)
}

func generatedFromSecret(secret [tokenByteLength]byte) Generated {
	plaintext := base64.RawURLEncoding.EncodeToString(secret[:])

	return Generated{Plaintext: plaintext, Hash: Hash(plaintext)}
}
