package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

type signUpGoogleForTest struct {
	users    *repositories.InMemoryUserRepository
	events   *eventstest.CapturedEvents
	verifier googleidtoken.InMemoryVerifier
}

func newSignUpGoogleForTest() signUpGoogleForTest {
	return signUpGoogleForTest{
		users:    repositories.NewInMemoryUserRepository(),
		events:   &eventstest.CapturedEvents{},
		verifier: newGoogleVerifier(),
	}
}

func (h signUpGoogleForTest) signUp(idToken string) (SignUpUserGoogleResult, error) {
	return SignUpUserGoogle(context.Background(), SignUpUserGoogleDependencies{
		UserWriteRepository: h.users,
		EventDispatcher:     h.events.Dispatcher(),
		VerifyGoogleIDToken: h.verifier.VerifyIDToken,
	}, SignUpUserGoogleParams{IDToken: idToken})
}

func (h signUpGoogleForTest) assertSignUpRejected(t *testing.T, result SignUpUserGoogleResult, usersBefore int) {
	t.Helper()

	if result != (SignUpUserGoogleResult{}) {
		t.Errorf("result = %+v, want zero", result)
	}
	if len(h.users.Users) != usersBefore {
		t.Errorf("stored users = %d, want the %d there before", len(h.users.Users), usersBefore)
	}
	h.events.AssertNoneDispatched(t)
}

func TestSignUpGoogleStoresAVerifiedPasswordlessUserAndAnnouncesIt(t *testing.T) {
	harness := newSignUpGoogleForTest()

	result, err := harness.signUp(verifiedGoogleIDToken)
	if err != nil {
		t.Fatalf("SignUpUserGoogle: %v", err)
	}

	if result.User.Email != testEmail || !result.User.IsEmailVerified() {
		t.Errorf("result.User = %+v, want %q stored verified", result.User, testEmail)
	}
	stored := storedUser(t, harness.users, result.User.ID)
	if stored.GoogleSubject != testGoogleSubject || stored.PasswordHash != "" {
		t.Errorf("stored = %+v, want google subject %q and no password", stored, testGoogleSubject)
	}
	harness.events.OnlyDispatchedBy(t, eventstest.DispatchedBy{Label: events.UserSignedUpLabel, UserID: result.User.ID})
}

func TestSignUpGoogleRejectsAccountsItCannotCreate(t *testing.T) {
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
			name:    "google account already signed up",
			idToken: verifiedGoogleIDToken,
			seed: func(t *testing.T, users *repositories.InMemoryUserRepository) {
				seedGoogleUser(t, users, testGoogleSubject)
			},
			isRejected: isError[e.ErrGoogleAccountTaken],
		},
		{
			name:    "email held by a password account",
			idToken: verifiedGoogleIDToken,
			seed: func(t *testing.T, users *repositories.InMemoryUserRepository) {
				seedUser(t, users, seedUserParams{email: testEmail})
			},
			isRejected: isError[e.ErrEmailTaken],
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newSignUpGoogleForTest()
			testCase.seed(t, harness.users)
			usersBefore := len(harness.users.Users)

			result, err := harness.signUp(testCase.idToken)

			if !testCase.isRejected(err) {
				t.Fatalf("err = %v, want the %s rejection", err, testCase.name)
			}
			harness.assertSignUpRejected(t, result, usersBefore)
		})
	}
}

func TestSignUpGoogleSurfacesInfrastructureFailures(t *testing.T) {
	cause := errors.New("connection refused")
	cases := []struct {
		name     string
		inject   func(harness *signUpGoogleForTest)
		isFailed func(err error) bool
	}{
		{
			name: "google unreachable",
			inject: func(harness *signUpGoogleForTest) {
				harness.verifier.VerifyFailure = googleidtoken.ErrIDTokenVerificationFailed{Cause: cause}
			},
			isFailed: isError[googleidtoken.ErrIDTokenVerificationFailed],
		},
		{
			name:     "lookup failure",
			inject:   func(harness *signUpGoogleForTest) { harness.users.LookupFailure = e.ErrFindUserFailed{Cause: cause} },
			isFailed: isError[e.ErrFindUserFailed],
		},
		{
			name:     "save failure",
			inject:   func(harness *signUpGoogleForTest) { harness.users.SaveFailure = e.ErrSaveUserFailed{Cause: cause} },
			isFailed: isError[e.ErrSaveUserFailed],
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newSignUpGoogleForTest()
			testCase.inject(&harness)

			result, err := harness.signUp(verifiedGoogleIDToken)

			if !testCase.isFailed(err) || !errors.Is(err, cause) {
				t.Fatalf("err = %v, want the %s failure wrapping its cause", err, testCase.name)
			}
			harness.assertSignUpRejected(t, result, 0)
		})
	}
}
