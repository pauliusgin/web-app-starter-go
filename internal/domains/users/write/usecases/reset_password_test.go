package usecases

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

const presentedResetToken = "Rt4pW8kL1zX6cV9bN2mQ5sD7fG0hJ3aE"

const resetUserSessionTokenHash = "reset-user-session-hash"

const otherUserSessionTokenHash = "other-user-session-hash"

type resetPasswordForTest struct {
	users         *repositories.InMemoryUserRepository
	challenges    *challenge.InMemoryChallengeRepository
	refreshTokens *refreshtoken.InMemoryRefreshTokenRepository
	events        *eventstest.CapturedEvents
	userID        uuid.UUID
}

// newResetPasswordForTest seeds the user on testEmail with a session, a live reset link mailed to testEmail
// and another user's session.
func newResetPasswordForTest(t *testing.T, seeded seedUserParams) *resetPasswordForTest {
	t.Helper()

	users := repositories.NewInMemoryUserRepository()
	userID := seedUser(t, users, seeded)

	return &resetPasswordForTest{
		users: users,
		challenges: challenge.NewInMemoryChallengeRepository(challenge.Challenge{
			ID:         uuid.New(),
			UserID:     userID,
			TargetType: challenge.TargetTypeEmail,
			Target:     testEmail,
			Purpose:    challenge.PurposePasswordReset,
			TokenHash:  token.Hash(presentedResetToken),
			CreatedAt:  fixedSeededAt,
			ExpiresAt:  liveLinkExpiresAt,
		}),
		refreshTokens: refreshtoken.NewInMemoryRefreshTokenRepository(
			refreshtoken.RefreshToken{
				ID: uuid.New(), UserID: userID, TokenHash: resetUserSessionTokenHash,
				CreatedAt: fixedSeededAt, ExpiresAt: liveLinkExpiresAt,
			},
			refreshtoken.RefreshToken{
				ID: uuid.New(), UserID: uuid.New(), TokenHash: otherUserSessionTokenHash,
				CreatedAt: fixedSeededAt, ExpiresAt: liveLinkExpiresAt,
			},
		),
		events: &eventstest.CapturedEvents{},
		userID: userID,
	}
}

func (h *resetPasswordForTest) reset(presentedToken string) error {
	return ResetPassword(context.Background(), ResetPasswordDependencies{
		UserWriteRepository:          h.users,
		EventDispatcher:              h.events.Dispatcher(),
		FindAuthChallengeByTokenHash: h.challenges.FindByTokenHash,
		ConsumeAuthChallenge:         h.challenges.MarkUsed,
		DeleteUserRefreshTokens: func(ctx context.Context, params refreshtoken.DeleteUserRefreshTokensParams) error {
			return refreshtoken.DeleteUserRefreshTokens(ctx,
				refreshtoken.DeleteUserRefreshTokensDependencies{WriteRepository: h.refreshTokens}, params)
		},
	}, ResetPasswordParams{Token: presentedToken, NewPassword: replacementPassword})
}

func (h *resetPasswordForTest) dispatchedLabels() []events.Label {
	labels := []events.Label{}
	for _, published := range h.events.Published {
		labels = append(labels, published.Label)
	}

	return labels
}

func (h *resetPasswordForTest) assertSessionKept(t *testing.T, tokenHash string, wantKept bool) {
	t.Helper()

	if _, kept := h.refreshTokens.Tokens[tokenHash]; kept != wantKept {
		t.Errorf("session %q kept = %t, want %t", tokenHash, kept, wantKept)
	}
}

type resetPasswordSnapshot struct {
	passwordHash string
	linkUsedAt   time.Time
}

func (h *resetPasswordForTest) snapshot() resetPasswordSnapshot {
	return resetPasswordSnapshot{
		passwordHash: h.users.Users[h.userID].PasswordHash,
		linkUsedAt:   h.challenges.Challenges[0].UsedAt,
	}
}

func (h *resetPasswordForTest) assertNothingWritten(t *testing.T, beforeReset resetPasswordSnapshot) {
	t.Helper()

	if h.snapshot() != beforeReset {
		t.Errorf("after reset = %+v, want it left as %+v", h.snapshot(), beforeReset)
	}
	h.assertSessionKept(t, resetUserSessionTokenHash, true)
	h.events.AssertNoneDispatched(t)
}

func TestResetPasswordStoresTheNewPasswordSpendsTheLinkAndEndsTheUsersSessions(t *testing.T) {
	harness := newResetPasswordForTest(t, seedUserParams{email: testEmail, verified: true})

	if err := harness.reset(presentedResetToken); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	if !hashVerifies(t, storedUser(t, harness.users, harness.userID).PasswordHash, replacementPassword) {
		t.Error("stored hash does not verify the new password")
	}
	if !harness.challenges.Challenges[0].IsUsed() {
		t.Error("reset link left unused, want it spent")
	}
	harness.assertSessionKept(t, resetUserSessionTokenHash, false)
	harness.assertSessionKept(t, otherUserSessionTokenHash, true)
	if storedUser(t, harness.users, harness.userID).User.EmailVerifiedAt != fixedSeededAt {
		t.Error("EmailVerifiedAt changed, want the earlier verification kept")
	}
	harness.events.OnlyDispatchedBy(t, eventstest.DispatchedBy{Label: events.UserPasswordResetLabel, UserID: harness.userID})
}

func TestResetPasswordMarksAnUnverifiedEmailVerifiedSinceTheLinkReachedTheInbox(t *testing.T) {
	harness := newResetPasswordForTest(t, seedUserParams{email: testEmail})

	if err := harness.reset(presentedResetToken); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	if !storedUser(t, harness.users, harness.userID).User.IsEmailVerified() {
		t.Error("IsEmailVerified() = false, want the reset to verify the email")
	}
	wantLabels := []events.Label{events.UserEmailVerifiedLabel, events.UserPasswordResetLabel}
	if !slices.Equal(harness.dispatchedLabels(), wantLabels) {
		t.Errorf("dispatched = %v, want %v", harness.dispatchedLabels(), wantLabels)
	}
}

func TestResetPasswordRejectsALinkThatNoLongerResetsTheAccount(t *testing.T) {
	cases := []struct {
		name           string
		presentedToken string
		arrange        func(t *testing.T, harness *resetPasswordForTest)
	}{
		{
			name:           "unknown token",
			presentedToken: unmatchedVerificationToken,
			arrange:        func(t *testing.T, harness *resetPasswordForTest) {},
		},
		{
			name: "expired link",
			arrange: func(t *testing.T, harness *resetPasswordForTest) {
				harness.challenges.Challenges[0].ExpiresAt = expiredLinkExpiresAt
			},
		},
		{
			name: "spent link",
			arrange: func(t *testing.T, harness *resetPasswordForTest) {
				harness.challenges.Challenges[0].UsedAt = fixedSeededAt
			},
		},
		{
			name: "email verification link",
			arrange: func(t *testing.T, harness *resetPasswordForTest) {
				harness.challenges.Challenges[0].Purpose = challenge.PurposeEmailVerification
			},
		},
		{
			name: "account moved to another email since",
			arrange: func(t *testing.T, harness *resetPasswordForTest) {
				stored := storedUser(t, harness.users, harness.userID)
				stored.User.Email = requestedNewEmail
				harness.users.Users[harness.userID] = stored
			},
		},
		{
			name: "account without a password",
			arrange: func(t *testing.T, harness *resetPasswordForTest) {
				stored := storedUser(t, harness.users, harness.userID)
				stored.PasswordHash = ""
				stored.GoogleSubject = testGoogleSubject
				harness.users.Users[harness.userID] = stored
			},
		},
		{
			name: "deleted account",
			arrange: func(t *testing.T, harness *resetPasswordForTest) {
				delete(harness.users.Users, harness.userID)
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newResetPasswordForTest(t, seedUserParams{email: testEmail})
			testCase.arrange(t, harness)
			beforeReset := harness.snapshot()
			presentedToken := testCase.presentedToken
			if presentedToken == "" {
				presentedToken = presentedResetToken
			}

			err := harness.reset(presentedToken)

			if !isError[e.ErrPasswordResetLinkInvalid](err) {
				t.Fatalf("error = %v, want ErrPasswordResetLinkInvalid", err)
			}
			harness.assertNothingWritten(t, beforeReset)
		})
	}
}

func TestResetPasswordReportsStoreFailuresWithoutAnnouncingTheReset(t *testing.T) {
	storeFailure := errors.New("connection reset by peer")

	cases := []struct {
		name    string
		arrange func(harness *resetPasswordForTest)
	}{
		{name: "link lookup", arrange: func(harness *resetPasswordForTest) { harness.challenges.FindFailure = storeFailure }},
		{name: "user lookup", arrange: func(harness *resetPasswordForTest) { harness.users.LookupFailure = storeFailure }},
		{name: "password write", arrange: func(harness *resetPasswordForTest) { harness.users.ChangePasswordFailure = storeFailure }},
		{name: "link spend", arrange: func(harness *resetPasswordForTest) { harness.challenges.MarkUsedFailure = storeFailure }},
		{name: "session delete", arrange: func(harness *resetPasswordForTest) { harness.refreshTokens.DeleteFailure = storeFailure }},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newResetPasswordForTest(t, seedUserParams{email: testEmail, verified: true})
			testCase.arrange(harness)

			err := harness.reset(presentedResetToken)

			if !errors.Is(err, storeFailure) {
				t.Fatalf("error = %v, want the store failure", err)
			}
			for _, label := range harness.dispatchedLabels() {
				if label == events.UserPasswordResetLabel {
					t.Error("user.password_reset dispatched, want no announcement for a failed reset")
				}
			}
		})
	}
}
