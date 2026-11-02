package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

type signupForTest struct {
	users      *repositories.InMemoryUserRepository
	events     *eventstest.CapturedEvents
	challenges challengeLinks
}

func newSignupForTest() signupForTest {
	return signupForTest{
		users:      repositories.NewInMemoryUserRepository(),
		events:     &eventstest.CapturedEvents{},
		challenges: newChallengeLinks(),
	}
}

func (h signupForTest) signUp() (SignUpUserResult, error) {
	return SignUpUser(context.Background(), SignUpUserDependencies{
		UserWriteRepository:   h.users,
		EventDispatcher:       h.events.Dispatcher(),
		GenerateAuthChallenge: h.challenges.generate,
	}, SignUpUserParams{Email: testEmail, Password: testPassword})
}

func TestSignupStoresTheUserAnnouncesItAndSendsAVerificationLink(t *testing.T) {
	harness := newSignupForTest()

	result, err := harness.signUp()
	if err != nil {
		t.Fatalf("SignUpUser: %v", err)
	}

	stored := storedUser(t, harness.users, result.User.ID)
	if stored.User.Email != testEmail {
		t.Errorf("stored Email = %q, want %q", stored.User.Email, testEmail)
	}
	harness.events.OnlyDispatchedBy(t, eventstest.DispatchedBy{Label: events.UserSignedUpLabel, UserID: result.User.ID})

	wantLink := challenge.GenerateParams{
		UserID:     result.User.ID,
		Target:     testEmail,
		TargetType: challenge.TargetTypeEmail,
		Purpose:    challenge.PurposeEmailVerification,
	}
	harness.challenges.assertOnlyLinkSent(t, wantLink)
}

func TestSignupStoresAnArgon2idHashAndNeverThePlaintext(t *testing.T) {
	harness := newSignupForTest()

	result, err := harness.signUp()
	if err != nil {
		t.Fatalf("SignUpUser: %v", err)
	}

	storedHash := storedUser(t, harness.users, result.User.ID).PasswordHash
	if !strings.HasPrefix(storedHash, "$argon2id$") || strings.Contains(storedHash, testPassword) {
		t.Errorf("stored hash = %q, want an argon2id hash without the plaintext", storedHash)
	}
	matches, err := password.Verify(password.VerificationParams{Password: testPassword, Hash: storedHash})
	if err != nil || !matches {
		t.Errorf("Verify = %t, %v, want the stored hash to verify the signup password", matches, err)
	}
}

func TestSignupRejectsAnEmailThatAlreadyHasAnAccount(t *testing.T) {
	harness := newSignupForTest()
	seedUser(t, harness.users, seedUserParams{email: testEmail})

	_, err := harness.signUp()

	emailTaken := e.ErrEmailTaken{}
	if !errors.As(err, &emailTaken) || emailTaken.Email != testEmail {
		t.Fatalf("error = %v, want ErrEmailTaken for %q", err, testEmail)
	}
	if len(harness.users.Users) != 1 {
		t.Errorf("stored users = %d, want only the account already there", len(harness.users.Users))
	}
	harness.events.AssertNoneDispatched(t)
	harness.challenges.assertNoneSent(t)
}

func TestSignupReportsALookupFailureRatherThanATakenEmail(t *testing.T) {
	harness := newSignupForTest()
	harness.users.LookupFailure = e.ErrFindUserFailed{Cause: errors.New("connection refused")}

	_, err := harness.signUp()

	if !errors.As(err, &e.ErrFindUserFailed{}) || errors.As(err, &e.ErrEmailTaken{}) {
		t.Fatalf("error = %v, want ErrFindUserFailed and not ErrEmailTaken", err)
	}
	if len(harness.users.Users) != 0 {
		t.Errorf("stored users = %d, want nothing stored when the lookup fails", len(harness.users.Users))
	}
}

func TestSignupAnnouncesNothingWhenTheUserIsNotStored(t *testing.T) {
	harness := newSignupForTest()
	harness.users.SaveFailure = e.ErrSaveUserFailed{Cause: errors.New("disk full")}

	_, err := harness.signUp()

	if !errors.As(err, &e.ErrSaveUserFailed{}) {
		t.Fatalf("error = %v, want ErrSaveUserFailed", err)
	}
	harness.events.AssertNoneDispatched(t)
	harness.challenges.assertNoneSent(t)
}

func TestSignupSucceedsAndLogsWhenTheVerificationLinkCannotBeCreated(t *testing.T) {
	logs := silenceLogger(t)
	harness := newSignupForTest()
	harness.challenges.repository.SaveFailure = errors.New("pq: relation \"auth_challenges\" does not exist")

	result, err := harness.signUp()
	if err != nil {
		t.Fatalf("SignUpUser = %v, want a failed challenge to leave the signup successful", err)
	}

	storedUser(t, harness.users, result.User.ID)
	for _, want := range []string{"failed generating the email verification challenge", "auth_challenges"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("logs = %q, want them to carry %q", logs, want)
		}
	}
}
