package primitive

import (
	"strings"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "exactly the minimum", password: "correct1"},
		{name: "long passphrase", password: "correct-horse-battery-staple"},
		{name: "one character short", password: "short12", wantErr: ErrPasswordTooShort{Length: 7}},
		{name: "empty", password: "", wantErr: ErrPasswordTooShort{Length: 0}},
		{name: "length counted in runes not bytes", password: "päss", wantErr: ErrPasswordTooShort{Length: 4}},
		{name: "leading space", password: " correct-horse", wantErr: ErrPasswordContainsWhitespace{Position: 0}},
		{name: "space in the middle", password: "correct horse", wantErr: ErrPasswordContainsWhitespace{Position: 7}},
		{name: "trailing space", password: "correct-horse ", wantErr: ErrPasswordContainsWhitespace{Position: 13}},
		{name: "tab", password: "correct\thorse", wantErr: ErrPasswordContainsWhitespace{Position: 7}},
		{name: "non-breaking space", password: "correct horse", wantErr: ErrPasswordContainsWhitespace{Position: 7}},
		{name: "position counted in runes not bytes", password: "päss word", wantErr: ErrPasswordContainsWhitespace{Position: 4}},
		{name: "too short wins over whitespace", password: "a b", wantErr: ErrPasswordTooShort{Length: 3}},
		{name: "exactly the maximum", password: strings.Repeat("a", MaxPasswordLength)},
		{name: "one character long", password: strings.Repeat("a", MaxPasswordLength+1),
			wantErr: ErrPasswordTooLong{Length: MaxPasswordLength + 1}},
		{name: "maximum counted in runes not bytes", password: strings.Repeat("ä", MaxPasswordLength)},
		{name: "too long wins over whitespace", password: strings.Repeat("a b", MaxPasswordLength),
			wantErr: ErrPasswordTooLong{Length: 3 * MaxPasswordLength}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidatePassword(testCase.password)

			if err != testCase.wantErr {
				t.Fatalf("error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}
