// Package challenge mints the verification challenges a user proves control of a target with.
package challenge

import (
	"time"

	"github.com/google/uuid"
)

// Lifetime is how long a generated challenge stays valid.
const Lifetime = 5 * time.Minute

// TargetType names the kind of target a challenge is sent to.
type TargetType string

const (
	TargetTypeEmail TargetType = "email"
	TargetTypePhone TargetType = "phone"
)

// Purpose names what a challenge is used for.
type Purpose string

const (
	PurposeEmailVerification Purpose = "email_verification"
	PurposePhoneVerification Purpose = "phone_verification"
	PasswordReset            Purpose = "password_reset"
)

// Challenge is a stored challenge as callers see it, carrying the hash and never the secret.
type Challenge struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	TargetType  TargetType
	Target      string
	Purpose     Purpose
	TokenHash   string
	Code        string
	Provider    string
	ReferenceID string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	UsedAt      time.Time
}

// NewChallenge is a challenge on its way in, the shape that carries no created timestamp.
type NewChallenge struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	TargetType  TargetType
	Target      string
	Purpose     Purpose
	TokenHash   string
	Code        string
	Provider    string
	ReferenceID string
	ExpiresAt   time.Time
	UsedAt      time.Time
}

// IsUsed reports whether the challenge has already been consumed.
func (c Challenge) IsUsed() bool {
	return !c.UsedAt.IsZero()
}

// IsExpired reports whether the challenge's expiry has passed at the given instant.
func (c Challenge) IsExpired(at time.Time) bool {
	return !at.Before(c.ExpiresAt)
}
