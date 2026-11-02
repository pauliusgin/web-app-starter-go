package users

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

const presentedResetToken = "Rt4pW8kL1zX6cV9bN2mQ5sD7fG0hJ3aE"

const resetPasswordForTest = "reset-ledger-password"

func (s *userStores) resetPasswordHandler() http.Handler {
	refreshTokens := refreshtoken.NewInMemoryRefreshTokenRepository()

	return ResetPassword(ResetPasswordDependencies{
		ResetPassword: func(ctx context.Context, params usecases.ResetPasswordParams) error {
			return usecases.ResetPassword(ctx, usecases.ResetPasswordDependencies{
				UserWriteRepository:          s.users,
				EventDispatcher:              ignoringDispatcher(),
				FindAuthChallengeByTokenHash: s.challenges.FindByTokenHash,
				ConsumeAuthChallenge:         s.challenges.MarkUsed,
				DeleteUserRefreshTokens: func(ctx context.Context, params refreshtoken.DeleteUserRefreshTokensParams) error {
					return refreshtoken.DeleteUserRefreshTokens(ctx,
						refreshtoken.DeleteUserRefreshTokensDependencies{WriteRepository: refreshTokens}, params)
				},
			}, params)
		},
	})
}

func (l challengeLinks) seedPasswordResetLink(userID uuid.UUID, plaintextToken string) {
	l.Challenges = append(l.Challenges, challenge.Challenge{
		ID:         uuid.New(),
		UserID:     userID,
		TargetType: challenge.TargetTypeEmail,
		Target:     testEmail,
		Purpose:    challenge.PurposePasswordReset,
		TokenHash:  token.Hash(plaintextToken),
		ExpiresAt:  liveLinkExpiresAt,
	})
}

func sendResetPassword(t *testing.T, stores *userStores, body string) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler: stores.resetPasswordHandler(),
		method:  http.MethodPost,
		path:    "/api/v1/users/reset-password",
		body:    body,
	})
}

func resetPasswordBody(presentedToken string, newPassword string) string {
	return `{"token":"` + presentedToken + `","newPassword":"` + newPassword + `"}`
}

func TestResetPasswordStoresTheNewPassword(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail, verified: true})
	stores.challenges.seedPasswordResetLink(userID, presentedResetToken)

	response := sendResetPassword(t, stores, resetPasswordBody(presentedResetToken, resetPasswordForTest))

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	matches, err := password.Verify(password.VerificationParams{
		Password: resetPasswordForTest,
		Hash:     stores.users.Users[userID].PasswordHash,
	})
	if err != nil || !matches {
		t.Errorf("stored hash verifies the new password = %t (%v), want true", matches, err)
	}
}

func TestResetPasswordRejectsWithoutChangingThePassword(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   handlers.ErrorCode
	}{
		{name: "malformed json", body: `{"token":`, wantStatus: http.StatusBadRequest, wantCode: handlers.ErrorCodeRequestBodyInvalid},
		{
			name:       "token missing",
			body:       resetPasswordBody("  ", resetPasswordForTest),
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordResetTokenMissing,
		},
		{
			name:       "password too short",
			body:       resetPasswordBody(presentedResetToken, "short"),
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordTooShort,
		},
		{
			name:       "password too long",
			body:       resetPasswordBody(presentedResetToken, strings.Repeat("a", 65)),
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordTooLong,
		},
		{
			name:       "password with whitespace",
			body:       resetPasswordBody(presentedResetToken, "reset ledger password"),
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordContainsWhitespace,
		},
		{
			name:       "unknown link",
			body:       resetPasswordBody("Zx1cV4bN7mQ0wE3rT6yU9iO2pA5sD8fG", resetPasswordForTest),
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordResetLinkInvalid,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stores := newUserStores(t)
			userID := stores.seedUser(t, seedUserParams{email: testEmail, verified: true})
			stores.challenges.seedPasswordResetLink(userID, presentedResetToken)
			seeded := stores.users.Users[userID]

			response := sendResetPassword(t, stores, testCase.body)

			assertErrorCode(t, response, testCase.wantStatus, testCase.wantCode)
			if stores.users.Users[userID] != seeded || stores.challenges.Challenges[0].IsUsed() {
				t.Error("the stores changed, want a rejected reset to touch nothing")
			}
		})
	}
}

func TestResetPasswordAnswersAFailedLookupWithAGenericError(t *testing.T) {
	silenceHandlerLogger(t)
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail})
	stores.challenges.seedPasswordResetLink(userID, presentedResetToken)
	stores.users.LookupFailure = errDatabaseDown

	response := sendResetPassword(t, stores, resetPasswordBody(presentedResetToken, resetPasswordForTest))

	assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
}
