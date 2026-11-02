package primitive

import (
	"errors"
	"strings"
	"testing"
)

func TestValidatePersonName(t *testing.T) {
	cases := []struct {
		name       string
		personName string
		wantErr    error
	}{
		{name: "single word", personName: "John"},
		{name: "space between words", personName: "John Smith"},
		{name: "dash between words", personName: "Taylor-Joy"},
		{name: "lithuanian letters", personName: "Jonas Žukauskas"},
		{name: "russian letters", personName: "Пётр Иванов"},
		{name: "other latin letters", personName: "Müller"},
		{name: "exactly the maximum", personName: strings.Repeat("ž", MaxPersonNameLength)},
		{name: "leading dash", personName: "-Taylor Joy", wantErr: ErrPersonNameMalformed{Name: "-Taylor Joy"}},
		{name: "trailing dash", personName: "Taylor-", wantErr: ErrPersonNameMalformed{Name: "Taylor-"}},
		{name: "leading space", personName: " John", wantErr: ErrPersonNameMalformed{Name: " John"}},
		{name: "double space", personName: "John  Smith", wantErr: ErrPersonNameMalformed{Name: "John  Smith"}},
		{name: "double dash", personName: "Anna--Maria", wantErr: ErrPersonNameMalformed{Name: "Anna--Maria"}},
		{name: "dash then space", personName: "John- Smith", wantErr: ErrPersonNameMalformed{Name: "John- Smith"}},
		{name: "apostrophe", personName: "O'Brien", wantErr: ErrPersonNameMalformed{Name: "O'Brien"}},
		{name: "digit", personName: "John3", wantErr: ErrPersonNameMalformed{Name: "John3"}},
		{name: "emoji", personName: "Ada😀", wantErr: ErrPersonNameMalformed{Name: "Ada😀"}},
		{name: "tab separator", personName: "John\tSmith", wantErr: ErrPersonNameMalformed{Name: "John\tSmith"}},
		{name: "other script", personName: "张伟", wantErr: ErrPersonNameMalformed{Name: "张伟"}},
		{name: "decomposed letter", personName: "Žė", wantErr: ErrPersonNameMalformed{Name: "Žė"}},
		{name: "empty", personName: "", wantErr: ErrPersonNameMalformed{Name: ""}},
		{name: "one character too long", personName: strings.Repeat("a", MaxPersonNameLength+1),
			wantErr: ErrPersonNameTooLong{Length: MaxPersonNameLength + 1}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidatePersonName(testCase.personName)

			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}
