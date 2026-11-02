package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
)

type requestPasswordResetForTest struct {
	users      *repositories.InMemoryUserRepository
	challenges challengeLinks
	issuance   inMemoryIssuanceGuard
}

func newRequestPasswordResetForTest() requestPasswordResetForTest {
	return requestPasswordResetForTest{
		users:      repositories.NewInMemoryUserRepository(),
		challenges: newChallengeLinks(),
		issuance:   inMemoryIssuanceGuard{rateLimitedTargets: map[string]bool{}},
	}
}

func (h requestPasswordResetForTest) request(email string) error {
	return RequestPasswordReset(context.Background(), RequestPasswordResetDependencies{
		UserWriteRepository:            h.users,
		EnsureChallengeIssuanceAllowed: h.issuance.ensureAllowed,
		GenerateAuthChallenge:          h.challenges.generate,
	}, RequestPasswordResetParams{Email: email})
}

func TestRequestPasswordResetSendsAResetLinkToThePasswordAccount(t *testing.T) {
	harness := newRequestPasswordResetForTest()
	userID := seedUser(t, harness.users, seedUserParams{email: testEmail})

	if err := harness.request(testEmail); err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}

	harness.challenges.assertOnlyLinkSent(t, challenge.GenerateParams{
		UserID:     userID,
		Target:     testEmail,
		TargetType: challenge.TargetTypeEmail,
		Purpose:    challenge.PurposePasswordReset,
	})
}

// Each case must answer exactly like a sent link, or the endpoint would tell callers which emails have accounts.
func TestRequestPasswordResetSilentlySendsNothingWhenNoLinkMayGoOut(t *testing.T) {
	cases := []struct {
		name    string
		arrange func(t *testing.T, harness requestPasswordResetForTest)
	}{
		{
			name:    "unknown email",
			arrange: func(t *testing.T, harness requestPasswordResetForTest) {},
		},
		{
			name: "google-only account",
			arrange: func(t *testing.T, harness requestPasswordResetForTest) {
				seedGoogleUser(t, harness.users, testGoogleSubject)
			},
		},
		{
			name: "rate limited address",
			arrange: func(t *testing.T, harness requestPasswordResetForTest) {
				seedUser(t, harness.users, seedUserParams{email: testEmail})
				harness.issuance.rateLimitedTargets[testEmail] = true
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newRequestPasswordResetForTest()
			testCase.arrange(t, harness)

			if err := harness.request(testEmail); err != nil {
				t.Fatalf("RequestPasswordReset = %v, want nil", err)
			}
			harness.challenges.assertNoneSent(t)
		})
	}
}

func TestRequestPasswordResetReportsAFailedLookup(t *testing.T) {
	harness := newRequestPasswordResetForTest()
	seedUser(t, harness.users, seedUserParams{email: testEmail})
	lookupFailure := errors.New("connection reset by peer")
	harness.users.LookupFailure = lookupFailure

	err := harness.request(testEmail)

	if !errors.Is(err, lookupFailure) {
		t.Fatalf("error = %v, want the lookup failure", err)
	}
	harness.challenges.assertNoneSent(t)
}
