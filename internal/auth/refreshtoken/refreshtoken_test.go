package refreshtoken

import (
	"testing"
	"time"
)

func TestIsRevokedReadsTheZeroTimeAsLive(t *testing.T) {
	cases := []struct {
		name        string
		revokedAt   time.Time
		wantRevoked bool
	}{
		{name: "zero time is live", revokedAt: time.Time{}},
		{name: "stamped time is revoked", revokedAt: fixedRevokedAt, wantRevoked: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			storedToken := RefreshToken{RevokedAt: testCase.revokedAt}

			if storedToken.IsRevoked() != testCase.wantRevoked {
				t.Errorf("IsRevoked() = %t, want %t", storedToken.IsRevoked(), testCase.wantRevoked)
			}
		})
	}
}

func TestIsExpiredComparesAgainstTheInstantItIsGiven(t *testing.T) {
	expiresAt := fixedRotatedAt

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
			storedToken := RefreshToken{ExpiresAt: expiresAt}

			if storedToken.IsExpired(testCase.at) != testCase.wantExpired {
				t.Errorf("IsExpired(%s) = %t, want %t",
					testCase.at, storedToken.IsExpired(testCase.at), testCase.wantExpired)
			}
		})
	}
}
