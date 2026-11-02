package handlers

import (
	"testing"
	"time"
)

func TestNullableString(t *testing.T) {
	ada := "Ada"
	cases := []struct {
		name  string
		value string
		want  *string
	}{
		{name: "empty answers nil", value: ""},
		{name: "text is pointed to", value: ada, want: &ada},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := NullableString(testCase.value)

			if (got == nil) != (testCase.want == nil) || (got != nil && *got != *testCase.want) {
				t.Errorf("NullableString(%q) = %v, want %v", testCase.value, got, testCase.want)
			}
		})
	}
}

func TestNullableTime(t *testing.T) {
	verifiedAt := time.Date(2026, time.March, 4, 10, 30, 0, 0, time.UTC)
	cases := []struct {
		name  string
		value time.Time
		want  *time.Time
	}{
		{name: "zero time answers nil", value: time.Time{}},
		{name: "time is pointed to", value: verifiedAt, want: &verifiedAt},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := NullableTime(testCase.value)

			if (got == nil) != (testCase.want == nil) || (got != nil && !got.Equal(*testCase.want)) {
				t.Errorf("NullableTime(%v) = %v, want %v", testCase.value, got, testCase.want)
			}
		})
	}
}
