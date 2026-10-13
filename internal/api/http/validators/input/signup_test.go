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
		wantErr       bool
		wantErrTarget any
	}{
		{
			name:  "valid email and password",
			input: SignUpInput{Email: testEmail, Password: testPassword},
		},
		{
			name:          "invalid email",
			input:         SignUpInput{Email: "bob", Password: testPassword},
			wantErr:       true,
			wantErrTarget: &primitive.ErrInvalidEmail{},
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
			wantErrTarget: &primitive.ErrInvalidEmail{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateSignUpInput(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateSignUpInput: %v", err)
				}
				return
			}

			if !errors.As(err, testCase.wantErrTarget) {
				t.Fatalf("error = %v, want %T", err, testCase.wantErrTarget)
			}
		})
	}
}
