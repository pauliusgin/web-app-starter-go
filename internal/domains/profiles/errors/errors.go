// Package errors holds the failures the profile use cases, ports and adapters report.
package errors

import (
	"fmt"

	"github.com/google/uuid"
)

const profileAlreadyExistsText = "profile already exists"
const profileNotFoundText = "profile not found"
const saveProfileFailedText = "failed saving the profile"
const updateProfileFailedText = "failed updating the profile"
const findProfileFailedText = "failed finding the profile"

// ErrProfileAlreadyExists reports a profile creation for a user who already has one.
type ErrProfileAlreadyExists struct {
	UserID uuid.UUID
}

func (e ErrProfileAlreadyExists) Error() string {
	return fmt.Sprintf("%s for user %s", profileAlreadyExistsText, e.UserID)
}

// ErrProfileNotFound reports a profile lookup or change for a user who has no live profile.
type ErrProfileNotFound struct {
	UserID uuid.UUID
}

func (e ErrProfileNotFound) Error() string {
	return fmt.Sprintf("%s for user %s", profileNotFoundText, e.UserID)
}

// ErrSaveProfileFailed reports a profile row the database failed to insert.
type ErrSaveProfileFailed struct {
	UserID uuid.UUID
	Cause  error
}

func (e ErrSaveProfileFailed) Error() string {
	return fmt.Sprintf("%s for user %s: %s", saveProfileFailedText, e.UserID, e.Cause)
}

func (e ErrSaveProfileFailed) Unwrap() error {
	return e.Cause
}

// ErrUpdateProfileFailed reports a profile row the database failed to update.
type ErrUpdateProfileFailed struct {
	UserID uuid.UUID
	Cause  error
}

func (e ErrUpdateProfileFailed) Error() string {
	return fmt.Sprintf("%s for user %s: %s", updateProfileFailedText, e.UserID, e.Cause)
}

func (e ErrUpdateProfileFailed) Unwrap() error {
	return e.Cause
}

// ErrFindProfileFailed reports a profile lookup the database could not complete.
type ErrFindProfileFailed struct {
	UserID uuid.UUID
	Cause  error
}

func (e ErrFindProfileFailed) Error() string {
	return fmt.Sprintf("%s for user %s: %s", findProfileFailedText, e.UserID, e.Cause)
}

func (e ErrFindProfileFailed) Unwrap() error {
	return e.Cause
}
