package integration

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
	userusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

func (w *integrationWorld) rotateRefreshToken(presentedToken string) (refreshtoken.RotateResult, error) {
	return refreshtoken.Rotate(w.ctx, refreshtoken.RotateDependencies{
		ReadRepository:  w.refreshTokenReadRepo,
		WriteRepository: w.refreshTokenWriteRepo,
	}, refreshtoken.RotateParams{PresentedToken: presentedToken})
}

func (w *integrationWorld) revokeRefreshToken(presentedToken string) error {
	return refreshtoken.Revoke(w.ctx, refreshtoken.RevokeDependencies{
		ReadRepository:  w.refreshTokenReadRepo,
		WriteRepository: w.refreshTokenWriteRepo,
	}, refreshtoken.RevokeParams{PresentedToken: presentedToken})
}

func (w *integrationWorld) refreshTokenRowCount(t *testing.T, userID uuid.UUID) int {
	t.Helper()

	var count int
	err := w.database.QueryRowContext(w.ctx, `SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		t.Fatalf("counting refresh tokens: %v", err)
	}

	return count
}

func TestRefreshTokenRotatesOnceAndTheReplacementKeepsWorking(t *testing.T) {
	world := newIntegrationWorld(t)
	userID := world.verifiedUserID(t)
	issued := world.issueRefreshToken(t, userID)

	rotated, err := world.rotateRefreshToken(issued.Plaintext)
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if rotated.UserID != userID {
		t.Errorf("UserID = %s, want %s", rotated.UserID, userID)
	}

	if _, err := world.rotateRefreshToken(issued.Plaintext); !errors.As(err, &refreshtoken.ErrRefreshTokenNotFound{}) {
		t.Errorf("replaying the rotated token err = %v, want ErrRefreshTokenNotFound", err)
	}
	if _, err := world.rotateRefreshToken(rotated.Plaintext); err != nil {
		t.Errorf("rotating the replacement: %v", err)
	}
}

func TestRevokedRefreshTokenCannotBeRotated(t *testing.T) {
	world := newIntegrationWorld(t)
	issued := world.issueRefreshToken(t, world.verifiedUserID(t))

	if err := world.revokeRefreshToken(issued.Plaintext); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	if _, err := world.rotateRefreshToken(issued.Plaintext); !errors.As(err, &refreshtoken.ErrRefreshTokenNotFound{}) {
		t.Errorf("rotating a revoked token err = %v, want ErrRefreshTokenNotFound", err)
	}
}

func TestDeletingUnusedRefreshTokensKeepsLiveAndRecentlyRevokedOnes(t *testing.T) {
	world := newIntegrationWorld(t)
	userID := world.verifiedUserID(t)
	world.issueRefreshToken(t, userID)
	recentlyRevoked := world.issueRefreshToken(t, userID)
	longRevoked := world.issueRefreshToken(t, userID)
	expired := world.issueRefreshToken(t, userID)
	for _, revoked := range []refreshtoken.IssueResult{recentlyRevoked, longRevoked} {
		if err := world.revokeRefreshToken(revoked.Plaintext); err != nil {
			t.Fatalf("Revoke: %v", err)
		}
	}
	world.backdateRefreshToken(t, backdateRefreshTokenParams{issued: longRevoked, column: "revoked_at", daysAgo: 8})
	world.backdateRefreshToken(t, backdateRefreshTokenParams{issued: expired, column: "expires_at", daysAgo: 1})

	deleted, err := refreshtoken.DeleteUnusedRefreshTokens(world.ctx, refreshtoken.DeleteUnusedRefreshTokensDependencies{
		WriteRepository: world.refreshTokenWriteRepo,
	}, refreshtoken.DeleteUnusedRefreshTokensParams{CleanUpThresholdInDays: 7})
	if err != nil {
		t.Fatalf("DeleteUnusedRefreshTokens: %v", err)
	}

	if deleted != 2 {
		t.Errorf("deleted = %d, want the long-revoked and the expired token", deleted)
	}
	if remaining := world.refreshTokenRowCount(t, userID); remaining != 2 {
		t.Errorf("remaining tokens = %d, want the live and the recently revoked one", remaining)
	}
}

type backdateRefreshTokenParams struct {
	issued  refreshtoken.IssueResult
	column  string
	daysAgo int
}

// backdateRefreshToken moves a token's clock column into the past, standing in for days going by.
func (w *integrationWorld) backdateRefreshToken(t *testing.T, params backdateRefreshTokenParams) {
	t.Helper()

	statement := `UPDATE refresh_tokens SET ` + params.column + ` = NOW() - ($1 * INTERVAL '1 day') WHERE token_hash = $2`
	if _, err := w.database.ExecContext(w.ctx, statement, params.daysAgo, token.Hash(params.issued.Plaintext)); err != nil {
		t.Fatalf("backdating %s: %v", params.column, err)
	}
}

func TestVerificationLinkIsSpentButAClickAfterVerifyingStillSucceeds(t *testing.T) {
	world := newIntegrationWorld(t)
	signedUp := world.signUp(t, "member@example.com")
	token := world.latestChallengeToken(t, "member@example.com")
	verifyParams := userusecases.VerifyEmailParams{VerificationToken: token, UserID: signedUp.User.ID}

	if err := world.verifyEmail(verifyParams); err != nil {
		t.Fatalf("first VerifyEmail: %v", err)
	}

	var usedAtSet bool
	err := world.database.QueryRowContext(world.ctx,
		`SELECT used_at IS NOT NULL FROM auth_challenges WHERE user_id = $1`, signedUp.User.ID).Scan(&usedAtSet)
	if err != nil {
		t.Fatalf("reading the challenge: %v", err)
	}
	if !usedAtSet {
		t.Error("used_at = NULL, want the challenge spent")
	}

	if err := world.verifyEmail(verifyParams); err != nil {
		t.Errorf("second VerifyEmail = %v, want an already verified user to succeed", err)
	}
}
