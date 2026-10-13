package primitive

import "fmt"

const emailExpectationText = "expected a plain address such as accountant@example.com, without a display name"

// ErrInvalidEmail reports an email that is not a plain address.
type ErrInvalidEmail struct {
	Email string
}

func (e ErrInvalidEmail) Error() string {
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
