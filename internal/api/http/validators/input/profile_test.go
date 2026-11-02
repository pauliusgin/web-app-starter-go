package input

import (
	"errors"
	"strings"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

func TestValidateProfileInput(t *testing.T) {
	cases := []struct {
		name          string
		input         ProfileInput
		want          ProfileInput
		wantErr       bool
		wantErrTarget any
	}{
		{
			name:  "both names",
			input: ProfileInput{FirstName: "Ada", LastName: "Lovelace"},
			want:  ProfileInput{FirstName: "Ada", LastName: "Lovelace"},
		},
		{
			name:  "names are trimmed",
			input: ProfileInput{FirstName: " Ada\t", LastName: "  Lovelace "},
			want:  ProfileInput{FirstName: "Ada", LastName: "Lovelace"},
		},
		{
			name: "picture upload id is trimmed",
			input: ProfileInput{FirstName: "Ada", LastName: "Lovelace",
				PictureUploadID: " 7d2e5f3c-6a91-4b22-9c4d-3f8b0e7a5d26\t"},
			want: ProfileInput{FirstName: "Ada", LastName: "Lovelace",
				PictureUploadID: "7d2e5f3c-6a91-4b22-9c4d-3f8b0e7a5d26"},
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
			name:  "space and dash separated names",
			input: ProfileInput{FirstName: "Anna Marija", LastName: "Taylor-Joy"},
			want:  ProfileInput{FirstName: "Anna Marija", LastName: "Taylor-Joy"},
		},
		{
			name:  "decomposed letters are composed",
			input: ProfileInput{FirstName: "Že\u0307ra", LastName: "Ivanova"},
			want:  ProfileInput{FirstName: "Žėra", LastName: "Ivanova"},
		},
		{
			name:  "name at the maximum length",
			input: ProfileInput{FirstName: strings.Repeat("ą", primitive.MaxPersonNameLength), LastName: "Lovelace"},
			want:  ProfileInput{FirstName: strings.Repeat("ą", primitive.MaxPersonNameLength), LastName: "Lovelace"},
		},
		{
			name:          "first name with a leading dash",
			input:         ProfileInput{FirstName: "-Taylor", LastName: "Joy"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrFirstNameInvalid{},
		},
		{
			name:          "first name too long",
			input:         ProfileInput{FirstName: strings.Repeat("a", primitive.MaxPersonNameLength+1), LastName: "Lovelace"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPersonNameTooLong{},
		},
		{
			name:          "last name with a double space",
			input:         ProfileInput{FirstName: "Ada", LastName: "Love  lace"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrLastNameInvalid{},
		},
		{
			name:          "invalid last name reports the person-name cause",
			input:         ProfileInput{FirstName: "Ada", LastName: "Lovelace3"},
			wantErr:       true,
			wantErrTarget: &primitive.ErrPersonNameMalformed{},
		},
		{
			name:          "missing wins over invalid",
			input:         ProfileInput{FirstName: "Ada3", LastName: " "},
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
			got, err := ValidateProfileInput(testCase.input)

			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateProfileInput: %v", err)
				}
				if got != testCase.want {
					t.Fatalf("ValidateProfileInput = %+v, want %+v", got, testCase.want)
				}
				return
			}

			if !errors.As(err, testCase.wantErrTarget) {
				t.Fatalf("error = %v, want %T", err, testCase.wantErrTarget)
			}
		})
	}
}
