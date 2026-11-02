package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

type signInGoogleForTest struct {
	users    *repositories.InMemoryUserRepository
	events   *eventstest.CapturedEvents
	verifier googleidtoken.InMemoryVerifier
	userID   uuid.UUID
}

func newSignInGoogleForTest(t *testing.T) signInGoogleForTest {
	t.Helper()

	users := repositories.NewInMemoryUserRepository()

	return signInGoogleForTest{
		users:    users,
		events:   &eventstest.CapturedEvents{},
		verifier: newGoogleVerifier(),
		userID:   seedGoogleUser(t, users, testGoogleSubject),
	}
}

func (h signInGoogleForTest) signIn(idToken string) (SignInUserGoogleResult, error) {
	return SignInUserGoogle(context.Background(), SignInUserGoogleDependencies{
		UserWriteRepository: h.users,
		EventDispatcher:     h.events.Dispatcher(),
		VerifyGoogleIDToken: h.verifier.VerifyIDToken,
	}, SignInUserGoogleParams{IDToken: idToken})
}

func (h signInGoogleForTest) assertSignInRejected(t *testing.T, result SignInUserGoogleResult) {
	t.Helper()

	if result != (SignInUserGoogleResult{}) {
		t.Errorf("result = %+v, want zero", result)
	}
	for _, stored := range h.users.Users {
		if !stored.User.SigninAt.Equal(fixedSeededAt) {
			t.Errorf("SigninAt of %s = %v, want the seeded %v left alone", stored.User.ID, stored.User.SigninAt, fixedSeededAt)
		}
	}
	h.events.AssertNoneDispatched(t)
}

func TestSignInGoogleStampsAndAnnouncesTheGoogleUsersSignIn(t *testing.T) {
	harness := newSignInGoogleForTest(t)

	result, err := harness.signIn(verifiedGoogleIDToken)
	if err != nil {
		t.Fatalf("SignInUserGoogle: %v", err)
	}

	if result.User.ID != harness.userID {
		t.Errorf("ID = %s, want %s", result.User.ID, harness.userID)
	}
	if signedInAt := storedUser(t, harness.users, harness.userID).User.SigninAt; !signedInAt.After(fixedSeededAt) {
		t.Errorf("SigninAt = %v, want it stamped past the seeded %v", signedInAt, fixedSeededAt)
	}
	harness.events.OnlyDispatchedBy(t, eventstest.DispatchedBy{Label: events.UserSignedInLabel, UserID: harness.userID})
}

func TestSignInGoogleMatchesTheGoogleSubjectEvenAfterAnEmailChange(t *testing.T) {
	harness := newSignInGoogleForTest(t)
	stored := storedUser(t, harness.users, harness.userID)
	stored.User.Email = requestedNewEmail
	harness.users.Users[harness.userID] = stored

	result, err := harness.signIn(verifiedGoogleIDToken)
	if err != nil {
		t.Fatalf("SignInUserGoogle: %v", err)
	}

	if result.User.ID != harness.userID || result.User.Email != requestedNewEmail {
		t.Errorf("result.User = %+v, want user %s under its changed email", result.User, harness.userID)
	}
}

func TestSignInGoogleRejectsAccountsItCannotSignIn(t *testing.T) {
	cases := []struct {
		name       string
		idToken    string
		seed       func(t *testing.T, users *repositories.InMemoryUserRepository)
		isRejected func(err error) bool
	}{
		{
			name:       "invalid ID token",
			idToken:    "forged-token",
			seed:       func(*testing.T, *repositories.InMemoryUserRepository) {},
			isRejected: isError[googleidtoken.ErrIDTokenInvalid],
		},
		{
			name:       "email google has not verified",
			idToken:    unverifiedGoogleIDToken,
			seed:       func(*testing.T, *repositories.InMemoryUserRepository) {},
			isRejected: isError[e.ErrGoogleEmailUnverified],
		},
		{
			name:       "google account and email unknown",
			idToken:    verifiedGoogleIDToken,
			seed:       func(*testing.T, *repositories.InMemoryUserRepository) {},
			isRejected: isError[e.ErrGoogleAccountNotFound],
		},
		{
			name:    "email held by a password account",
			idToken: verifiedGoogleIDToken,
			seed: func(t *testing.T, users *repositories.InMemoryUserRepository) {
				seedUser(t, users, seedUserParams{email: testEmail})
			},
			isRejected: isError[e.ErrGoogleAccountNotLinked],
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newSignInGoogleForTest(t)
			harness.users.Users = map[uuid.UUID]repositories.InMemoryUser{}
			testCase.seed(t, harness.users)

			result, err := harness.signIn(testCase.idToken)

			if !testCase.isRejected(err) {
				t.Fatalf("err = %v, want the %s rejection", err, testCase.name)
			}
			harness.assertSignInRejected(t, result)
		})
	}
}

func TestSignInGoogleSurfacesInfrastructureFailures(t *testing.T) {
	cause := errors.New("connection refused")
	cases := []struct {
		name     string
		inject   func(harness *signInGoogleForTest)
		isFailed func(err error) bool
	}{
		{
			name: "google unreachable",
			inject: func(harness *signInGoogleForTest) {
				harness.verifier.VerifyFailure = googleidtoken.ErrIDTokenVerificationFailed{Cause: cause}
			},
			isFailed: isError[googleidtoken.ErrIDTokenVerificationFailed],
		},
		{
			name:     "lookup failure",
			inject:   func(harness *signInGoogleForTest) { harness.users.LookupFailure = e.ErrFindUserFailed{Cause: cause} },
			isFailed: isError[e.ErrFindUserFailed],
		},
		{
			name: "sign-in stamp failure",
			inject: func(harness *signInGoogleForTest) {
				harness.users.MarkSignedInFailure = e.ErrMarkSignedInFailed{Cause: cause}
			},
			isFailed: isError[e.ErrMarkSignedInFailed],
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newSignInGoogleForTest(t)
			testCase.inject(&harness)

			result, err := harness.signIn(verifiedGoogleIDToken)

			if !testCase.isFailed(err) || !errors.Is(err, cause) {
				t.Fatalf("err = %v, want the %s failure wrapping its cause", err, testCase.name)
			}
			harness.assertSignInRejected(t, result)
		})
	}
}
