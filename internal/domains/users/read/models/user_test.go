package models

import (
	"slices"
	"testing"
)

func TestNewSignInMethodsListsStoredCredentialsInFixedOrder(t *testing.T) {
	cases := []struct {
		name   string
		params SignInMethodsParams
		want   []SignInMethod
	}{
		{
			name:   "password only",
			params: SignInMethodsParams{HasPassword: true},
			want:   []SignInMethod{SignInMethodEmailPassword},
		},
		{
			name:   "google only",
			params: SignInMethodsParams{HasGoogleSubject: true},
			want:   []SignInMethod{SignInMethodGoogle},
		},
		{
			name:   "both",
			params: SignInMethodsParams{HasPassword: true, HasGoogleSubject: true},
			want:   []SignInMethod{SignInMethodEmailPassword, SignInMethodGoogle},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := NewSignInMethods(testCase.params)

			if !slices.Equal(got, testCase.want) {
				t.Errorf("NewSignInMethods = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestNewSignInMethodsWithoutCredentialsIsEmptyNotNil(t *testing.T) {
	got := NewSignInMethods(SignInMethodsParams{})

	if got == nil || len(got) != 0 {
		t.Errorf("NewSignInMethods = %#v, want empty non-nil slice", got)
	}
}
