package primitive

import (
	"errors"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	cases := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{name: "plain address", email: "accountant@example.com"},
		{name: "mixed case is left to the caller", email: "Bob@Example.COM"},
		{name: "no domain", email: "bob", wantErr: true},
		{name: "empty", email: "", wantErr: true},
		{name: "display name", email: "Bob <bob@example.com>", wantErr: true},
		{name: "surrounding whitespace", email: " bob@example.com ", wantErr: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateEmail(testCase.email)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateEmail(%q): %v", testCase.email, err)
				}
				return
			}

			invalidEmail := ErrInvalidEmail{}
			if !errors.As(err, &invalidEmail) {
				t.Fatalf("error = %v, want ErrInvalidEmail", err)
			}
			if invalidEmail.Email != testCase.email {
				t.Errorf("Email = %q, want %q", invalidEmail.Email, testCase.email)
			}
		})
	}
}
