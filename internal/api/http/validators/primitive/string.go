package primitive

import (
	"strings"
	"unicode/utf8"
)

// MinimumStringLength is the shortest required string accepted, in runes.
const MinimumStringLength = 1

// RequiredString checks that input, ignoring surrounding whitespace, meets MinimumStringLength.
func RequiredString(input string) error {
	stringLength := utf8.RuneCountInString(strings.TrimSpace(input))
	if stringLength < MinimumStringLength {
		return ErrStringMissing{}
	}

	return nil
}
