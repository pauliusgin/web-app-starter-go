package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

type addGoogleAccountForTest struct {
	users  *repositories.InMemoryUserRepository
	events *eventstest.CapturedEvents
	userID uuid.UUID
}

func newAddGoogleAccountForTest(t *testing.T) addGoogleAccountForTest {
	t.Helper()

	users := repositories.NewInMemoryUserRepository()

	return addGoogleAccountForTest{
		users:  users,
		events: &eventstest.CapturedEvents{},
		userID: seedUser(t, users, seedUserParams{email: testEmail}),
	}
}

func (h addGoogleAccountForTest) add(idToken string) error {
	return AddGoogleAccount(context.Background(), AddGoogleAccountDependencies{
		UserWriteRepository: h.users,
		EventDispatcher:     h.events.Dispatcher(),
		VerifyGoogleIDToken: newGoogleVerifier().VerifyIDToken,
	}, AddGoogleAccountParams{UserID: h.userID, IDToken: idToken})
}

func (h addGoogleAccountForTest) assertNothingAdded(t *testing.T, wantSubject string) {
	t.Helper()

	if stored, found := h.users.Users[h.userID]; found && stored.GoogleSubject != wantSubject {
		t.Errorf("stored google subject = %q, want %q left untouched for a rejected addition", stored.GoogleSubject, wantSubject)
	}
	h.events.AssertNoneDispatched(t)
}

func TestAddGoogleAccountStoresTheSubjectVerifiesTheEmailAndAnnouncesIt(t *testing.T) {
	harness := newAddGoogleAccountForTest(t)

	if err := harness.add(verifiedGoogleIDToken); err != nil {
		t.Fatalf("AddGoogleAccount: %v", err)
	}

	stored := storedUser(t, harness.users, harness.userID)
	if stored.GoogleSubject != testGoogleSubject {
		t.Errorf("GoogleSubject = %q, want %q", stored.GoogleSubject, testGoogleSubject)
	}
	if !stored.User.IsEmailVerified() {
		t.Error("email is unverified, want it stamped verified since google vouched for it")
	}
	if stored.PasswordHash != unreadableHash {
		t.Errorf("PasswordHash = %q, want %q kept so password sign-in still works", stored.PasswordHash, unreadableHash)
	}
	harness.events.OnlyDispatchedBy(t, eventstest.DispatchedBy{Label: events.UserGoogleAccountAddedLabel, UserID: harness.userID})
}

func TestAddGoogleAccountKeepsAnEarlierEmailVerification(t *testing.T) {
	harness := newAddGoogleAccountForTest(t)
	harness.userID = seedUser(t, harness.users, seedUserParams{email: testEmail, verified: true})

	if err := harness.add(verifiedGoogleIDToken); err != nil {
		t.Fatalf("AddGoogleAccount: %v", err)
	}

	stored := storedUser(t, harness.users, harness.userID)
	if !stored.User.EmailVerifiedAt.Equal(fixedSeededAt) {
		t.Errorf("EmailVerifiedAt = %v, want the earlier stamp %v kept", stored.User.EmailVerifiedAt, fixedSeededAt)
	}
}

func TestAddGoogleAccountRejectsAnAccountThatAlreadyHasOne(t *testing.T) {
	harness := newAddGoogleAccountForTest(t)
	harness.userID = seedGoogleUser(t, harness.users, "earlier-google-subject")

	err := harness.add(verifiedGoogleIDToken)

	if !errors.As(err, &e.ErrGoogleAccountAlreadyConnected{}) {
		t.Fatalf("error = %v, want ErrGoogleAccountAlreadyConnected", err)
	}
	harness.assertNothingAdded(t, "earlier-google-subject")
}

func TestAddGoogleAccountRejectsAnUnknownUser(t *testing.T) {
	harness := newAddGoogleAccountForTest(t)
	harness.userID = uuid.New()

	err := harness.add(verifiedGoogleIDToken)

	if !errors.As(err, &e.ErrUserNotFound{}) {
		t.Fatalf("error = %v, want ErrUserNotFound", err)
	}
	harness.assertNothingAdded(t, "")
}

func TestAddGoogleAccountRejectsAnInvalidIDToken(t *testing.T) {
	harness := newAddGoogleAccountForTest(t)

	err := harness.add("forged-google-id-token")

	if !errors.As(err, &googleidtoken.ErrIDTokenInvalid{}) {
		t.Fatalf("error = %v, want ErrIDTokenInvalid", err)
	}
	harness.assertNothingAdded(t, "")
}

func TestAddGoogleAccountRejectsAnUnverifiedGoogleEmail(t *testing.T) {
	harness := newAddGoogleAccountForTest(t)

	err := harness.add(unverifiedGoogleIDToken)

	if !errors.As(err, &e.ErrGoogleEmailUnverified{}) {
		t.Fatalf("error = %v, want ErrGoogleEmailUnverified", err)
	}
	harness.assertNothingAdded(t, "")
}

func TestAddGoogleAccountRejectsAGoogleEmailOtherThanTheAccounts(t *testing.T) {
	harness := newAddGoogleAccountForTest(t)
	harness.userID = seedUser(t, harness.users, seedUserParams{email: requestedNewEmail})

	err := harness.add(verifiedGoogleIDToken)

	var mismatch e.ErrGoogleEmailMismatch
	if !errors.As(err, &mismatch) {
		t.Fatalf("error = %v, want ErrGoogleEmailMismatch", err)
	}
	if mismatch.GoogleEmail != testEmail {
		t.Errorf("GoogleEmail = %q, want the normalized %q", mismatch.GoogleEmail, testEmail)
	}
	harness.assertNothingAdded(t, "")
}

func TestAddGoogleAccountRejectsAGoogleAccountAnotherUserHolds(t *testing.T) {
	harness := newAddGoogleAccountForTest(t)
	holderID := uuid.New()
	harness.users.Users[holderID] = repositories.InMemoryUser{
		User:          models.User{ID: holderID, Email: requestedNewEmail, CreatedAt: fixedSeededAt},
		GoogleSubject: testGoogleSubject,
	}

	err := harness.add(verifiedGoogleIDToken)

	if !errors.As(err, &e.ErrGoogleAccountTaken{}) {
		t.Fatalf("error = %v, want ErrGoogleAccountTaken", err)
	}
	harness.assertNothingAdded(t, "")
}

func TestAddGoogleAccountAnnouncesNothingWhenTheSubjectIsNotStored(t *testing.T) {
	harness := newAddGoogleAccountForTest(t)
	harness.users.AddGoogleFailure = e.ErrAddGoogleAccountFailed{Cause: errors.New("connection refused")}

	err := harness.add(verifiedGoogleIDToken)

	if !errors.As(err, &e.ErrAddGoogleAccountFailed{}) {
		t.Fatalf("error = %v, want ErrAddGoogleAccountFailed", err)
	}
	harness.assertNothingAdded(t, "")
}
