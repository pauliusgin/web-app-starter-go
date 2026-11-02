package input

import (
	"errors"
	"strings"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

const testEmail = "accountant@example.com"

const testPassword = "correct-horse-battery-staple"

func TestValidateSignUpInput(t *testing.T) {
	cases := []struct {
		name          string
		input         SignUpInput
		want          SignUpInput
		wantErr       bool
		wantErrTarget any
	}{
		{
			name:  "valid email and password",
			input: SignUpInput{Email: testEmail, Password: testPassword},
			want:  SignUpInput{Email: testEmail, Password: testPassword},
		},
		{
			name:  "email is trimmed and lowercased, password kept verbatim",
			input: SignUpInput{Email: " Accountant@Example.COM ", Password: testPassword},
			want:  SignUpInput{Email: testEmail, Password: testPassword},
		},
		{
			name:          "invalid email",
			input:         SignUpInput{Email: "bob", Password: testPassword},
			wantErr:       true,
			wantErrTarget: &primitive.ErrEmailInvalid{},
		},
		{
			name:          "short password",
			input:         SignUpInput{Email: testEmail, Password: "short12"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordTooShort{},
		},
		{
			name:          "long password",
			input:         SignUpInput{Email: testEmail, Password: strings.Repeat("a", primitive.MaxPasswordLength+1)},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordTooLong{},
		},
		{
			name:          "password with whitespace",
			input:         SignUpInput{Email: testEmail, Password: "correct horse"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordContainsWhitespace{},
		},
		{
			name:          "both invalid reports the email first",
			input:         SignUpInput{Email: "bob", Password: "short12"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrEmailInvalid{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ValidateSignUpInput(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateSignUpInput: %v", err)
				}
				if got != testCase.want {
					t.Fatalf("ValidateSignUpInput = %+v, want %+v", got, testCase.want)
				}
				return
			}

			if !errors.As(err, testCase.wantErrTarget) {
				t.Fatalf("error = %v, want %T", err, testCase.wantErrTarget)
			}
		})
	}
}
