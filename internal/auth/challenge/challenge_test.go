package challenge

import (
	"testing"
	"time"
)

func TestIsUsedReadsTheZeroTimeAsUnused(t *testing.T) {
	cases := []struct {
		name     string
		usedAt   time.Time
		wantUsed bool
	}{
		{name: "zero time is unused", usedAt: time.Time{}},
		{name: "stamped time is used", usedAt: fixedUsedAt, wantUsed: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			storedChallenge := Challenge{UsedAt: testCase.usedAt}

			if storedChallenge.IsUsed() != testCase.wantUsed {
				t.Errorf("IsUsed() = %t, want %t", storedChallenge.IsUsed(), testCase.wantUsed)
			}
		})
	}
}

func TestIsExpiredComparesAgainstTheInstantItIsGiven(t *testing.T) {
	expiresAt := fixedCreatedAt.Add(Lifetime)

	cases := []struct {
		name        string
		at          time.Time
		wantExpired bool
	}{
		{name: "before the expiry", at: expiresAt.Add(-time.Second)},
		{name: "at the expiry", at: expiresAt, wantExpired: true},
		{name: "after the expiry", at: expiresAt.Add(time.Second), wantExpired: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			storedChallenge := Challenge{ExpiresAt: expiresAt}

			if storedChallenge.IsExpired(testCase.at) != testCase.wantExpired {
				t.Errorf("IsExpired(%s) = %t, want %t", testCase.at,
					storedChallenge.IsExpired(testCase.at), testCase.wantExpired)
			}
		})
	}
}
