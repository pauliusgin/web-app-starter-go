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

const replacementPassword = "new-ledger-password-2026"

type changePasswordForTest struct {
	dependencies ChangePasswordDependencies
	store        *inMemoryUserStore
	eventLog     *inMemoryEventLog
	broker       *inMemoryBroker
	userID       uuid.UUID
	storedHash   string
}

func newChangePasswordForTest(t *testing.T) changePasswordForTest {
	t.Helper()

	storedHash, err := password.Hash(testPassword)
	if err != nil {
		t.Fatalf("hashing the seeded password: %v", err)
	}

	store := newInMemoryUserStore()
	userID := uuid.New()
	store.savedUsers = append(store.savedUsers, user.NewUser{
		ID:           userID,
		Email:        testEmail,
		PasswordHash: storedHash,
	})

	eventLog := newInMemoryEventLog()
	broker := newInMemoryBroker()
	recorder := func(ctx context.Context, event events.NewEvent) error {
		_, err := events.Record(ctx, events.RecordDependencies{EventWriteRepository: eventLog}, event)
		return err
	}

	return changePasswordForTest{
		dependencies: ChangePasswordDependencies{
			UserWriteRepository: store,
			EventDispatcher:     events.Dispatcher{Recorder: recorder, Publisher: broker.publish},
		},
		store:      store,
		eventLog:   eventLog,
		broker:     broker,
		userID:     userID,
		storedHash: storedHash,
	}
}

func (h changePasswordForTest) change(params ChangePasswordParams) error {
	return ChangePassword(context.Background(), h.dependencies, params)
}

func (h changePasswordForTest) validParams() ChangePasswordParams {
	return ChangePasswordParams{
		UserID:          h.userID,
		CurrentPassword: testPassword,
		NewPassword:     replacementPassword,
	}
}

func (h changePasswordForTest) currentStoredHash(t *testing.T) string {
	t.Helper()

	credentials, err := h.store.FindUserCredentialsByID(context.Background(), h.userID)
	if err != nil {
		t.Fatalf("finding the user's credentials: %v", err)
	}

	return credentials.PasswordHash
}

func assertPasswordNotChanged(t *testing.T, harness changePasswordForTest) {
	t.Helper()

	if harness.currentStoredHash(t) != harness.storedHash {
		t.Error("stored hash changed, want it left untouched for a rejected password change")
	}
	if len(harness.eventLog.recordedEvents) != 0 || len(harness.broker.publishedEvents) != 0 {
		t.Errorf("recorded/published events = %d/%d, want none for a rejected password change",
			len(harness.eventLog.recordedEvents), len(harness.broker.publishedEvents))
	}
}

func storedHashMatches(t *testing.T, storedHash string, candidatePassword string) bool {
	t.Helper()

	matches, err := password.Verify(password.VerificationParams{Password: candidatePassword, Hash: storedHash})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	return matches
}

func TestChangePasswordStoresAHashOfTheNewPassword(t *testing.T) {
	harness := newChangePasswordForTest(t)

	if err := harness.change(harness.validParams()); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	updatedHash := harness.currentStoredHash(t)
	if !storedHashMatches(t, updatedHash, replacementPassword) {
		t.Error("stored hash does not verify the new password")
	}
	if storedHashMatches(t, updatedHash, testPassword) {
		t.Error("stored hash still verifies the old password")
	}
}

func TestChangePasswordAnnouncesTheChangeWithOnlyTheUserID(t *testing.T) {
	harness := newChangePasswordForTest(t)

	if err := harness.change(harness.validParams()); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	if len(harness.eventLog.recordedEvents) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(harness.eventLog.recordedEvents))
	}
	recorded := harness.eventLog.recordedEvents[0]
	if recorded.Label != events.UserPasswordChangedLabel {
		t.Errorf("Label = %q, want %q", recorded.Label, events.UserPasswordChangedLabel)
	}
	if recorded.UserID != (uuid.NullUUID{UUID: harness.userID, Valid: true}) {
		t.Errorf("UserID = %+v, want %s", recorded.UserID, harness.userID)
	}
	if len(harness.broker.publishedEvents) != 1 {
		t.Errorf("published events = %d, want 1", len(harness.broker.publishedEvents))
	}
}

func TestChangePasswordRejectsAnIncorrectCurrentPassword(t *testing.T) {
	harness := newChangePasswordForTest(t)
	params := harness.validParams()
	params.CurrentPassword = "not-the-current-password"

	err := harness.change(params)

	if !errors.As(err, &usererrors.ErrCurrentPasswordIncorrect{}) {
		t.Fatalf("error = %v, want ErrCurrentPasswordIncorrect", err)
	}
	assertPasswordNotChanged(t, harness)
}

func TestChangePasswordRejectsANewPasswordEqualToTheCurrentOne(t *testing.T) {
	harness := newChangePasswordForTest(t)
	params := harness.validParams()
	params.NewPassword = testPassword

	err := harness.change(params)

	if !errors.As(err, &usererrors.ErrPasswordUnchanged{}) {
		t.Fatalf("error = %v, want ErrPasswordUnchanged", err)
	}
	assertPasswordNotChanged(t, harness)
}

func TestChangePasswordReportsIncorrectBeforeUnchanged(t *testing.T) {
	harness := newChangePasswordForTest(t)
	params := harness.validParams()
	params.CurrentPassword = "wrong-guess-password"
	params.NewPassword = testPassword

	err := harness.change(params)

	if !errors.As(err, &usererrors.ErrCurrentPasswordIncorrect{}) {
		t.Fatalf("error = %v, want ErrCurrentPasswordIncorrect so a guess cannot be probed via unchanged", err)
	}
	assertPasswordNotChanged(t, harness)
}

func TestChangePasswordReportsAnUnknownUser(t *testing.T) {
	harness := newChangePasswordForTest(t)
	params := harness.validParams()
	params.UserID = uuid.New()

	err := harness.change(params)

	if !errors.As(err, &usererrors.ErrUserNotFound{}) {
		t.Fatalf("error = %v, want ErrUserNotFound", err)
	}
	assertPasswordNotChanged(t, harness)
}

func TestChangePasswordReportsAnUnreadableStoredHash(t *testing.T) {
	harness := newChangePasswordForTest(t)
	harness.store.savedUsers[0].PasswordHash = seededHash
	harness.storedHash = seededHash

	err := harness.change(harness.validParams())

	if !errors.As(err, &usererrors.ErrPasswordVerificationFailed{}) {
		t.Fatalf("error = %v, want ErrPasswordVerificationFailed", err)
	}
	assertPasswordNotChanged(t, harness)
}

func TestChangePasswordPropagatesALookupFailure(t *testing.T) {
	harness := newChangePasswordForTest(t)
	lookupFailure := usererrors.ErrIDLookupFailed{Cause: errors.New("pq: connection refused")}
	harness.store.lookupFailure = lookupFailure

	err := harness.change(harness.validParams())

	if err != lookupFailure {
		t.Fatalf("error = %#v, want the repository's own error passed through unchanged", err)
	}
	if len(harness.eventLog.recordedEvents) != 0 || len(harness.broker.publishedEvents) != 0 {
		t.Errorf("recorded/published events = %d/%d, want none when the lookup fails",
			len(harness.eventLog.recordedEvents), len(harness.broker.publishedEvents))
	}
}

func TestChangePasswordPropagatesASaveFailure(t *testing.T) {
	harness := newChangePasswordForTest(t)
	saveFailure := usererrors.ErrChangePasswordFailed{Cause: errors.New("pq: connection refused")}
	harness.store.changePasswordFailure = saveFailure

	err := harness.change(harness.validParams())

	if err != saveFailure {
		t.Fatalf("error = %#v, want the repository's own error passed through unchanged", err)
	}
	assertPasswordNotChanged(t, harness)
}
