// Package errors holds the failures the user use cases and repositories report.
package errors

import (
	"fmt"

	"github.com/google/uuid"
)

const userNotFoundText = "user not found"
const findUserFailedText = "failed finding the user"
const passwordHashingFailedText = "failed hashing the password"
const saveUserFailedText = "failed saving the user"
const markEmailVerifiedFailedText = "failed marking the user's email verified"
const invalidCredentialsText = "invalid credentials"
const passwordVerificationFailedText = "failed checking the password against the stored hash"
const markSignedInFailedText = "failed stamping the user's sign-in"
const verificationLinkInvalidText = "expected a verification link that is unused and not expired"
const passwordResetLinkInvalidText = "expected a password reset link that is unused, not expired and issued for the account's current email"
const softDeleteUserFailedText = "failed soft-deleting the user"
const emailUnchangedText = "expected a new email different from the current one"
const changeEmailFailedText = "failed changing the user's email"
const currentPasswordIncorrectText = "expected the current password to match the stored one"
const passwordUnchangedText = "expected a new password different from the current one"
const changePasswordFailedText = "failed changing the user's password"
const emailAlreadyVerifiedText = "expected an email that is not verified yet"
const emailTakenText = "expected an email that does not already have an account"
const googleEmailUnverifiedText = "expected a google account whose email google has verified"
const googleAccountTakenText = "expected a google account that is not signed up yet"
const googleAccountNotFoundText = "expected a google account that is signed up"
const googleAccountNotLinkedText = "expected the email's account to be signed up with google, not a password"
const passwordNotSetText = "expected an account that signs in with a password"
const passwordAlreadySetText = "expected an account that has no password yet"
const googleAccountNotConnectedText = "expected an account signed up with google"
const addPasswordFailedText = "failed adding the user's password"
const googleAccountAlreadyConnectedText = "expected an account no google account is connected to yet"
const googleEmailMismatchText = "expected a google account with the same email as the account"
const addGoogleAccountFailedText = "failed adding the user's google account"

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
	ID            uuid.UUID
	Email         string
	GoogleSubject string
}

func (e ErrUserNotFound) Error() string {
	return fmt.Sprintf("%s (id %s, email %q, google subject %q)", userNotFoundText, e.ID, e.Email, e.GoogleSubject)
}

// ErrFindUserFailed reports a lookup by id, email or google subject the database could not complete.
type ErrFindUserFailed struct {
	ID            uuid.UUID
	Email         string
	GoogleSubject string
	Cause         error
}

func (e ErrFindUserFailed) Error() string {
	return fmt.Sprintf("%s (id %s, email %q, google subject %q): %s",
		findUserFailedText, e.ID, e.Email, e.GoogleSubject, e.Cause)
}

func (e ErrFindUserFailed) Unwrap() error {
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
	Email string
	Cause error
}

func (e ErrSaveUserFailed) Error() string {
	return fmt.Sprintf("%s for email %q: %s", saveUserFailedText, e.Email, e.Cause)
}

func (e ErrSaveUserFailed) Unwrap() error {
	return e.Cause
}

// ErrMarkEmailVerifiedFailed reports a verification stamp the database could not write.
type ErrMarkEmailVerifiedFailed struct {
	UserID uuid.UUID
	Cause  error
}

func (e ErrMarkEmailVerifiedFailed) Error() string {
	return fmt.Sprintf("%s for user %s: %s", markEmailVerifiedFailedText, e.UserID, e.Cause)
}

func (e ErrMarkEmailVerifiedFailed) Unwrap() error {
	return e.Cause
}

// ErrVerificationLinkInvalid reports a link whose challenge is used, expired or unknown.
type ErrVerificationLinkInvalid struct{}

func (e ErrVerificationLinkInvalid) Error() string {
	return fmt.Sprintf("invalid verification link: %s", verificationLinkInvalidText)
}

// ErrPasswordResetLinkInvalid reports a reset link that is unknown, spent, expired or no longer fits its account.
type ErrPasswordResetLinkInvalid struct{}

func (e ErrPasswordResetLinkInvalid) Error() string {
	return fmt.Sprintf("invalid password reset link: %s", passwordResetLinkInvalidText)
}

// ErrCredentialsInvalid reports a sign-in whose email is unknown or whose password does not match.
type ErrCredentialsInvalid struct{}

func (e ErrCredentialsInvalid) Error() string {
	return invalidCredentialsText
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
	UserID uuid.UUID
	Cause  error
}

func (e ErrMarkSignedInFailed) Error() string {
	return fmt.Sprintf("%s for user %s: %s", markSignedInFailedText, e.UserID, e.Cause)
}

func (e ErrMarkSignedInFailed) Unwrap() error {
	return e.Cause
}

// ErrSoftDeleteUserFailed reports a soft delete the database could not write.
type ErrSoftDeleteUserFailed struct {
	UserID uuid.UUID
	Cause  error
}

func (e ErrSoftDeleteUserFailed) Error() string {
	return fmt.Sprintf("%s for user %s: %s", softDeleteUserFailedText, e.UserID, e.Cause)
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
	UserID uuid.UUID
	Cause  error
}

func (e ErrChangeEmailFailed) Error() string {
	return fmt.Sprintf("%s for user %s: %s", changeEmailFailedText, e.UserID, e.Cause)
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
	UserID uuid.UUID
	Cause  error
}

func (e ErrChangePasswordFailed) Error() string {
	return fmt.Sprintf("%s for user %s: %s", changePasswordFailedText, e.UserID, e.Cause)
}

func (e ErrChangePasswordFailed) Unwrap() error {
	return e.Cause
}

// ErrEmailAlreadyVerified reports a verification email asked for an address the account already verified.
type ErrEmailAlreadyVerified struct {
	Email string
}

func (e ErrEmailAlreadyVerified) Error() string {
	return fmt.Sprintf("email %q is already verified: %s", e.Email, emailAlreadyVerifiedText)
}

// ErrGoogleEmailUnverified reports a google identity whose email google has not verified.
type ErrGoogleEmailUnverified struct {
	Email string
}

func (e ErrGoogleEmailUnverified) Error() string {
	return fmt.Sprintf("google email %q is unverified: %s", e.Email, googleEmailUnverifiedText)
}

// ErrGoogleAccountTaken reports a google signup for a google account that already has a user.
type ErrGoogleAccountTaken struct {
	GoogleSubject string
}

func (e ErrGoogleAccountTaken) Error() string {
	return fmt.Sprintf("google account %q is already signed up: %s", e.GoogleSubject, googleAccountTakenText)
}

// ErrGoogleAccountNotFound reports a google sign-in for a google account and email no user holds.
type ErrGoogleAccountNotFound struct {
	GoogleSubject string
}

func (e ErrGoogleAccountNotFound) Error() string {
	return fmt.Sprintf("google account %q not found: %s", e.GoogleSubject, googleAccountNotFoundText)
}

// ErrGoogleAccountNotLinked reports a google sign-in whose email belongs to a password account.
type ErrGoogleAccountNotLinked struct {
	Email string
}

func (e ErrGoogleAccountNotLinked) Error() string {
	return fmt.Sprintf("email %q is not linked to google: %s", e.Email, googleAccountNotLinkedText)
}

// ErrPasswordNotSet reports a password change for an account that signs in with google only.
type ErrPasswordNotSet struct {
	UserID uuid.UUID
}

func (e ErrPasswordNotSet) Error() string {
	return fmt.Sprintf("user %s has no password: %s", e.UserID, passwordNotSetText)
}

// ErrPasswordAlreadySet reports a password addition for an account that already signs in with a password.
type ErrPasswordAlreadySet struct {
	UserID uuid.UUID
}

func (e ErrPasswordAlreadySet) Error() string {
	return fmt.Sprintf("user %s already has a password: %s", e.UserID, passwordAlreadySetText)
}

// ErrGoogleAccountNotConnected reports a password addition for an account no google account is connected to.
type ErrGoogleAccountNotConnected struct {
	UserID uuid.UUID
}

func (e ErrGoogleAccountNotConnected) Error() string {
	return fmt.Sprintf("user %s has no google account: %s", e.UserID, googleAccountNotConnectedText)
}

// ErrAddPasswordFailed reports a password addition the database could not write.
type ErrAddPasswordFailed struct {
	UserID uuid.UUID
	Cause  error
}

func (e ErrAddPasswordFailed) Error() string {
	return fmt.Sprintf("%s for user %s: %s", addPasswordFailedText, e.UserID, e.Cause)
}

func (e ErrAddPasswordFailed) Unwrap() error {
	return e.Cause
}

// ErrGoogleAccountAlreadyConnected reports a google account addition for an account that already has one.
type ErrGoogleAccountAlreadyConnected struct {
	UserID uuid.UUID
}

func (e ErrGoogleAccountAlreadyConnected) Error() string {
	return fmt.Sprintf("user %s already has a google account: %s", e.UserID, googleAccountAlreadyConnectedText)
}

// ErrGoogleEmailMismatch reports a google account addition whose google email differs from the account's.
type ErrGoogleEmailMismatch struct {
	UserID      uuid.UUID
	GoogleEmail string
}

func (e ErrGoogleEmailMismatch) Error() string {
	return fmt.Sprintf("google email %q does not match user %s: %s", e.GoogleEmail, e.UserID, googleEmailMismatchText)
}

// ErrAddGoogleAccountFailed reports a google account addition the database could not write.
type ErrAddGoogleAccountFailed struct {
	UserID uuid.UUID
	Cause  error
}

func (e ErrAddGoogleAccountFailed) Error() string {
	return fmt.Sprintf("%s for user %s: %s", addGoogleAccountFailedText, e.UserID, e.Cause)
}

func (e ErrAddGoogleAccountFailed) Unwrap() error {
	return e.Cause
}
