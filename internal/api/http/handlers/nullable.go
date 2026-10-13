package handlers

import "time"

// NullableString answers nil for an empty domain string, so the response field encodes as null.
func NullableString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

// NullableTime answers nil for a zero domain time, so the response field encodes as null.
func NullableTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	return &value
}
