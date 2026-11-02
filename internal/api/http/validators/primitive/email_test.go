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

			invalidEmail := ErrEmailInvalid{}
			if !errors.As(err, &invalidEmail) {
				t.Fatalf("error = %v, want ErrEmailInvalid", err)
			}
			if invalidEmail.Email != testCase.email {
				t.Errorf("Email = %q, want %q", invalidEmail.Email, testCase.email)
			}
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	cases := []struct {
		name  string
		email string
		want  string
	}{
		{name: "already normalized", email: "accountant@example.com", want: "accountant@example.com"},
		{name: "mixed case", email: "Bob@Example.COM", want: "bob@example.com"},
		{name: "surrounding whitespace", email: " \tbob@example.com\n", want: "bob@example.com"},
		{name: "empty", email: "", want: ""},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := NormalizeEmail(testCase.email); got != testCase.want {
				t.Fatalf("NormalizeEmail(%q) = %q, want %q", testCase.email, got, testCase.want)
			}
		})
	}
}
