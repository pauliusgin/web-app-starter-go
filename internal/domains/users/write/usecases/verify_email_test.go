package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

const presentedVerificationToken = "Kq7wL2mZ9xR4tV6yB1nH8pD3sG5jF0aC"

const unmatchedVerificationToken = "Zx1cV4bN7mQ0wE3rT6yU9iO2pA5sD8fG"

// liveLinkExpiresAt and expiredLinkExpiresAt sit far either side of the wall clock the repository checks against.
var liveLinkExpiresAt = time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

var expiredLinkExpiresAt = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)

type verifyEmailFixture struct {
	users      *repositories.InMemoryUserRepository
	challenges *challenge.InMemoryChallengeRepository
	events     *eventstest.CapturedEvents
	userID     uuid.UUID
}

// newVerifyEmailFixture seeds an unverified user on testEmail and one live link proving challengeTarget.
func newVerifyEmailFixture(t *testing.T, challengeTarget string) *verifyEmailFixture {
	t.Helper()

	users := repositories.NewInMemoryUserRepository()
	userID := seedUser(t, users, seedUserParams{email: testEmail})

	return &verifyEmailFixture{
		users: users,
		challenges: challenge.NewInMemoryChallengeRepository(challenge.Challenge{
			ID:         uuid.New(),
			UserID:     userID,
			TargetType: challenge.TargetTypeEmail,
			Target:     challengeTarget,
			Purpose:    challenge.PurposeEmailVerification,
			TokenHash:  token.Hash(presentedVerificationToken),
			CreatedAt:  fixedSeededAt,
			ExpiresAt:  liveLinkExpiresAt,
		}),
		events: &eventstest.CapturedEvents{},
		userID: userID,
	}
}

func (f *verifyEmailFixture) verify(params VerifyEmailParams) error {
	return VerifyEmail(context.Background(), VerifyEmailDependencies{
		UserWriteRepository:          f.users,
		EventDispatcher:              f.events.Dispatcher(),
		FindAuthChallengeByTokenHash: f.challenges.FindByTokenHash,
		ConsumeAuthChallenge:         f.challenges.MarkUsed,
	}, params)
}

func (f *verifyEmailFixture) clickLink() error {
	return f.verify(VerifyEmailParams{VerificationToken: presentedVerificationToken, UserID: f.userID})
}

func (f *verifyEmailFixture) stored(t *testing.T) repositories.InMemoryUser {
	t.Helper()

	return storedUser(t, f.users, f.userID)
}

func (f *verifyEmailFixture) assertChallengeSpent(t *testing.T, wantSpent bool) {
	t.Helper()

	if spent := f.challenges.Challenges[0].IsUsed(); spent != wantSpent {
		t.Errorf("challenge IsUsed() = %t, want %t", spent, wantSpent)
	}
}

func (f *verifyEmailFixture) assertNothingWritten(t *testing.T) {
	t.Helper()

	stored := f.stored(t)
	if stored.User.IsEmailVerified() || stored.User.Email != testEmail {
		t.Errorf("stored user = %+v, want it left unverified on %q", stored.User, testEmail)
	}
	f.assertChallengeSpent(t, false)
	f.events.AssertNoneDispatched(t)
}

func TestVerifyEmailStampsTheCurrentAddressSpendsTheLinkAndAnnouncesIt(t *testing.T) {
	fixture := newVerifyEmailFixture(t, testEmail)

	if err := fixture.clickLink(); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	if !fixture.stored(t).User.IsEmailVerified() {
		t.Error("IsEmailVerified() = false, want the user stamped verified")
	}
	fixture.assertChallengeSpent(t, true)
	fixture.events.OnlyDispatchedBy(t, eventstest.DispatchedBy{Label: events.UserEmailVerifiedLabel, UserID: fixture.userID})
}

func TestVerifyEmailSwitchesToAProvenNewAddressAndAnnouncesBothAddresses(t *testing.T) {
	fixture := newVerifyEmailFixture(t, requestedNewEmail)

	if err := fixture.clickLink(); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	stored := fixture.stored(t)
	if stored.User.Email != requestedNewEmail || !stored.User.IsEmailVerified() {
		t.Errorf("stored user = %+v, want it moved to a verified %q", stored.User, requestedNewEmail)
	}
	fixture.assertChallengeSpent(t, true)

	changed := fixture.events.OnlyDispatchedBy(t, eventstest.DispatchedBy{Label: events.UserEmailChangedLabel, UserID: fixture.userID})
	payload := struct {
		PreviousEmail string `json:"previousEmail"`
		Email         string `json:"email"`
	}{}
	if err := json.Unmarshal(changed.Data, &payload); err != nil {
		t.Fatalf("decoding the event: %v", err)
	}
	if payload.PreviousEmail != testEmail || payload.Email != requestedNewEmail {
		t.Errorf("payload = %+v, want a move from %q to %q", payload, testEmail, requestedNewEmail)
	}
}

func TestVerifyEmailAcceptsASecondClickWithoutAnnouncingAgain(t *testing.T) {
	for _, challengeTarget := range []string{testEmail, requestedNewEmail} {
		t.Run(challengeTarget, func(t *testing.T) {
			fixture := newVerifyEmailFixture(t, challengeTarget)

			if err := fixture.clickLink(); err != nil {
				t.Fatalf("the first VerifyEmail: %v", err)
			}
			if err := fixture.clickLink(); err != nil {
				t.Fatalf("the second VerifyEmail: %v, want a link clicked twice to succeed", err)
			}

			if len(fixture.events.Recorded) != 1 {
				t.Errorf("dispatched events = %d, want only the first click announced", len(fixture.events.Recorded))
			}
		})
	}
}

func TestVerifyEmailRejectsALinkThatProvesNothingForThisUser(t *testing.T) {
	cases := []struct {
		name    string
		arrange func(fixture *verifyEmailFixture) VerifyEmailParams
		wantErr func(err error) bool
	}{
		{
			name: "token matching no challenge",
			arrange: func(fixture *verifyEmailFixture) VerifyEmailParams {
				return VerifyEmailParams{VerificationToken: unmatchedVerificationToken, UserID: fixture.userID}
			},
			wantErr: isError[e.ErrVerificationLinkInvalid],
		},
		{
			name: "expired challenge",
			arrange: func(fixture *verifyEmailFixture) VerifyEmailParams {
				fixture.challenges.Challenges[0].ExpiresAt = expiredLinkExpiresAt
				return VerifyEmailParams{VerificationToken: presentedVerificationToken, UserID: fixture.userID}
			},
			wantErr: isError[e.ErrVerificationLinkInvalid],
		},
		{
			// The authenticated identity wins: another user's link must never stamp its owner.
			name: "challenge minted for another user",
			arrange: func(fixture *verifyEmailFixture) VerifyEmailParams {
				return VerifyEmailParams{VerificationToken: presentedVerificationToken, UserID: uuid.New()}
			},
			wantErr: isError[challenge.ErrChallengeOwnedByAnotherUser],
		},
		{
			name: "challenge minted for another purpose",
			arrange: func(fixture *verifyEmailFixture) VerifyEmailParams {
				fixture.challenges.Challenges[0].Purpose = challenge.PurposePasswordReset
				return VerifyEmailParams{VerificationToken: presentedVerificationToken, UserID: fixture.userID}
			},
			wantErr: isError[challenge.ErrChallengePurposeMismatch],
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newVerifyEmailFixture(t, testEmail)

			err := fixture.verify(testCase.arrange(fixture))

			if !testCase.wantErr(err) {
				t.Fatalf("error = %#v, want the case's rejection", err)
			}
			fixture.assertNothingWritten(t)
		})
	}
}

func TestVerifyEmailReportsALookupFailureRatherThanAnInvalidLink(t *testing.T) {
	cases := []struct {
		name    string
		arrange func(fixture *verifyEmailFixture)
		wantErr func(err error) bool
	}{
		{
			name: "challenge lookup",
			arrange: func(fixture *verifyEmailFixture) {
				fixture.challenges.FindFailure = challenge.ErrChallengeLookupFailed{Cause: errors.New("connection reset")}
			},
			wantErr: isError[challenge.ErrChallengeLookupFailed],
		},
		{
			name: "user lookup after an unmatched token",
			arrange: func(fixture *verifyEmailFixture) {
				fixture.challenges.Challenges = nil
				fixture.users.LookupFailure = e.ErrFindUserFailed{Cause: errors.New("connection refused")}
			},
			wantErr: isError[e.ErrFindUserFailed],
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newVerifyEmailFixture(t, testEmail)
			testCase.arrange(fixture)

			err := fixture.clickLink()

			if !testCase.wantErr(err) || errors.As(err, &e.ErrVerificationLinkInvalid{}) {
				t.Fatalf("error = %#v, want the lookup failure and not ErrVerificationLinkInvalid", err)
			}
			fixture.events.AssertNoneDispatched(t)
		})
	}
}

func TestVerifyEmailLeavesTheLinkLiveWhenTheAddressCannotBeWritten(t *testing.T) {
	cases := []struct {
		name            string
		challengeTarget string
		arrange         func(t *testing.T, fixture *verifyEmailFixture)
		wantErr         func(err error) bool
	}{
		{
			name:            "verification stamp fails",
			challengeTarget: testEmail,
			arrange: func(t *testing.T, fixture *verifyEmailFixture) {
				fixture.users.MarkVerifiedFailure = e.ErrMarkEmailVerifiedFailed{Cause: errors.New("disk full")}
			},
			wantErr: isError[e.ErrMarkEmailVerifiedFailed],
		},
		{
			name:            "switch fails",
			challengeTarget: requestedNewEmail,
			arrange: func(t *testing.T, fixture *verifyEmailFixture) {
				fixture.users.ChangeEmailFailure = e.ErrChangeEmailFailed{Cause: errors.New("disk full")}
			},
			wantErr: isError[e.ErrChangeEmailFailed],
		},
		{
			name:            "new address claimed since the request",
			challengeTarget: requestedNewEmail,
			arrange: func(t *testing.T, fixture *verifyEmailFixture) {
				seedUser(t, fixture.users, seedUserParams{email: requestedNewEmail})
			},
			wantErr: isError[e.ErrEmailTaken],
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newVerifyEmailFixture(t, testCase.challengeTarget)
			testCase.arrange(t, fixture)

			err := fixture.clickLink()

			if !testCase.wantErr(err) {
				t.Fatalf("error = %#v, want the case's failure", err)
			}
			fixture.assertNothingWritten(t)
		})
	}
}

func TestVerifyEmailKeepsTheWriteButAnnouncesNothingWhenTheLinkCannotBeSpent(t *testing.T) {
	for _, challengeTarget := range []string{testEmail, requestedNewEmail} {
		t.Run(challengeTarget, func(t *testing.T) {
			fixture := newVerifyEmailFixture(t, challengeTarget)
			fixture.challenges.MarkUsedFailure = challenge.ErrConsumeChallengeFailed{Cause: errors.New("deadlock detected")}

			err := fixture.clickLink()

			if !errors.As(err, &challenge.ErrConsumeChallengeFailed{}) {
				t.Fatalf("error = %v, want ErrConsumeChallengeFailed", err)
			}
			stored := fixture.stored(t)
			if !stored.User.IsEmailVerified() || stored.User.Email != challengeTarget {
				t.Errorf("stored user = %+v, want the proven %q kept even though spending failed", stored.User, challengeTarget)
			}
			fixture.events.AssertNoneDispatched(t)
		})
	}
}
