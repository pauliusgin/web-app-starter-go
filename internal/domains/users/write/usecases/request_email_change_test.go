package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	user "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
)

const requestedNewEmail = "bookkeeper@example.com"

const otherAccountEmail = "taken@example.com"

type requestEmailChangeForTest struct {
	dependencies RequestEmailChangeDependencies
	store        *inMemoryUserStore
	challenges   *inMemoryChallengeGenerator
	userID       uuid.UUID
}

func newRequestEmailChangeForTest(t *testing.T) requestEmailChangeForTest {
	t.Helper()

	store := newInMemoryUserStore()
	userID := uuid.New()
	store.savedUsers = append(store.savedUsers,
		user.NewUser{ID: userID, Email: testEmail, PasswordHash: seededHash},
		user.NewUser{ID: uuid.New(), Email: otherAccountEmail, PasswordHash: seededHash},
	)

	challenges := newInMemoryChallengeGenerator()

	return requestEmailChangeForTest{
		dependencies: RequestEmailChangeDependencies{
			UserWriteRepository:   store,
			GenerateAuthChallenge: challenges.generate,
		},
		store:      store,
		challenges: challenges,
		userID:     userID,
	}
}

func (h requestEmailChangeForTest) request(params RequestEmailChangeParams) error {
	return RequestEmailChange(context.Background(), h.dependencies, params)
}

func (h requestEmailChangeForTest) validParams() RequestEmailChangeParams {
	return RequestEmailChangeParams{
		UserID:   h.userID,
		NewEmail: requestedNewEmail,
	}
}

func assertEmailChangeNotRequested(t *testing.T, harness requestEmailChangeForTest) {
	t.Helper()

	if len(harness.challenges.requests) != 0 {
		t.Errorf("challenge requests = %d, want none for a rejected email change", len(harness.challenges.requests))
	}

	storedUser, err := harness.store.FindUserByID(context.Background(), harness.userID)
	if err != nil {
		t.Fatalf("finding the user: %v", err)
	}
	if storedUser.Email != testEmail {
		t.Errorf("stored email = %q, want %q left untouched", storedUser.Email, testEmail)
	}
}

func TestRequestEmailChangeAsksForAChallengeToTheNewAddress(t *testing.T) {
	harness := newRequestEmailChangeForTest(t)

	if err := harness.request(harness.validParams()); err != nil {
		t.Fatalf("RequestEmailChange: %v", err)
	}

	if len(harness.challenges.requests) != 1 {
		t.Fatalf("challenge requests = %d, want 1", len(harness.challenges.requests))
	}

	want := challenge.GenerateParams{
		UserID:     harness.userID,
		Target:     requestedNewEmail,
		TargetType: challenge.TargetTypeEmail,
		Purpose:    challenge.PurposeEmailVerification,
	}
	if harness.challenges.requests[0] != want {
		t.Errorf("challenge request = %+v, want %+v", harness.challenges.requests[0], want)
	}

	storedUser, err := harness.store.FindUserByID(context.Background(), harness.userID)
	if err != nil {
		t.Fatalf("finding the user: %v", err)
	}
	if storedUser.Email != testEmail {
		t.Errorf("stored email = %q, want %q kept until the new address is verified", storedUser.Email, testEmail)
	}
}

func TestRequestEmailChangeRejectsTheCurrentEmail(t *testing.T) {
	harness := newRequestEmailChangeForTest(t)
	params := harness.validParams()
	params.NewEmail = testEmail

	err := harness.request(params)

	unchanged := usererrors.ErrEmailUnchanged{}
	if !errors.As(err, &unchanged) {
		t.Fatalf("error = %v, want ErrEmailUnchanged", err)
	}
	if unchanged.Email != testEmail {
		t.Errorf("ErrEmailUnchanged.Email = %q, want %q", unchanged.Email, testEmail)
	}
	assertEmailChangeNotRequested(t, harness)
}

func TestRequestEmailChangeRejectsAnEmailAnotherAccountHolds(t *testing.T) {
	harness := newRequestEmailChangeForTest(t)
	params := harness.validParams()
	params.NewEmail = otherAccountEmail

	err := harness.request(params)

	taken := usererrors.ErrEmailTaken{}
	if !errors.As(err, &taken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
	if taken.Email != otherAccountEmail {
		t.Errorf("ErrEmailTaken.Email = %q, want %q", taken.Email, otherAccountEmail)
	}
	assertEmailChangeNotRequested(t, harness)
}

func TestRequestEmailChangeReportsAnUnknownUser(t *testing.T) {
	harness := newRequestEmailChangeForTest(t)
	params := harness.validParams()
	params.UserID = uuid.New()

	err := harness.request(params)

	if !errors.As(err, &usererrors.ErrUserNotFound{}) {
		t.Fatalf("error = %v, want ErrUserNotFound", err)
	}
	assertEmailChangeNotRequested(t, harness)
}

func TestRequestEmailChangePropagatesALookupFailure(t *testing.T) {
	harness := newRequestEmailChangeForTest(t)
	lookupFailure := usererrors.ErrIDLookupFailed{Cause: errors.New("pq: connection refused")}
	harness.store.lookupFailure = lookupFailure

	err := harness.request(harness.validParams())

	if err != lookupFailure {
		t.Fatalf("error = %#v, want the repository's own error passed through unchanged", err)
	}
	if len(harness.challenges.requests) != 0 {
		t.Errorf("challenge requests = %d, want none when the lookup fails", len(harness.challenges.requests))
	}
}

func TestRequestEmailChangeReportsAChallengeThatCouldNotBeGenerated(t *testing.T) {
	harness := newRequestEmailChangeForTest(t)
	generateFailure := challenge.ErrSaveChallengeFailed{Cause: errors.New("pq: connection refused")}
	harness.challenges.failure = generateFailure

	err := harness.request(harness.validParams())

	if err != generateFailure {
		t.Fatalf("error = %#v, want the generator's error, since no link reaches the new address", err)
	}
	assertEmailChangeNotRequested(t, harness)
}
