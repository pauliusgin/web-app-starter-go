package input

import (
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

func TestValidateProfileInput(t *testing.T) {
	cases := []struct {
		name          string
		input         ProfileInput
		wantErr       bool
		wantErrTarget any
	}{
		{
			name:  "both names",
			input: ProfileInput{FirstName: "Ada", LastName: "Lovelace"},
		},
		{
			name:          "missing first name",
			input:         ProfileInput{LastName: "Lovelace"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrFirstNameMissing{},
		},
		{
			name:          "blank first name",
			input:         ProfileInput{FirstName: " \t", LastName: "Lovelace"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrFirstNameMissing{},
		},
		{
			name:          "missing last name",
			input:         ProfileInput{FirstName: "Ada"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrLastNameMissing{},
		},
		{
			name:          "blank last name",
			input:         ProfileInput{FirstName: "Ada", LastName: "  "},
			wantErr:       true,
			wantErrTarget: &primitive.ErrLastNameMissing{},
		},
		{
			name:          "both missing reports the first name",
			input:         ProfileInput{},
			wantErr:       true,
			wantErrTarget: &primitive.ErrFirstNameMissing{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateProfileInput(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateProfileInput: %v", err)
				}
				return
			}

			if !errors.As(err, testCase.wantErrTarget) {
				t.Fatalf("error = %v, want %T", err, testCase.wantErrTarget)
			}
		})
	}
}
