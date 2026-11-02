// Package challenge mints the verification challenges a user proves control of a target with.
package challenge

import (
	"time"

	"github.com/google/uuid"
)

// DefaultLifetime is how long a generated challenge stays valid unless its purpose sets its own lifetime.
const DefaultLifetime = 5 * time.Minute

// PasswordResetLifetime is how long a password reset link stays valid; the user needs time to pick a new password.
const PasswordResetLifetime = 15 * time.Minute

var lifetimeByPurpose = map[Purpose]time.Duration{
	PurposePasswordReset: PasswordResetLifetime,
}

// TargetType names the kind of target a challenge is sent to.
type TargetType string

const (
	// TargetTypeEmail sends the challenge to an email address.
	TargetTypeEmail TargetType = "email"
	// TargetTypePhone sends the challenge to a phone number.
	TargetTypePhone TargetType = "phone"
)

// Purpose names what a challenge is used for.
type Purpose string

const (
	// PurposeEmailVerification proves the user owns the email address on the account.
	PurposeEmailVerification Purpose = "email_verification"
	// PurposePhoneVerification proves the user owns the phone number on the account.
	PurposePhoneVerification Purpose = "phone_verification"
	// PurposePasswordReset lets the user set a new password without the current one.
	PurposePasswordReset Purpose = "password_reset"
)

// LifetimeFor returns how long a challenge minted for the purpose stays valid.
func LifetimeFor(purpose Purpose) time.Duration {
	lifetime, overridden := lifetimeByPurpose[purpose]
	if !overridden {
		return DefaultLifetime
	}

	return lifetime
}

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
