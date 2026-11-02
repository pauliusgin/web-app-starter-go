package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

const replacementPassword = "new-ledger-password-2026"

type changePasswordForTest struct {
	users      *repositories.InMemoryUserRepository
	events     *eventstest.CapturedEvents
	userID     uuid.UUID
	seededHash string
}

func newChangePasswordForTest(t *testing.T) changePasswordForTest {
	t.Helper()

	users := repositories.NewInMemoryUserRepository()
	userID := seedUser(t, users, seedUserParams{email: testEmail, password: testPassword})

	return changePasswordForTest{
		users:      users,
		events:     &eventstest.CapturedEvents{},
		userID:     userID,
		seededHash: storedUser(t, users, userID).PasswordHash,
	}
}

func (h changePasswordForTest) change(params ChangePasswordParams) error {
	return ChangePassword(context.Background(), ChangePasswordDependencies{
		UserWriteRepository: h.users,
		EventDispatcher:     h.events.Dispatcher(),
	}, params)
}

func (h changePasswordForTest) validParams() ChangePasswordParams {
	return ChangePasswordParams{UserID: h.userID, CurrentPassword: testPassword, NewPassword: replacementPassword}
}

func (h changePasswordForTest) assertPasswordNotChanged(t *testing.T) {
	t.Helper()

	if storedUser(t, h.users, h.userID).PasswordHash != h.seededHash {
		t.Error("stored hash changed, want it left untouched for a rejected password change")
	}
	h.events.AssertNoneDispatched(t)
}

func hashVerifies(t *testing.T, storedHash string, candidatePassword string) bool {
	t.Helper()

	matches, err := password.Verify(password.VerificationParams{Password: candidatePassword, Hash: storedHash})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	return matches
}

func TestChangePasswordStoresAHashOfTheNewPasswordAndAnnouncesIt(t *testing.T) {
	harness := newChangePasswordForTest(t)

	if err := harness.change(harness.validParams()); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	updatedHash := storedUser(t, harness.users, harness.userID).PasswordHash
	if !hashVerifies(t, updatedHash, replacementPassword) || hashVerifies(t, updatedHash, testPassword) {
		t.Error("stored hash does not verify only the new password")
	}
	harness.events.OnlyDispatchedBy(t, eventstest.DispatchedBy{Label: events.UserPasswordChangedLabel, UserID: harness.userID})
}

func TestChangePasswordRejectsWithoutChangingAnything(t *testing.T) {
	cases := []struct {
		name    string
		change  func(params *ChangePasswordParams)
		wantErr func(err error) bool
	}{
		{
			name:    "incorrect current password",
			change:  func(params *ChangePasswordParams) { params.CurrentPassword = "not-the-current-password" },
			wantErr: isError[e.ErrCurrentPasswordIncorrect],
		},
		{
			name:    "new password equal to the current one",
			change:  func(params *ChangePasswordParams) { params.NewPassword = testPassword },
			wantErr: isError[e.ErrPasswordUnchanged],
		},
		{
			// Checking the guess first stops a caller probing a password through ErrPasswordUnchanged.
			name: "incorrect guess equal to the new password",
			change: func(params *ChangePasswordParams) {
				params.CurrentPassword = "wrong-guess-password"
				params.NewPassword = "wrong-guess-password"
			},
			wantErr: isError[e.ErrCurrentPasswordIncorrect],
		},
		{
			name:    "unknown user",
			change:  func(params *ChangePasswordParams) { params.UserID = uuid.New() },
			wantErr: isError[e.ErrUserNotFound],
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newChangePasswordForTest(t)
			params := harness.validParams()
			testCase.change(&params)

			err := harness.change(params)

			if !testCase.wantErr(err) {
				t.Fatalf("error = %#v, want the case's rejection", err)
			}
			harness.assertPasswordNotChanged(t)
		})
	}
}

func TestChangePasswordReportsAnUnreadableStoredHash(t *testing.T) {
	harness := newChangePasswordForTest(t)
	stored := storedUser(t, harness.users, harness.userID)
	stored.PasswordHash = unreadableHash
	harness.users.Users[harness.userID] = stored
	harness.seededHash = unreadableHash

	err := harness.change(harness.validParams())

	if !errors.As(err, &e.ErrPasswordVerificationFailed{}) {
		t.Fatalf("error = %v, want ErrPasswordVerificationFailed", err)
	}
	harness.assertPasswordNotChanged(t)
}

func TestChangePasswordAnnouncesNothingWhenTheNewHashIsNotStored(t *testing.T) {
	harness := newChangePasswordForTest(t)
	harness.users.ChangePasswordFailure = e.ErrChangePasswordFailed{Cause: errors.New("connection refused")}

	err := harness.change(harness.validParams())

	if !errors.As(err, &e.ErrChangePasswordFailed{}) {
		t.Fatalf("error = %v, want ErrChangePasswordFailed", err)
	}
	harness.assertPasswordNotChanged(t)
}

func TestChangePasswordRejectsAGoogleOnlyAccount(t *testing.T) {
	harness := newChangePasswordForTest(t)
	harness.users.Users = map[uuid.UUID]repositories.InMemoryUser{}
	harness.userID = seedGoogleUser(t, harness.users, testGoogleSubject)
	harness.seededHash = ""

	err := harness.change(harness.validParams())

	if !errors.As(err, &e.ErrPasswordNotSet{}) {
		t.Fatalf("error = %v, want ErrPasswordNotSet", err)
	}
	harness.assertPasswordNotChanged(t)
}
