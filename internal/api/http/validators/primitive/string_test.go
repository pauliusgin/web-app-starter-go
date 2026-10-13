package primitive

import (
	"errors"
	"testing"
)

func TestRequiredString(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "single character", input: "a"},
		{name: "surrounded by whitespace", input: "  a  "},
		{name: "empty", input: "", wantErr: true},
		{name: "whitespace only", input: " \t\n ", wantErr: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := RequiredString(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("RequiredString: %v", err)
				}
				return
			}

			if !errors.As(err, &ErrStringMissing{}) {
				t.Fatalf("error = %v, want ErrStringMissing", err)
			}
		})
	}
}
