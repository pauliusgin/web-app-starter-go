package handlers

import (
	"testing"
	"time"
)

func TestNullableStringAnswersNilForTheEmptyString(t *testing.T) {
	if got := NullableString(""); got != nil {
		t.Errorf("NullableString(\"\") = %q, want nil", *got)
	}
}

func TestNullableStringPointsAtAPresentValue(t *testing.T) {
	got := NullableString("Ada")
	if got == nil || *got != "Ada" {
		t.Errorf("NullableString(\"Ada\") = %v, want a pointer to \"Ada\"", got)
	}
}

func TestNullableTimeAnswersNilForTheZeroTime(t *testing.T) {
	if got := NullableTime(time.Time{}); got != nil {
		t.Errorf("NullableTime(zero) = %v, want nil", *got)
	}
}

func TestNullableTimePointsAtAPresentValue(t *testing.T) {
	stamp := time.Date(2026, 3, 15, 8, 30, 0, 0, time.UTC)

	got := NullableTime(stamp)
	if got == nil || !got.Equal(stamp) {
		t.Errorf("NullableTime(%v) = %v, want a pointer to it", stamp, got)
	}
}
