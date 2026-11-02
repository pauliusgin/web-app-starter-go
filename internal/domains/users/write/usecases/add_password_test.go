package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

const addedPassword = "added-ledger-password-2026"

type addPasswordForTest struct {
	users  *repositories.InMemoryUserRepository
	events *eventstest.CapturedEvents
	userID uuid.UUID
}

func newAddPasswordForTest(t *testing.T) addPasswordForTest {
	t.Helper()

	users := repositories.NewInMemoryUserRepository()

	return addPasswordForTest{
		users:  users,
		events: &eventstest.CapturedEvents{},
		userID: seedGoogleUser(t, users, testGoogleSubject),
	}
}

func (h addPasswordForTest) add(params AddPasswordParams) error {
	return AddPassword(context.Background(), AddPasswordDependencies{
		UserWriteRepository: h.users,
		EventDispatcher:     h.events.Dispatcher(),
	}, params)
}

func (h addPasswordForTest) validParams() AddPasswordParams {
	return AddPasswordParams{UserID: h.userID, Password: addedPassword}
}

func (h addPasswordForTest) assertNothingAdded(t *testing.T, wantHash string) {
	t.Helper()

	if stored, found := h.users.Users[h.userID]; found && stored.PasswordHash != wantHash {
		t.Errorf("stored hash = %q, want %q left untouched for a rejected password addition", stored.PasswordHash, wantHash)
	}
	h.events.AssertNoneDispatched(t)
}

func TestAddPasswordStoresAHashBesideTheGoogleAccountAndAnnouncesIt(t *testing.T) {
	harness := newAddPasswordForTest(t)

	if err := harness.add(harness.validParams()); err != nil {
		t.Fatalf("AddPassword: %v", err)
	}

	stored := storedUser(t, harness.users, harness.userID)
	if !hashVerifies(t, stored.PasswordHash, addedPassword) {
		t.Error("stored hash does not verify the added password")
	}
	if stored.GoogleSubject != testGoogleSubject {
		t.Errorf("GoogleSubject = %q, want %q kept so google sign-in still works", stored.GoogleSubject, testGoogleSubject)
	}
	harness.events.OnlyDispatchedBy(t, eventstest.DispatchedBy{Label: events.UserPasswordAddedLabel, UserID: harness.userID})
}

func TestAddPasswordRejectsAnAccountThatAlreadyHasAPassword(t *testing.T) {
	harness := newAddPasswordForTest(t)
	harness.userID = seedUser(t, harness.users, seedUserParams{email: requestedNewEmail})

	err := harness.add(harness.validParams())

	if !errors.As(err, &e.ErrPasswordAlreadySet{}) {
		t.Fatalf("error = %v, want ErrPasswordAlreadySet", err)
	}
	harness.assertNothingAdded(t, unreadableHash)
}

func TestAddPasswordRejectsAnAccountWithoutAGoogleAccount(t *testing.T) {
	harness := newAddPasswordForTest(t)
	harness.userID = uuid.New()
	harness.users.Users[harness.userID] = repositories.InMemoryUser{
		User: models.User{ID: harness.userID, Email: requestedNewEmail, CreatedAt: fixedSeededAt},
	}

	err := harness.add(harness.validParams())

	if !errors.As(err, &e.ErrGoogleAccountNotConnected{}) {
		t.Fatalf("error = %v, want ErrGoogleAccountNotConnected", err)
	}
	harness.assertNothingAdded(t, "")
}

func TestAddPasswordRejectsAnUnknownUser(t *testing.T) {
	harness := newAddPasswordForTest(t)
	harness.userID = uuid.New()

	err := harness.add(harness.validParams())

	if !errors.As(err, &e.ErrUserNotFound{}) {
		t.Fatalf("error = %v, want ErrUserNotFound", err)
	}
	harness.assertNothingAdded(t, "")
}

func TestAddPasswordAnnouncesNothingWhenTheHashIsNotStored(t *testing.T) {
	harness := newAddPasswordForTest(t)
	harness.users.AddPasswordFailure = e.ErrAddPasswordFailed{Cause: errors.New("connection refused")}

	err := harness.add(harness.validParams())

	if !errors.As(err, &e.ErrAddPasswordFailed{}) {
		t.Fatalf("error = %v, want ErrAddPasswordFailed", err)
	}
	harness.assertNothingAdded(t, "")
}
