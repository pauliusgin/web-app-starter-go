// Package models holds the user shapes the write side stores and hands back.
package models

import (
	"time"

	"github.com/google/uuid"
)

// User is a stored user as callers see it, deliberately carrying no password hash.
type User struct {
	ID              uuid.UUID
	Email           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	EmailVerifiedAt time.Time
	SigninAt        time.Time
}

// IsEmailVerified reports whether the user has proven ownership of the email.
func (u User) IsEmailVerified() bool {
	return !u.EmailVerifiedAt.IsZero()
}

// NewUser is a user on its way in, the only shape in this package that carries a password hash.
type NewUser struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
}

// NewGoogleUser is a google-authenticated user on its way in, carrying no password.
type NewGoogleUser struct {
	ID            uuid.UUID
	Email         string
	GoogleSubject string
}

// NewGoogleUserParams is the google-verified email and google subject NewGoogle stores.
type NewGoogleUserParams struct {
	Email         string
	GoogleSubject string
}

// NewGoogle builds the google user row to insert, generating the id.
func NewGoogle(params NewGoogleUserParams) NewGoogleUser {
	return NewGoogleUser{
		ID:            uuid.New(),
		Email:         params.Email,
		GoogleSubject: params.GoogleSubject,
	}
}

// UserCredentials is the stored identity, password hash and google subject a sign-in or password change checks.
type UserCredentials struct {
	ID            uuid.UUID
	Email         string
	PasswordHash  string
	GoogleSubject string
}

// HasPassword reports whether the account can sign in with a password; google-only accounts cannot.
func (c UserCredentials) HasPassword() bool {
	return c.PasswordHash != ""
}

// HasGoogleAccount reports whether the account can sign in with google.
func (c UserCredentials) HasGoogleAccount() bool {
	return c.GoogleSubject != ""
}

// NewUserParams is the email and already-hashed password New stores.
type NewUserParams struct {
	Email        string
	PasswordHash string
}

// New builds the row to insert, generating the id; the email must arrive already validated.
func New(params NewUserParams) NewUser {
	return NewUser{
		ID:           uuid.New(),
		Email:        params.Email,
		PasswordHash: params.PasswordHash,
	}
}

const anonymizedEmailPrefix = "deleted-"

// AnonymizedEmail is the placeholder a deleted user's email is overwritten with, unique per user.
func AnonymizedEmail(id uuid.UUID) string {
	return anonymizedEmailPrefix + id.String()
}

// AnonymizedFirstName replaces the first name on a deleted user's profile.
const AnonymizedFirstName = "Deleted"

// AnonymizedLastName replaces the last name on a deleted user's profile.
const AnonymizedLastName = "User"

// SoftDeleteUserParams is the user to soft-delete and the placeholders replacing its personal details.
type SoftDeleteUserParams struct {
	ID                  uuid.UUID
	AnonymizedEmail     string
	AnonymizedFirstName string
	AnonymizedLastName  string
}

// ChangeEmailParams is the user whose email changes and the already-verified address it becomes.
type ChangeEmailParams struct {
	ID    uuid.UUID
	Email string
}

// ChangePasswordParams is the user whose password changes and the already-hashed password it becomes.
type ChangePasswordParams struct {
	ID           uuid.UUID
	PasswordHash string
}

// AddPasswordParams is the google-only user gaining a password and the already-hashed password it gets.
type AddPasswordParams struct {
	ID           uuid.UUID
	PasswordHash string
}

// AddGoogleAccountParams is the password user gaining a google account and the google subject it gets.
type AddGoogleAccountParams struct {
	ID            uuid.UUID
	GoogleSubject string
}
