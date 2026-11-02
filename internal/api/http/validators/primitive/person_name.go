package primitive

import (
	"regexp"
	"unicode/utf8"
)

// MaxPersonNameLength is the longest first or last name accepted, in runes.
const MaxPersonNameLength = 50

// personNamePattern accepts Latin and Cyrillic letters, covering the English, Lithuanian and Russian alphabets
// plus neighbouring EU names (Müller, Łukasz); a single space or dash may only join two letter runs.
var personNamePattern = regexp.MustCompile(`^[\p{Latin}\p{Cyrillic}]+(?:[ -][\p{Latin}\p{Cyrillic}]+)*$`)

// ValidatePersonName checks an already trimmed, NFC-normalized name against the length and letter rules.
func ValidatePersonName(name string) error {
	nameLength := utf8.RuneCountInString(name)
	if nameLength > MaxPersonNameLength {
		return ErrPersonNameTooLong{Length: nameLength}
	}
	if !personNamePattern.MatchString(name) {
		return ErrPersonNameMalformed{Name: name}
	}

	return nil
}
