// Package refreshtoken mints, rotates, revokes and purges the opaque refresh tokens clients hold.
package refreshtoken

import (
	"time"

	"github.com/google/uuid"
)

// Lifetime is how long an issued refresh token stays valid (30 days).
const Lifetime = 30 * 24 * time.Hour

// CleanUpThresholdInDays is how many days a revoked token is kept in the database before it is purged.
const CleanUpThresholdInDays = 7

// RefreshToken is a stored refresh token as callers see it, carrying the hash and never the secret.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt time.Time
}

// NewRefreshToken is a refresh token on its way in, the shape that carries no created timestamp.
type NewRefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
}

// IsRevoked reports whether the token has already been revoked.
func (t RefreshToken) IsRevoked() bool {
	return !t.RevokedAt.IsZero()
}

// IsExpired reports whether the token's expiry has passed at the given instant.
func (t RefreshToken) IsExpired(at time.Time) bool {
	return !at.Before(t.ExpiresAt)
}
