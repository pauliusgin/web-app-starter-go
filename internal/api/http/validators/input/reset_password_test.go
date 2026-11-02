package input

import (
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

const testResetToken = "Rt4pW8kL1zX6cV9bN2mQ5sD7fG0hJ3aE"

func TestValidateResetPasswordInput(t *testing.T) {
	cases := []struct {
		name          string
		input         ResetPasswordInput
		want          ResetPasswordInput
		wantErr       bool
		wantErrTarget any
	}{
		{
			name:  "valid token and password",
			input: ResetPasswordInput{Token: testResetToken, NewPassword: replacementPassword},
			want:  ResetPasswordInput{Token: testResetToken, NewPassword: replacementPassword},
		},
		{
			name:  "token is trimmed",
			input: ResetPasswordInput{Token: " " + testResetToken + "\n", NewPassword: replacementPassword},
			want:  ResetPasswordInput{Token: testResetToken, NewPassword: replacementPassword},
		},
		{
			name:          "missing token",
			input:         ResetPasswordInput{Token: "   ", NewPassword: replacementPassword},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordResetTokenMissing{},
		},
		{
			name:          "missing token is reported before the password",
			input:         ResetPasswordInput{},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordResetTokenMissing{},
		},
		{
			name:          "short password",
			input:         ResetPasswordInput{Token: testResetToken, NewPassword: "short"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordTooShort{},
		},
		{
			name:          "password with surrounding whitespace is not trimmed",
			input:         ResetPasswordInput{Token: testResetToken, NewPassword: " " + replacementPassword},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordContainsWhitespace{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ValidateResetPasswordInput(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateResetPasswordInput: %v", err)
				}
				if got != testCase.want {
					t.Fatalf("ValidateResetPasswordInput = %+v, want %+v", got, testCase.want)
				}
				return
			}

			if !errors.As(err, testCase.wantErrTarget) {
				t.Fatalf("error = %v, want %T", err, testCase.wantErrTarget)
			}
		})
	}
}
