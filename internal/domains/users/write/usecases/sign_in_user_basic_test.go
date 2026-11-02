package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

type signInForTest struct {
	users  *repositories.InMemoryUserRepository
	events *eventstest.CapturedEvents
	userID uuid.UUID
}

func newSignInForTest(t *testing.T) signInForTest {
	t.Helper()

	users := repositories.NewInMemoryUserRepository()

	return signInForTest{
		users:  users,
		events: &eventstest.CapturedEvents{},
		userID: seedUser(t, users, seedUserParams{email: testEmail, password: testPassword}),
	}
}

func (h signInForTest) signIn(params SignInUserParams) (SignInUserResult, error) {
	return SignInUser(context.Background(), SignInUserDependencies{
		UserWriteRepository: h.users,
		EventDispatcher:     h.events.Dispatcher(),
	}, params)
}

func (h signInForTest) assertSignInRejected(t *testing.T) {
	t.Helper()

	if signedInAt := storedUser(t, h.users, h.userID).User.SigninAt; !signedInAt.Equal(fixedSeededAt) {
		t.Errorf("SigninAt = %v, want the seeded %v left alone by a rejected sign-in", signedInAt, fixedSeededAt)
	}
	h.events.AssertNoneDispatched(t)
}

func TestSignInStampsAndAnnouncesAnUnverifiedUsersSignIn(t *testing.T) {
	harness := newSignInForTest(t)

	result, err := harness.signIn(SignInUserParams{Email: testEmail, Password: testPassword})
	if err != nil {
		t.Fatalf("SignInUser: %v, want an unverified email to sign in", err)
	}

	if result.User.ID != harness.userID {
		t.Errorf("ID = %s, want %s", result.User.ID, harness.userID)
	}
	if signedInAt := storedUser(t, harness.users, harness.userID).User.SigninAt; !signedInAt.After(fixedSeededAt) {
		t.Errorf("SigninAt = %v, want it stamped past the seeded %v", signedInAt, fixedSeededAt)
	}
	harness.events.OnlyDispatchedBy(t, eventstest.DispatchedBy{Label: events.UserSignedInLabel, UserID: harness.userID})
}

func TestSignInRejectsAWrongPasswordAndAnUnknownEmailAlike(t *testing.T) {
	cases := []struct {
		name   string
		params SignInUserParams
	}{
		{name: "wrong password", params: SignInUserParams{Email: testEmail, Password: "wrong horse battery"}},
		{name: "unknown email", params: SignInUserParams{Email: "stranger@example.com", Password: testPassword}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newSignInForTest(t)

			_, err := harness.signIn(testCase.params)

			if !errors.As(err, &e.ErrCredentialsInvalid{}) {
				t.Fatalf("error = %v, want ErrCredentialsInvalid", err)
			}
			harness.assertSignInRejected(t)
		})
	}
}

func TestSignInReportsALookupFailureRatherThanInvalidCredentials(t *testing.T) {
	harness := newSignInForTest(t)
	harness.users.LookupFailure = e.ErrFindUserFailed{Cause: errors.New("connection refused")}

	_, err := harness.signIn(SignInUserParams{Email: testEmail, Password: testPassword})

	if !errors.As(err, &e.ErrFindUserFailed{}) || errors.As(err, &e.ErrCredentialsInvalid{}) {
		t.Fatalf("error = %v, want ErrFindUserFailed and not ErrCredentialsInvalid", err)
	}
	harness.users.LookupFailure = nil
	harness.assertSignInRejected(t)
}

func TestSignInReportsAnUnreadableStoredHash(t *testing.T) {
	harness := newSignInForTest(t)
	stored := storedUser(t, harness.users, harness.userID)
	stored.PasswordHash = unreadableHash
	harness.users.Users[harness.userID] = stored

	_, err := harness.signIn(SignInUserParams{Email: testEmail, Password: testPassword})

	if !errors.As(err, &e.ErrPasswordVerificationFailed{}) {
		t.Fatalf("error = %v, want ErrPasswordVerificationFailed", err)
	}
	harness.assertSignInRejected(t)
}
