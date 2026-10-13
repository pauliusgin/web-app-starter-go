package tokens

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
)

const presentedRefreshTokenForTest = "Xh2q8kU1n0dJ6oQpZ4tR7yC3vB5mN9wL1aS8dF0gH2k"

func silenceRefreshTokenLogger(t *testing.T) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}

func revokeWithUseCase(
	t *testing.T,
	useCase RevokeRefreshTokenUseCase,
) *httptest.ResponseRecorder {
	t.Helper()

	handler := RevokeRefreshToken(
		RevokeRefreshTokenDependencies{RevokeRefreshToken: useCase})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tokens/revoke", nil)
	request.Header.Set("Authorization", "Bearer "+presentedRefreshTokenForTest)

	handler.ServeHTTP(recorder, request)

	return recorder
}

func TestRevokeRefreshTokenAnswersOKWhenTheLiveTokenIsRevoked(t *testing.T) {
	var revokedTokens []string

	recorder := revokeWithUseCase(t, func(_ context.Context, params refreshtoken.RevokeParams) error {
		revokedTokens = append(revokedTokens, params.PresentedToken)
		return nil
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if len(revokedTokens) != 1 || revokedTokens[0] != presentedRefreshTokenForTest {
		t.Errorf("revocations = %v, want the one presented token", revokedTokens)
	}
}

func TestRevokeRefreshTokenAnswersOKWhenTheTokenIsAlreadyGone(t *testing.T) {
	recorder := revokeWithUseCase(t, func(context.Context, refreshtoken.RevokeParams) error {
		return refreshtoken.ErrRefreshTokenNotFound{}
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d on a repeated logout", recorder.Code, http.StatusOK)
	}
	if challenge := recorder.Header().Get("WWW-Authenticate"); challenge != "" {
		t.Errorf("WWW-Authenticate = %q, want no challenge", challenge)
	}
}

func TestRevokeRefreshTokenAnswersServerErrorWhenTheDatabaseFails(t *testing.T) {
	silenceRefreshTokenLogger(t)

	recorder := revokeWithUseCase(t, func(context.Context, refreshtoken.RevokeParams) error {
		return refreshtoken.ErrRevokeRefreshTokenFailed{Cause: context.DeadlineExceeded}
	})

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}
