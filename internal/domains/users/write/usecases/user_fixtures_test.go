package usecases

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

const testEmail = "accountant@example.com"

const testPassword = "correct horse battery staple"

const requestedNewEmail = "bookkeeper@example.com"

// unreadableHash carries the argon2id prefix but no parameters, so verifying against it errors.
const unreadableHash = "$argon2id$seeded"

var fixedSeededAt = time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)

type seedUserParams struct {
	email    string
	password string
	verified bool
}

// seedUser stores a user directly; without a password it gets unreadableHash, skipping argon2id's cost.
func seedUser(t *testing.T, users *repositories.InMemoryUserRepository, params seedUserParams) uuid.UUID {
	t.Helper()

	passwordHash := unreadableHash
	if params.password != "" {
		hashed, err := password.Hash(params.password)
		if err != nil {
			t.Fatalf("hashing the seeded password: %v", err)
		}
		passwordHash = hashed
	}

	seeded := models.User{
		ID:        uuid.New(),
		Email:     params.email,
		CreatedAt: fixedSeededAt,
		UpdatedAt: fixedSeededAt,
		SigninAt:  fixedSeededAt,
	}
	if params.verified {
		seeded.EmailVerifiedAt = fixedSeededAt
	}
	users.Users[seeded.ID] = repositories.InMemoryUser{User: seeded, PasswordHash: passwordHash}

	return seeded.ID
}

const testGoogleSubject = "109876543210987654321"

// seedGoogleUser stores a verified, password-less user signed up with the google account.
func seedGoogleUser(t *testing.T, users *repositories.InMemoryUserRepository, googleSubject string) uuid.UUID {
	t.Helper()

	seeded := models.User{
		ID:              uuid.New(),
		Email:           testEmail,
		CreatedAt:       fixedSeededAt,
		UpdatedAt:       fixedSeededAt,
		EmailVerifiedAt: fixedSeededAt,
		SigninAt:        fixedSeededAt,
	}
	users.Users[seeded.ID] = repositories.InMemoryUser{User: seeded, GoogleSubject: googleSubject}

	return seeded.ID
}

func storedUser(t *testing.T, users *repositories.InMemoryUserRepository, userID uuid.UUID) repositories.InMemoryUser {
	t.Helper()

	stored, found := users.Users[userID]
	if !found {
		t.Fatalf("user %s is not stored", userID)
	}

	return stored
}

// challengeLinks runs the real challenge.Generate over an in-memory auth_challenges store.
type challengeLinks struct {
	repository *challenge.InMemoryChallengeRepository
}

func newChallengeLinks() challengeLinks {
	return challengeLinks{repository: challenge.NewInMemoryChallengeRepository()}
}

func (l challengeLinks) generate(ctx context.Context, params challenge.GenerateParams) (challenge.GenerateResult, error) {
	return challenge.Generate(ctx, challenge.GenerateDependencies{
		WriteRepository: l.repository,
		EventDispatcher: (&eventstest.CapturedEvents{}).Dispatcher(),
	}, params)
}

func (l challengeLinks) assertNoneSent(t *testing.T) {
	t.Helper()

	if len(l.repository.Challenges) != 0 {
		t.Errorf("stored challenges = %+v, want none", l.repository.Challenges)
	}
}

// assertOnlyLinkSent fails unless the store holds exactly one live challenge matching want.
func (l challengeLinks) assertOnlyLinkSent(t *testing.T, want challenge.GenerateParams) {
	t.Helper()

	if len(l.repository.Challenges) != 1 {
		t.Fatalf("stored challenges = %d, want exactly one", len(l.repository.Challenges))
	}
	stored := l.repository.Challenges[0]
	sent := challenge.GenerateParams{
		UserID:     stored.UserID,
		TargetType: stored.TargetType,
		Target:     stored.Target,
		Purpose:    stored.Purpose,
	}
	if sent != want || stored.IsUsed() || stored.TokenHash == "" {
		t.Errorf("stored challenge = %+v, want an unused one for %+v", stored, want)
	}
}

// inMemoryIssuanceGuard stands in for challenge.EnsureIssuanceAllowed, whose rules are tested in its package.
type inMemoryIssuanceGuard struct {
	rateLimitedTargets map[string]bool
}

func (g inMemoryIssuanceGuard) ensureAllowed(_ context.Context, params challenge.EnsureIssuanceAllowedParams) error {
	if g.rateLimitedTargets[params.Target] {
		return challenge.ErrIssuanceRateLimited{RetryAfter: 20 * time.Second}
	}

	return nil
}

// isError reports whether err is, or wraps, an E; table cases name the rejection they expect with it.
func isError[E error](err error) bool {
	_, matched := errors.AsType[E](err)

	return matched
}

func silenceLogger(t *testing.T) *bytes.Buffer {
	t.Helper()

	captured := &bytes.Buffer{}
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(captured, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	return captured
}
