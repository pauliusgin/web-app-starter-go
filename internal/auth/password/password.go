// Package password hashes and verifies user passwords with argon2id.
package password

import "github.com/alexedwards/argon2id"

// Hash returns an argon2id encoded hash of the password, safe to store as-is.
func Hash(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", ErrHashingFailed{Cause: err}
	}

	return hash, nil
}

// VerificationParams is the candidate password and the stored hash Verify compares.
type VerificationParams struct {
	Password string
	Hash     string
}

// Verify reports whether the password matches the stored argon2id hash.
func Verify(params VerificationParams) (bool, error) {
	matches, err := argon2id.ComparePasswordAndHash(params.Password, params.Hash)
	if err != nil {
		return false, ErrEncodedHashUnreadable{Cause: err}
	}

	return matches, nil
}
