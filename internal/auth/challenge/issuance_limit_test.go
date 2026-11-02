package challenge

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

var issuanceCheckedAt = time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)

const limitedTarget = "unverified@example.com"

type issuanceLimitForTest struct {
	store  *InMemoryChallengeRepository
	userID uuid.UUID
}

func newIssuanceLimitForTest() issuanceLimitForTest {
	return issuanceLimitForTest{store: NewInMemoryChallengeRepository(), userID: uuid.New()}
}

type seedSentChallengeParams struct {
	UserID  uuid.UUID
	Target  string
	Purpose Purpose
	SentAgo time.Duration
}

func (h issuanceLimitForTest) seedSentChallenge(params seedSentChallengeParams) {
	h.store.Challenges = append(h.store.Challenges, Challenge{
		ID:        uuid.New(),
		UserID:    params.UserID,
		Target:    params.Target,
		Purpose:   params.Purpose,
		CreatedAt: issuanceCheckedAt.Add(-params.SentAgo),
	})
}

func (h issuanceLimitForTest) seedEmailVerificationsToTarget(sentAgo ...time.Duration) {
	for _, ago := range sentAgo {
		h.seedSentChallenge(seedSentChallengeParams{
			UserID:  h.userID,
			Target:  limitedTarget,
			Purpose: PurposeEmailVerification,
			SentAgo: ago,
		})
	}
}

func (h issuanceLimitForTest) ensureAllowed() error {
	return EnsureIssuanceAllowed(context.Background(),
		EnsureIssuanceAllowedDependencies{
			ReadRepository: h.store,
			Now:            func() time.Time { return issuanceCheckedAt },
		},
		EnsureIssuanceAllowedParams{
			UserID:  h.userID,
			Target:  limitedTarget,
			Purpose: PurposeEmailVerification,
		})
}

func assertRateLimitedFor(t *testing.T, err error, wantRetryAfter time.Duration) {
	t.Helper()

	rateLimited, isRateLimited := errors.AsType[ErrIssuanceRateLimited](err)
	if !isRateLimited {
		t.Fatalf("err = %v, want ErrIssuanceRateLimited", err)
	}
	if rateLimited.RetryAfter != wantRetryAfter {
		t.Errorf("RetryAfter = %s, want %s", rateLimited.RetryAfter, wantRetryAfter)
	}
}

func TestEnsureIssuanceAllowedAllowsAFirstChallenge(t *testing.T) {
	harness := newIssuanceLimitForTest()

	if err := harness.ensureAllowed(); err != nil {
		t.Fatalf("EnsureIssuanceAllowed: %v, want nil", err)
	}
}

func TestEnsureIssuanceAllowedEnforcesTheMinimumGapPerTarget(t *testing.T) {
	cases := []struct {
		name           string
		latestSentAgo  time.Duration
		wantRetryAfter time.Duration
	}{
		{name: "just sent", latestSentAgo: 0, wantRetryAfter: IssuanceMinimumGap},
		{name: "one second short of the gap", latestSentAgo: 29 * time.Second, wantRetryAfter: time.Second},
		{name: "exactly the gap", latestSentAgo: IssuanceMinimumGap},
		{name: "past the gap", latestSentAgo: time.Minute},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newIssuanceLimitForTest()
			harness.seedEmailVerificationsToTarget(time.Hour, testCase.latestSentAgo)

			err := harness.ensureAllowed()

			if testCase.wantRetryAfter == 0 {
				if err != nil {
					t.Fatalf("EnsureIssuanceAllowed: %v, want nil", err)
				}
				return
			}
			assertRateLimitedFor(t, err, testCase.wantRetryAfter)
		})
	}
}

func TestEnsureIssuanceAllowedCapsTheTargetUntilItsOldestWindowedChallengeAgesOut(t *testing.T) {
	harness := newIssuanceLimitForTest()
	harness.seedEmailVerificationsToTarget(10*time.Hour, 8*time.Hour, 6*time.Hour, 4*time.Hour, 2*time.Hour)

	assertRateLimitedFor(t, harness.ensureAllowed(), 14*time.Hour)
}

func TestEnsureIssuanceAllowedIgnoresChallengesOutsideTheWindow(t *testing.T) {
	harness := newIssuanceLimitForTest()
	harness.seedEmailVerificationsToTarget(IssuanceWindow, 30*time.Hour, 8*time.Hour, 6*time.Hour, 4*time.Hour, 2*time.Hour)

	if err := harness.ensureAllowed(); err != nil {
		t.Fatalf("EnsureIssuanceAllowed: %v, want nil with only 4 challenges in the window", err)
	}
}

func TestEnsureIssuanceAllowedCapsTheUserAcrossTargets(t *testing.T) {
	harness := newIssuanceLimitForTest()
	for index := range IssuanceLimitPerUser {
		harness.seedSentChallenge(seedSentChallengeParams{
			UserID:  harness.userID,
			Target:  fmt.Sprintf("address-%d@example.com", index),
			Purpose: PurposeEmailVerification,
			SentAgo: time.Duration(20-index) * time.Hour,
		})
	}

	assertRateLimitedFor(t, harness.ensureAllowed(), 4*time.Hour)
}

func TestEnsureIssuanceAllowedIgnoresOtherUsersChallengesToTheSameTarget(t *testing.T) {
	harness := newIssuanceLimitForTest()
	for range IssuanceLimitPerTarget {
		harness.seedSentChallenge(seedSentChallengeParams{
			UserID:  uuid.New(),
			Target:  limitedTarget,
			Purpose: PurposeEmailVerification,
			SentAgo: 0,
		})
	}

	if err := harness.ensureAllowed(); err != nil {
		t.Fatalf("EnsureIssuanceAllowed: %v, want nil since the limits are per user", err)
	}
}

func TestEnsureIssuanceAllowedIgnoresChallengesForAnotherPurpose(t *testing.T) {
	harness := newIssuanceLimitForTest()
	for range IssuanceLimitPerUser {
		harness.seedSentChallenge(seedSentChallengeParams{
			UserID:  harness.userID,
			Target:  limitedTarget,
			Purpose: PurposePasswordReset,
			SentAgo: 0,
		})
	}

	if err := harness.ensureAllowed(); err != nil {
		t.Fatalf("EnsureIssuanceAllowed: %v, want nil", err)
	}
}

func TestEnsureIssuanceAllowedPropagatesAHistoryLookupFailure(t *testing.T) {
	harness := newIssuanceLimitForTest()
	lookupFailure := ErrIssuanceHistoryLookupFailed{Cause: errors.New("connection reset")}
	harness.store.ListFailure = lookupFailure

	err := harness.ensureAllowed()

	if _, isLookupFailure := errors.AsType[ErrIssuanceHistoryLookupFailed](err); !isLookupFailure {
		t.Fatalf("err = %v, want ErrIssuanceHistoryLookupFailed", err)
	}
}
