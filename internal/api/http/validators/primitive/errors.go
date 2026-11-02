package primitive

import "fmt"

const emailExpectationText = "expected a plain address such as accountant@example.com, without a display name"

// ErrEmailInvalid reports an email that is not a plain address.
type ErrEmailInvalid struct {
	Email string
}

func (e ErrEmailInvalid) Error() string {
	return fmt.Sprintf("invalid email %q: %s", e.Email, emailExpectationText)
}

// ErrPasswordTooShort reports a password shorter than MinPasswordLength runes.
type ErrPasswordTooShort struct {
	Length int
}

func (e ErrPasswordTooShort) Error() string {
	return fmt.Sprintf("password is %d characters long: expected at least %d",
		e.Length, MinPasswordLength)
}

// ErrPasswordTooLong reports a password longer than MaxPasswordLength runes.
type ErrPasswordTooLong struct {
	Length int
}

func (e ErrPasswordTooLong) Error() string {
	return fmt.Sprintf("password is %d characters long: expected at most %d",
		e.Length, MaxPasswordLength)
}

// ErrPasswordContainsWhitespace reports a password with whitespace at the given rune position.
type ErrPasswordContainsWhitespace struct {
	Position int
}

func (e ErrPasswordContainsWhitespace) Error() string {
	return fmt.Sprintf("password contains whitespace at position %d: expected no spaces, tabs or line breaks", e.Position)
}

// ErrPasswordMissing reports a request that arrived without a password.
type ErrPasswordMissing struct{}

func (e ErrPasswordMissing) Error() string {
	return "password is missing: expected a non-empty password"
}

// ErrGoogleIDTokenMissing reports a google signup or sign-in that arrived without an ID token.
type ErrGoogleIDTokenMissing struct{}

func (e ErrGoogleIDTokenMissing) Error() string {
	return "google ID token is missing: expected the credential google returned to the frontend"
}

// ErrPasswordResetTokenMissing reports a password reset that arrived without the token its link carried.
type ErrPasswordResetTokenMissing struct{}

func (e ErrPasswordResetTokenMissing) Error() string {
	return "password reset token is missing: expected the token the emailed reset link carried"
}

// ErrStringMissing reports a request that arrived without a string.
type ErrStringMissing struct{}

func (e ErrStringMissing) Error() string {
	return "input is missing: expected a non-empty string"
}

// ErrFirstNameMissing reports a request that arrived without a first name.
type ErrFirstNameMissing struct{}

func (e ErrFirstNameMissing) Error() string {
	return "first name is missing: expected a non-empty first name"
}

// ErrLastNameMissing reports a request that arrived without a last name.
type ErrLastNameMissing struct{}

func (e ErrLastNameMissing) Error() string {
	return "last name is missing: expected a non-empty last name"
}

const personNameExpectationText = "expected Latin or Cyrillic letters, joined by single spaces or dashes"

// ErrPersonNameTooLong reports a name longer than MaxPersonNameLength runes.
type ErrPersonNameTooLong struct {
	Length int
}

func (e ErrPersonNameTooLong) Error() string {
	return fmt.Sprintf("name is %d characters long: expected at most %d", e.Length, MaxPersonNameLength)
}

// ErrPersonNameMalformed reports a name with characters or separators outside the person-name rules.
type ErrPersonNameMalformed struct {
	Name string
}

func (e ErrPersonNameMalformed) Error() string {
	return fmt.Sprintf("name %q is malformed: %s", e.Name, personNameExpectationText)
}

// ErrFirstNameInvalid reports a first name that breaks the person-name rules.
type ErrFirstNameInvalid struct {
	Cause error
}

func (e ErrFirstNameInvalid) Error() string {
	return fmt.Sprintf("first name is invalid: %s", e.Cause)
}

func (e ErrFirstNameInvalid) Unwrap() error {
	return e.Cause
}

// ErrLastNameInvalid reports a last name that breaks the person-name rules.
type ErrLastNameInvalid struct {
	Cause error
}

func (e ErrLastNameInvalid) Error() string {
	return fmt.Sprintf("last name is invalid: %s", e.Cause)
}

func (e ErrLastNameInvalid) Unwrap() error {
	return e.Cause
}

// ErrPictureUploadIDMalformed reports a picture upload id that is neither blank nor a uuid.
type ErrPictureUploadIDMalformed struct {
	RawPictureUploadID string
}

func (e ErrPictureUploadIDMalformed) Error() string {
	return fmt.Sprintf("picture upload id %q is malformed: expected a uuid, or empty for no picture", e.RawPictureUploadID)
}

// ErrPageNumberInvalid reports a page number that is not a whole number between 1 and MaxPageNumber.
type ErrPageNumberInvalid struct {
	RawPage string
}

func (e ErrPageNumberInvalid) Error() string {
	return fmt.Sprintf("page %q: expected a whole number from 1 to %d", e.RawPage, MaxPageNumber)
}

// ErrPageLimitInvalid reports a per-page limit that is not a whole number between 1 and MaxPageLimit.
type ErrPageLimitInvalid struct {
	RawLimit string
}

func (e ErrPageLimitInvalid) Error() string {
	return fmt.Sprintf("limit %q: expected a whole number from 1 to %d", e.RawLimit, MaxPageLimit)
}
