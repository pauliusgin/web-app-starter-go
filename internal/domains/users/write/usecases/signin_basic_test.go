package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	user "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type basicSignInForTest struct {
	dependencies SignInUserDependencies
	store        *inMemoryUserStore
	eventLog     *inMemoryEventLog
	broker       *inMemoryBroker
	userID       uuid.UUID
}

func newBasicSignInForTest(t *testing.T) basicSignInForTest {
	t.Helper()

	hashedPassword, err := password.Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	store := newInMemoryUserStore()
	userID := uuid.New()
	store.savedUsers = append(store.savedUsers, user.NewUser{
		ID:           userID,
		Email:        testEmail,
		PasswordHash: hashedPassword,
	})

	eventLog := newInMemoryEventLog()
	broker := newInMemoryBroker()
	recorder := func(ctx context.Context, event events.NewEvent) error {
		_, err := events.Record(ctx, events.RecordDependencies{EventWriteRepository: eventLog}, event)
		return err
	}

	return basicSignInForTest{
		dependencies: SignInUserDependencies{
			UserWriteRepository: store,
			EventDispatcher:     events.Dispatcher{Recorder: recorder, Publisher: broker.publish},
		},
		store:    store,
		eventLog: eventLog,
		broker:   broker,
		userID:   userID,
	}
}

func assertSignInRejected(t *testing.T, harness basicSignInForTest, signedInUser user.User) {
	t.Helper()

	if signedInUser != (user.User{}) {
		t.Errorf("user = %+v, want the zero value on failure", signedInUser)
	}
	if _, stamped := harness.store.signedInAt[harness.userID]; stamped {
		t.Error("signin_at was stamped for a rejected sign-in")
	}
	if len(harness.eventLog.recordedEvents) != 0 || len(harness.broker.publishedEvents) != 0 {
		t.Errorf("recorded/published events = %d/%d, want no sign-in event for a rejected sign-in",
			len(harness.eventLog.recordedEvents), len(harness.broker.publishedEvents))
	}
}

func TestBasicSignInStampsTheSignInAndReturnsTheUser(t *testing.T) {
	harness := newBasicSignInForTest(t)

	result, err := SignInUser(context.Background(), harness.dependencies, SignInUserParams{
		Email:    testEmail,
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("SignInUser: %v", err)
	}

	if result.User.ID != harness.userID {
		t.Errorf("ID = %s, want %s", result.User.ID, harness.userID)
	}
	if result.User.Email != testEmail {
		t.Errorf("Email = %q, want %q", result.User.Email, testEmail)
	}
	if !result.User.SigninAt.Equal(fixedSignedInAt) {
		t.Errorf("SigninAt = %v, want the stamp the store wrote %v", result.User.SigninAt, fixedSignedInAt)
	}
	if !harness.store.signedInAt[harness.userID].Equal(fixedSignedInAt) {
		t.Error("the store holds no sign-in stamp for the user")
	}
}

func TestBasicSignInAllowsAnUnverifiedEmail(t *testing.T) {
	harness := newBasicSignInForTest(t)

	result, err := SignInUser(context.Background(), harness.dependencies, SignInUserParams{
		Email:    testEmail,
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("SignInUser: %v", err)
	}

	if result.User.IsEmailVerified() {
		t.Fatal("fixture user is verified, want an unverified one for this test")
	}
}

func TestBasicSignInRejectsInvalidCredentialsAlike(t *testing.T) {
	cases := []struct {
		name   string
		params SignInUserParams
	}{
		{name: "wrong password", params: SignInUserParams{Email: testEmail, Password: "wrong horse battery"}},
		{name: "unknown email", params: SignInUserParams{Email: "stranger@example.com", Password: testPassword}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newBasicSignInForTest(t)

			result, err := SignInUser(context.Background(), harness.dependencies, testCase.params)

			if !errors.As(err, &usererrors.ErrInvalidCredentials{}) {
				t.Fatalf("error = %v, want ErrInvalidCredentials", err)
			}
			assertSignInRejected(t, harness, result.User)
		})
	}
}

func TestBasicSignInSurfacesALookupFailureRatherThanInvalidCredentials(t *testing.T) {
	cause := errors.New("connection refused")
	harness := newBasicSignInForTest(t)
	harness.store.lookupFailure = usererrors.ErrEmailLookupFailed{Cause: cause}

	result, err := SignInUser(context.Background(), harness.dependencies, SignInUserParams{
		Email:    testEmail,
		Password: testPassword,
	})

	if !errors.As(err, &usererrors.ErrEmailLookupFailed{}) {
		t.Fatalf("error = %v, want ErrEmailLookupFailed", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the lookup failure does not unwrap to the driver error")
	}
	if errors.As(err, &usererrors.ErrInvalidCredentials{}) {
		t.Error("a lookup failure reports as invalid credentials, want an honest sign-in left unblamed")
	}
	assertSignInRejected(t, harness, result.User)
}

func TestBasicSignInReportsAnUnreadableStoredHash(t *testing.T) {
	harness := newBasicSignInForTest(t)
	harness.store.savedUsers[0].PasswordHash = seededHash

	result, err := SignInUser(context.Background(), harness.dependencies, SignInUserParams{
		Email:    testEmail,
		Password: testPassword,
	})

	if !errors.As(err, &usererrors.ErrPasswordVerificationFailed{}) {
		t.Fatalf("error = %v, want ErrPasswordVerificationFailed", err)
	}
	assertSignInRejected(t, harness, result.User)
}

func TestBasicSignInSurfacesAStampFailureUnchanged(t *testing.T) {
	cause := errors.New("connection reset")
	harness := newBasicSignInForTest(t)
	harness.store.markSignedInFailure = usererrors.ErrMarkSignedInFailed{Cause: cause}

	result, err := SignInUser(context.Background(), harness.dependencies, SignInUserParams{
		Email:    testEmail,
		Password: testPassword,
	})

	if !errors.As(err, &usererrors.ErrMarkSignedInFailed{}) {
		t.Fatalf("error = %v, want ErrMarkSignedInFailed", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the stamp failure does not unwrap to the driver error")
	}
	assertSignInRejected(t, harness, result.User)
}

func TestBasicSignInAnnouncesTheSignIn(t *testing.T) {
	harness := newBasicSignInForTest(t)

	_, err := SignInUser(context.Background(), harness.dependencies, SignInUserParams{
		Email:    testEmail,
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("SignInUser: %v", err)
	}

	if len(harness.eventLog.recordedEvents) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(harness.eventLog.recordedEvents))
	}
	recorded := harness.eventLog.recordedEvents[0]
	if recorded.Label != events.UserSignedInLabel {
		t.Errorf("Label = %q, want %q", recorded.Label, events.UserSignedInLabel)
	}
	if recorded.UserID != (uuid.NullUUID{UUID: harness.userID, Valid: true}) {
		t.Errorf("UserID = %+v, want the signed in account %s", recorded.UserID, harness.userID)
	}
	if len(harness.broker.publishedEvents) != 1 {
		t.Errorf("published events = %d, want 1", len(harness.broker.publishedEvents))
	}
}

func TestBasicSignInSucceedsWhenTheEventCannotBeDispatched(t *testing.T) {
	silenceLogger(t)
	harness := newBasicSignInForTest(t)
	harness.eventLog.failure = errors.New("event log unavailable")
	harness.broker.failure = errors.New("broker unavailable")

	result, err := SignInUser(context.Background(), harness.dependencies, SignInUserParams{
		Email:    testEmail,
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("SignInUser: %v, want event dispatch failures to never fail the sign-in", err)
	}

	if result.User.ID != harness.userID {
		t.Errorf("ID = %s, want %s", result.User.ID, harness.userID)
	}
}
