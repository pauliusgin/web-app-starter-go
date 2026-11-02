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

type deleteUserForTest struct {
	users  *repositories.InMemoryUserRepository
	events *eventstest.CapturedEvents
	userID uuid.UUID
}

func newDeleteUserForTest(t *testing.T) deleteUserForTest {
	t.Helper()

	users := repositories.NewInMemoryUserRepository()

	return deleteUserForTest{
		users:  users,
		events: &eventstest.CapturedEvents{},
		userID: seedUser(t, users, seedUserParams{email: testEmail}),
	}
}

func (h deleteUserForTest) deleteUser(userID uuid.UUID) error {
	return DeleteUser(context.Background(), DeleteUserDependencies{
		UserWriteRepository: h.users,
		EventDispatcher:     h.events.Dispatcher(),
	}, DeleteUserParams{UserID: userID})
}

func TestDeleteUserAnonymizesTheAccountAndAnnouncesIt(t *testing.T) {
	harness := newDeleteUserForTest(t)

	if err := harness.deleteUser(harness.userID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	want := struct{ email, firstName, lastName string }{"deleted-" + harness.userID.String(), "Deleted", "User"}
	deleted, found := harness.users.SoftDeleted[harness.userID]
	if !found {
		t.Fatal("the user was not soft-deleted")
	}
	if deleted.AnonymizedEmail != want.email ||
		deleted.AnonymizedFirstName != want.firstName ||
		deleted.AnonymizedLastName != want.lastName {
		t.Errorf("anonymized as %q %q %q, want %q %q %q", deleted.AnonymizedEmail,
			deleted.AnonymizedFirstName, deleted.AnonymizedLastName, want.email, want.firstName, want.lastName)
	}
	harness.events.OnlyDispatchedBy(t, eventstest.DispatchedBy{Label: events.UserDeletedLabel, UserID: harness.userID})
}

func TestDeleteUserKeepsTheAccountAndAnnouncesNothingWhenRejected(t *testing.T) {
	cases := []struct {
		name    string
		arrange func(harness deleteUserForTest) uuid.UUID
		wantErr func(err error) bool
	}{
		{
			name:    "unknown account",
			arrange: func(deleteUserForTest) uuid.UUID { return uuid.New() },
			wantErr: isError[e.ErrUserNotFound],
		},
		{
			name: "deletion fails",
			arrange: func(harness deleteUserForTest) uuid.UUID {
				harness.users.SoftDeleteFailure = e.ErrSoftDeleteUserFailed{Cause: errors.New("connection reset")}
				return harness.userID
			},
			wantErr: isError[e.ErrSoftDeleteUserFailed],
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newDeleteUserForTest(t)

			err := harness.deleteUser(testCase.arrange(harness))

			if !testCase.wantErr(err) {
				t.Fatalf("error = %#v, want the case's rejection", err)
			}
			storedUser(t, harness.users, harness.userID)
			harness.events.AssertNoneDispatched(t)
		})
	}
}
