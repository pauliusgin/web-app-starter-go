package input

import (
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

const replacementPassword = "new-ledger-password"

func TestValidateChangePasswordInput(t *testing.T) {
	cases := []struct {
		name          string
		input         ChangePasswordInput
		wantErr       bool
		wantErrTarget any
	}{
		{
			name:  "current password and a valid new password",
			input: ChangePasswordInput{CurrentPassword: testPassword, NewPassword: replacementPassword},
		},
		{
			name:  "current password predating the rules is still accepted",
			input: ChangePasswordInput{CurrentPassword: "short", NewPassword: replacementPassword},
		},
		{
			name:          "missing current password",
			input:         ChangePasswordInput{NewPassword: replacementPassword},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordMissing{},
		},
		{
			name:          "missing new password",
			input:         ChangePasswordInput{CurrentPassword: testPassword},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordTooShort{},
		},
		{
			name:          "new password with whitespace",
			input:         ChangePasswordInput{CurrentPassword: testPassword, NewPassword: "new ledger password"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordContainsWhitespace{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateChangePasswordInput(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateChangePasswordInput: %v", err)
				}
				return
			}

			if !errors.As(err, testCase.wantErrTarget) {
				t.Fatalf("error = %v, want %T", err, testCase.wantErrTarget)
			}
		})
	}
}
