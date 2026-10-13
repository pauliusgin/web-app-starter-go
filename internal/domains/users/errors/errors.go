// Package errors holds the failures the user use cases and repositories report.
package errors

import (
	"fmt"

	"github.com/google/uuid"
)

const emailTakenText = "expected an email that does not already have an account"
const userNotFoundText = "user not found"
const emailLookupFailedText = "failed checking whether the email already has an account"
const idLookupFailedText = "failed finding the user by id"
const passwordHashingFailedText = "failed hashing the password"
const saveUserFailedText = "failed saving the user"
const markEmailVerifiedFailedText = "failed marking the user's email verified"
const invalidCredentialsText = "expected an email and password matching an existing account"
const passwordVerificationFailedText = "failed checking the password against the stored hash"
const markSignedInFailedText = "failed stamping the user's sign-in"
const verificationLinkInvalidText = "expected a verification link that is unused and not expired"
const userDeletionForbiddenText = "expected users to delete only their own account"
const softDeleteUserFailedText = "failed soft-deleting the user"
const emailUnchangedText = "expected a new email different from the current one"
const changeEmailFailedText = "failed changing the user's email"
const currentPasswordIncorrectText = "expected the current password to match the stored one"
const passwordUnchangedText = "expected a new password different from the current one"
const changePasswordFailedText = "failed changing the user's password"

// GenerateAuthChallengeFailedText is logged by the signup use case when the challenge call fails.
const GenerateAuthChallengeFailedText = "failed generating the email verification challenge"

// ChallengeGeneratorMissingText is logged by the signup use case when no generator was wired.
const ChallengeGeneratorMissingText = "no challenge generator is wired, so the email is unverified"

// ErrEmailTaken reports a signup for an email that already has an account.
type ErrEmailTaken struct {
	Email string
}

func (e ErrEmailTaken) Error() string {
	return fmt.Sprintf("email %q is already taken: %s", e.Email, emailTakenText)
}

// ErrUserNotFound reports a lookup that matched no account.
type ErrUserNotFound struct {
	ID    uuid.UUID
	Email string
}

func (e ErrUserNotFound) Error() string {
	return fmt.Sprintf("%s", userNotFoundText)
}

// ErrIDLookupFailed reports a lookup by id the database could not complete.
type ErrIDLookupFailed struct {
	Cause error
}

func (e ErrIDLookupFailed) Error() string {
	return fmt.Sprintf("%s: %s", idLookupFailedText, e.Cause)
}

func (e ErrIDLookupFailed) Unwrap() error {
	return e.Cause
}

// ErrEmailLookupFailed reports a lookup by email the database could not complete.
type ErrEmailLookupFailed struct {
	Cause error
}

func (e ErrEmailLookupFailed) Error() string {
	return fmt.Sprintf("%s: %s", emailLookupFailedText, e.Cause)
}

func (e ErrEmailLookupFailed) Unwrap() error {
	return e.Cause
}

// ErrPasswordHashingFailed reports a password the hasher could not hash.
type ErrPasswordHashingFailed struct {
	Cause error
}

func (e ErrPasswordHashingFailed) Error() string {
	return fmt.Sprintf("%s: %s", passwordHashingFailedText, e.Cause)
}

func (e ErrPasswordHashingFailed) Unwrap() error {
	return e.Cause
}

// ErrSaveUserFailed reports a user the database refused to store.
type ErrSaveUserFailed struct {
	Cause error
}

func (e ErrSaveUserFailed) Error() string {
	return fmt.Sprintf("%s: %s", saveUserFailedText, e.Cause)
}

func (e ErrSaveUserFailed) Unwrap() error {
	return e.Cause
}

// ErrMarkEmailVerifiedFailed reports a verification stamp the database could not write.
type ErrMarkEmailVerifiedFailed struct {
	Cause error
}

func (e ErrMarkEmailVerifiedFailed) Error() string {
	return fmt.Sprintf("%s: %s", markEmailVerifiedFailedText, e.Cause)
}

func (e ErrMarkEmailVerifiedFailed) Unwrap() error {
	return e.Cause
}

// ErrVerificationLinkInvalid reports a link whose challenge is used, expired or unknown.
type ErrVerificationLinkInvalid struct{}

func (e ErrVerificationLinkInvalid) Error() string {
	return fmt.Sprintf("invalid verification link: %s", verificationLinkInvalidText)
}

// ErrInvalidCredentials reports a sign-in whose email is unknown or whose password does not match.
type ErrInvalidCredentials struct{}

func (e ErrInvalidCredentials) Error() string {
	return fmt.Sprintf("invalid credentials: %s", invalidCredentialsText)
}

// ErrPasswordVerificationFailed reports a stored password hash the verifier could not read.
type ErrPasswordVerificationFailed struct {
	Cause error
}

func (e ErrPasswordVerificationFailed) Error() string {
	return fmt.Sprintf("%s: %s", passwordVerificationFailedText, e.Cause)
}

func (e ErrPasswordVerificationFailed) Unwrap() error {
	return e.Cause
}

// ErrMarkSignedInFailed reports a sign-in stamp the database could not write.
type ErrMarkSignedInFailed struct {
	Cause error
}

func (e ErrMarkSignedInFailed) Error() string {
	return fmt.Sprintf("%s: %s", markSignedInFailedText, e.Cause)
}

func (e ErrMarkSignedInFailed) Unwrap() error {
	return e.Cause
}

// ErrUserDeletionForbidden reports an attempt to delete an account other than the caller's own.
type ErrUserDeletionForbidden struct {
	ActingUserID uuid.UUID
	TargetUserID uuid.UUID
}

func (e ErrUserDeletionForbidden) Error() string {
	return fmt.Sprintf("user %s cannot delete user %s: %s", e.ActingUserID, e.TargetUserID, userDeletionForbiddenText)
}

// ErrSoftDeleteUserFailed reports a soft delete the database could not write.
type ErrSoftDeleteUserFailed struct {
	Cause error
}

func (e ErrSoftDeleteUserFailed) Error() string {
	return fmt.Sprintf("%s: %s", softDeleteUserFailedText, e.Cause)
}

func (e ErrSoftDeleteUserFailed) Unwrap() error {
	return e.Cause
}

// ErrEmailUnchanged reports an email change request naming the address the account already has.
type ErrEmailUnchanged struct {
	Email string
}

func (e ErrEmailUnchanged) Error() string {
	return fmt.Sprintf("email %q is unchanged: %s", e.Email, emailUnchangedText)
}

// ErrChangeEmailFailed reports an email change the database could not write.
type ErrChangeEmailFailed struct {
	Cause error
}

func (e ErrChangeEmailFailed) Error() string {
	return fmt.Sprintf("%s: %s", changeEmailFailedText, e.Cause)
}

func (e ErrChangeEmailFailed) Unwrap() error {
	return e.Cause
}

// ErrCurrentPasswordIncorrect reports a password change whose current password does not match.
type ErrCurrentPasswordIncorrect struct{}

func (e ErrCurrentPasswordIncorrect) Error() string {
	return fmt.Sprintf("current password incorrect: %s", currentPasswordIncorrectText)
}

// ErrPasswordUnchanged reports a password change whose new password equals the current one.
type ErrPasswordUnchanged struct{}

func (e ErrPasswordUnchanged) Error() string {
	return fmt.Sprintf("password unchanged: %s", passwordUnchangedText)
}

// ErrChangePasswordFailed reports a password change the database could not write.
type ErrChangePasswordFailed struct {
	Cause error
}

func (e ErrChangePasswordFailed) Error() string {
	return fmt.Sprintf("%s: %s", changePasswordFailedText, e.Cause)
}

func (e ErrChangePasswordFailed) Unwrap() error {
	return e.Cause
}
