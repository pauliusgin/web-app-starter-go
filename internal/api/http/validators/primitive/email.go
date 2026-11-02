// Package primitive validates single request values such as an email or a password.
package primitive

import (
	"net/mail"
	"strings"
)

// ValidateEmail checks that email is a bare address, with no display name or surrounding text.
func ValidateEmail(email string) error {
	parsedAddress, err := mail.ParseAddress(email)
	if err != nil {
		return ErrEmailInvalid{Email: email}
	}
	if parsedAddress.Name != "" || parsedAddress.Address != email {
		return ErrEmailInvalid{Email: email}
	}

	return nil
}

// NormalizeEmail trims and lowercases email, the form addresses are stored and compared in.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
