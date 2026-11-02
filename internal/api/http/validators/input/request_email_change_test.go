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
		want          RequestEmailChangeInput
		wantErr       bool
		wantErrTarget any
	}{
		{
			name:  "valid email",
			input: RequestEmailChangeInput{NewEmail: testEmail},
			want:  RequestEmailChangeInput{NewEmail: testEmail},
		},
		{
			name:  "email is trimmed and lowercased",
			input: RequestEmailChangeInput{NewEmail: " Accountant@Example.COM "},
			want:  RequestEmailChangeInput{NewEmail: testEmail},
		},
		{
			name:          "invalid email",
			input:         RequestEmailChangeInput{NewEmail: "bob"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrEmailInvalid{},
		},
		{
			name:          "missing email",
			input:         RequestEmailChangeInput{},
			wantErr:       true,
			wantErrTarget: &primitive.ErrEmailInvalid{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ValidateRequestEmailChangeInput(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateRequestEmailChangeInput: %v", err)
				}
				if got != testCase.want {
					t.Fatalf("ValidateRequestEmailChangeInput = %+v, want %+v", got, testCase.want)
				}
				return
			}

			if !errors.As(err, testCase.wantErrTarget) {
				t.Fatalf("error = %v, want %T", err, testCase.wantErrTarget)
			}
		})
	}
}
