package input

import (
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

func TestValidateSignInInput(t *testing.T) {
	cases := []struct {
		name          string
		input         SignInInput
		wantErr       bool
		wantErrTarget any
	}{
		{
			name:  "valid email and password",
			input: SignInInput{Email: testEmail, Password: testPassword},
		},
		{
			name:          "invalid email",
			input:         SignInInput{Email: "bob", Password: testPassword},
			wantErr:       true,
			wantErrTarget: &primitive.ErrInvalidEmail{},
		},
		{
			name:          "missing password",
			input:         SignInInput{Email: testEmail},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordMissing{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateSignInInput(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateSignInInput: %v", err)
				}
				return
			}

			if !errors.As(err, testCase.wantErrTarget) {
				t.Fatalf("error = %v, want %T", err, testCase.wantErrTarget)
			}
		})
	}
}
