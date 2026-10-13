package input

import (
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

func TestValidateRequestEmailChangeInput(t *testing.T) {
	cases := []struct {
		name          string
		input         RequestEmailChangeInput
		wantErr       bool
		wantErrTarget any
	}{
		{
			name:  "valid email",
			input: RequestEmailChangeInput{NewEmail: testEmail},
		},
		{
			name:          "invalid email",
			input:         RequestEmailChangeInput{NewEmail: "bob"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrInvalidEmail{},
		},
		{
			name:          "missing email",
			input:         RequestEmailChangeInput{},
			wantErr:       true,
			wantErrTarget: &primitive.ErrInvalidEmail{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateRequestEmailChangeInput(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateRequestEmailChangeInput: %v", err)
				}
				return
			}

			if !errors.As(err, testCase.wantErrTarget) {
				t.Fatalf("error = %v, want %T", err, testCase.wantErrTarget)
			}
		})
	}
}
