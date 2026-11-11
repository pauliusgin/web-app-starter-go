// Package models holds the user shapes the read side returns.
package models

import (
	"time"

	"github.com/google/uuid"
)

// UserReadModel is a returned user as clients see it; a nil pointer is a value the user does not have yet.
// It is also the HTTP response body: json keys are API contract, and a missing detail encodes as null.
type UserReadModel struct {
	ID              uuid.UUID      `json:"id"`
	Email           string         `json:"email"`
	CreatedAt       time.Time      `json:"createdAt"`
	EmailVerifiedAt *time.Time     `json:"emailVerifiedAt"`
	SigninAt        *time.Time     `json:"signinAt"`
	FirstName       *string        `json:"firstName"`
	LastName        *string        `json:"lastName"`
	SignInMethods   []SignInMethod `json:"signInMethods"`
}

// SignInMethod is one way a user can prove who they are when signing in.
type SignInMethod string

const (
	// SignInMethodEmailPassword is signing in with the account email and a stored password.
	SignInMethodEmailPassword SignInMethod = "email_password"
	// SignInMethodGoogle is signing in with a Google ID token matching the stored google subject.
	SignInMethodGoogle SignInMethod = "google"
)

// SignInMethodsParams is which credentials a user has stored.
type SignInMethodsParams struct {
	HasPassword      bool
	HasGoogleSubject bool
}

// NewSignInMethods lists the sign-in methods the stored credentials allow, email_password first;
// never nil, so a client always receives an array.
func NewSignInMethods(params SignInMethodsParams) []SignInMethod {
	methods := []SignInMethod{}
	if params.HasPassword {
		methods = append(methods, SignInMethodEmailPassword)
	}
	if params.HasGoogleSubject {
		methods = append(methods, SignInMethodGoogle)
	}

	return methods
}
