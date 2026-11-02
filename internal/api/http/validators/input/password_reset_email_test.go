package input

import (
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

func TestValidatePasswordResetEmailInput(t *testing.T) {
	cases := []struct {
		name          string
		input         PasswordResetEmailInput
		want          PasswordResetEmailInput
		wantErr       bool
		wantErrTarget any
	}{
		{
			name:  "valid email",
			input: PasswordResetEmailInput{Email: testEmail},
			want:  PasswordResetEmailInput{Email: testEmail},
		},
		{
			name:  "email is trimmed and lowercased",
			input: PasswordResetEmailInput{Email: " Accountant@Example.COM "},
			want:  PasswordResetEmailInput{Email: testEmail},
		},
		{
			name:          "invalid email",
			input:         PasswordResetEmailInput{Email: "bob"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrEmailInvalid{},
		},
		{
			name:          "missing email",
			input:         PasswordResetEmailInput{},
			wantErr:       true,
			wantErrTarget: &primitive.ErrEmailInvalid{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ValidatePasswordResetEmailInput(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidatePasswordResetEmailInput: %v", err)
				}
				if got != testCase.want {
					t.Fatalf("ValidatePasswordResetEmailInput = %+v, want %+v", got, testCase.want)
				}
				return
			}

			if !errors.As(err, testCase.wantErrTarget) {
				t.Fatalf("error = %v, want %T", err, testCase.wantErrTarget)
			}
		})
	}
}
