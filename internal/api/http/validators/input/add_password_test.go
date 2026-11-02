package input

import (
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

func TestValidateAddPasswordInput(t *testing.T) {
	cases := []struct {
		name          string
		input         AddPasswordInput
		want          AddPasswordInput
		wantErr       bool
		wantErrTarget any
	}{
		{
			name:  "valid password",
			input: AddPasswordInput{Password: replacementPassword},
			want:  AddPasswordInput{Password: replacementPassword},
		},
		{
			name:          "missing password",
			input:         AddPasswordInput{},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordTooShort{},
		},
		{
			name:          "password with surrounding whitespace is not trimmed",
			input:         AddPasswordInput{Password: " " + replacementPassword},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPasswordContainsWhitespace{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ValidateAddPasswordInput(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateAddPasswordInput: %v", err)
				}
				if got != testCase.want {
					t.Fatalf("ValidateAddPasswordInput = %+v, want %+v", got, testCase.want)
				}
				return
			}

			if !errors.As(err, testCase.wantErrTarget) {
				t.Fatalf("error = %v, want %T", err, testCase.wantErrTarget)
			}
		})
	}
}
