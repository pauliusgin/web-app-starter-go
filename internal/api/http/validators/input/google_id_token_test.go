package input

import (
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

func TestValidateGoogleIDTokenInput(t *testing.T) {
	cases := []struct {
		name    string
		input   GoogleIDTokenInput
		want    GoogleIDTokenInput
		wantErr bool
	}{
		{name: "token kept", input: GoogleIDTokenInput{IDToken: "eyJhbGciOiJSUzI1NiJ9.e30.c2ln"}, want: GoogleIDTokenInput{IDToken: "eyJhbGciOiJSUzI1NiJ9.e30.c2ln"}},
		{name: "token trimmed", input: GoogleIDTokenInput{IDToken: " eyJ.e30.c2ln\n"}, want: GoogleIDTokenInput{IDToken: "eyJ.e30.c2ln"}},
		{name: "empty token", input: GoogleIDTokenInput{}, wantErr: true},
		{name: "blank token", input: GoogleIDTokenInput{IDToken: "   "}, wantErr: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ValidateGoogleIDTokenInput(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateGoogleIDTokenInput: %v", err)
				}
				if got != testCase.want {
					t.Errorf("got %+v, want %+v", got, testCase.want)
				}
				return
			}

			if !errors.As(err, &primitive.ErrGoogleIDTokenMissing{}) {
				t.Errorf("err = %v, want ErrGoogleIDTokenMissing", err)
			}
			if got != (GoogleIDTokenInput{}) {
				t.Errorf("got %+v, want zero input", got)
			}
		})
	}
}
