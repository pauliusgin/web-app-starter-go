// Package repositories declares the user persistence port the write side depends on.
package repositories

import (
	"context"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"

	"github.com/google/uuid"
)

// UserRepository is the user persistence port the domain depends on.
type UserRepository interface {
	Save(ctx context.Context, newUser models.NewUser) (models.User, error)

	// FindUserByID loads the write model for internal checks such as existence or verification state.
	FindUserByID(ctx context.Context, id uuid.UUID) (models.User, error)

	// FindUserByEmail loads the write model for internal checks such as email availability.
	FindUserByEmail(ctx context.Context, email string) (models.User, error)

	// MarkEmailVerified stamps the instant the user proved ownership of their current email.
	MarkEmailVerified(ctx context.Context, id uuid.UUID) (models.User, error)

	// FindUserCredentialsByEmail loads the password hash, and is meant only for checking a sign-in.
	FindUserCredentialsByEmail(ctx context.Context, email string) (models.UserCredentials, error)

	// FindUserCredentialsByID loads the password hash, and is meant only for checking a password change.
	FindUserCredentialsByID(ctx context.Context, id uuid.UUID) (models.UserCredentials, error)

	// ChangePassword swaps in an already-hashed new password.
	ChangePassword(ctx context.Context, params models.ChangePasswordParams) error

	// ChangeEmail swaps in an already-verified email and stamps it verified.
	ChangeEmail(ctx context.Context, params models.ChangeEmailParams) (models.User, error)

	// MarkSignedIn stamps the instant the user last signed in.
	MarkSignedIn(ctx context.Context, id uuid.UUID) (models.User, error)

	// SoftDeleteUser stamps deleted_at on the user and its profile, swapping in the anonymized placeholders.
	SoftDeleteUser(ctx context.Context, params models.SoftDeleteUserParams) error
}
