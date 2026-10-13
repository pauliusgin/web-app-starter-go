// Package primitive validates single request values such as an email or a password.
package primitive

import "net/mail"

// ValidateEmail checks that email is a bare address, with no display name or surrounding text.
func ValidateEmail(email string) error {
	parsedAddress, err := mail.ParseAddress(email)
	if err != nil {
		return ErrInvalidEmail{Email: email}
	}
	if parsedAddress.Name != "" || parsedAddress.Address != email {
		return ErrInvalidEmail{Email: email}
	}

	return nil
}
