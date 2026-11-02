package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
)

type resendEmailVerificationForTest struct {
	users      *repositories.InMemoryUserRepository
	challenges challengeLinks
	issuance   inMemoryIssuanceGuard
	userID     uuid.UUID
}

func newResendEmailVerificationForTest(t *testing.T, seeded seedUserParams) resendEmailVerificationForTest {
	t.Helper()

	users := repositories.NewInMemoryUserRepository()

	return resendEmailVerificationForTest{
		users:      users,
		challenges: newChallengeLinks(),
		issuance:   inMemoryIssuanceGuard{rateLimitedTargets: map[string]bool{}},
		userID:     seedUser(t, users, seeded),
	}
}

func (h resendEmailVerificationForTest) resend() error {
	return ResendEmailVerification(context.Background(), ResendEmailVerificationDependencies{
		UserWriteRepository:            h.users,
		EnsureChallengeIssuanceAllowed: h.issuance.ensureAllowed,
		GenerateAuthChallenge:          h.challenges.generate,
	}, ResendEmailVerificationParams{UserID: h.userID})
}

func TestResendEmailVerificationSendsALinkToTheCurrentEmail(t *testing.T) {
	harness := newResendEmailVerificationForTest(t, seedUserParams{email: testEmail})

	if err := harness.resend(); err != nil {
		t.Fatalf("ResendEmailVerification: %v", err)
	}

	wantLink := challenge.GenerateParams{
		UserID:     harness.userID,
		Target:     testEmail,
		TargetType: challenge.TargetTypeEmail,
		Purpose:    challenge.PurposeEmailVerification,
	}
	harness.challenges.assertOnlyLinkSent(t, wantLink)
}

func TestResendEmailVerificationRejectsAVerifiedEmailBeforeTheRateLimit(t *testing.T) {
	harness := newResendEmailVerificationForTest(t, seedUserParams{email: testEmail, verified: true})
	harness.issuance.rateLimitedTargets[testEmail] = true

	err := harness.resend()

	alreadyVerified, isAlreadyVerified := errors.AsType[e.ErrEmailAlreadyVerified](err)
	if !isAlreadyVerified || alreadyVerified.Email != testEmail {
		t.Fatalf("error = %v, want ErrEmailAlreadyVerified for %q", err, testEmail)
	}
	harness.challenges.assertNoneSent(t)
}

func TestResendEmailVerificationSendsNothingWhenRejected(t *testing.T) {
	cases := []struct {
		name    string
		arrange func(harness *resendEmailVerificationForTest)
		wantErr func(err error) bool
	}{
		{
			name:    "rate limited address",
			arrange: func(harness *resendEmailVerificationForTest) { harness.issuance.rateLimitedTargets[testEmail] = true },
			wantErr: isError[challenge.ErrIssuanceRateLimited],
		},
		{
			name:    "unknown user",
			arrange: func(harness *resendEmailVerificationForTest) { harness.userID = uuid.New() },
			wantErr: isError[e.ErrUserNotFound],
		},
		{
			name: "link cannot be created",
			arrange: func(harness *resendEmailVerificationForTest) {
				harness.challenges.repository.SaveFailure = challenge.ErrSaveChallengeFailed{Cause: errors.New("connection refused")}
			},
			wantErr: isError[challenge.ErrSaveChallengeFailed],
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newResendEmailVerificationForTest(t, seedUserParams{email: testEmail})
			testCase.arrange(&harness)

			err := harness.resend()

			if !testCase.wantErr(err) {
				t.Fatalf("error = %#v, want the case's rejection", err)
			}
			harness.challenges.assertNoneSent(t)
		})
	}
}
