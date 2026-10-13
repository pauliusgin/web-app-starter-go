package primitive

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// MinPasswordLength is the shortest password accepted, in runes.
const MinPasswordLength = 8

// MaxPasswordLength is the longest password accepted, in runes.
const MaxPasswordLength = 64

// ValidatePassword checks that password is MinPasswordLength to MaxPasswordLength runes long and holds no whitespace, reporting the first failure.
func ValidatePassword(password string) error {
	passwordLength := utf8.RuneCountInString(password)
	if passwordLength < MinPasswordLength {
		return ErrPasswordTooShort{Length: passwordLength}
	}

	if passwordLength > MaxPasswordLength {
		return ErrPasswordTooLong{Length: passwordLength}
	}

	whitespaceIndex := strings.IndexFunc(password, unicode.IsSpace)
	if whitespaceIndex >= 0 {
		return ErrPasswordContainsWhitespace{Position: utf8.RuneCountInString(password[:whitespaceIndex])}
	}

	return nil
}
