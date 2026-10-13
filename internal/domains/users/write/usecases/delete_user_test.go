package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	user "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type deleteUserForTest struct {
	dependencies DeleteUserDependencies
	store        *inMemoryUserStore
	eventLog     *inMemoryEventLog
	broker       *inMemoryBroker
	userID       uuid.UUID
}

func newDeleteUserForTest() deleteUserForTest {
	store := newInMemoryUserStore()
	userID := uuid.New()
	store.savedUsers = append(store.savedUsers, user.NewUser{
		ID:           userID,
		Email:        testEmail,
		PasswordHash: seededHash,
	})

	eventLog := newInMemoryEventLog()
	broker := newInMemoryBroker()
	recorder := func(ctx context.Context, event events.NewEvent) error {
		_, err := events.Record(ctx, events.RecordDependencies{EventWriteRepository: eventLog}, event)
		return err
	}

	return deleteUserForTest{
		dependencies: DeleteUserDependencies{
			UserWriteRepository: store,
			EventDispatcher:     events.Dispatcher{Recorder: recorder, Publisher: broker.publish},
		},
		store:    store,
		eventLog: eventLog,
		broker:   broker,
		userID:   userID,
	}
}

func assertDeletionRejected(t *testing.T, harness deleteUserForTest) {
	t.Helper()

	if len(harness.store.softDeletedUsers) != 0 {
		t.Errorf("soft-deleted users = %+v, want none for a rejected deletion", harness.store.softDeletedUsers)
	}
	if _, err := harness.store.FindUserByID(context.Background(), harness.userID); err != nil {
		t.Errorf("FindUserByID: %v, want the account still live", err)
	}
	if len(harness.eventLog.recordedEvents) != 0 || len(harness.broker.publishedEvents) != 0 {
		t.Errorf("recorded/published events = %d/%d, want no deletion event for a rejected deletion",
			len(harness.eventLog.recordedEvents), len(harness.broker.publishedEvents))
	}
}

func TestDeleteUserSoftDeletesAndAnonymizesTheOwnAccount(t *testing.T) {
	harness := newDeleteUserForTest()

	err := DeleteUser(context.Background(), harness.dependencies, DeleteUserParams{
		ActingUserID: harness.userID,
		TargetUserID: harness.userID,
	})
	if err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	if len(harness.store.softDeletedUsers) != 1 {
		t.Fatalf("soft-deleted users = %d, want 1", len(harness.store.softDeletedUsers))
	}
	deletedUser := harness.store.softDeletedUsers[0]
	if wantEmail := "deleted-" + harness.userID.String(); deletedUser.Email != wantEmail {
		t.Errorf("Email = %q, want %q", deletedUser.Email, wantEmail)
	}
	softDeleteRequest := harness.store.softDeleteRequests[0]
	if softDeleteRequest.AnonymizedFirstName != "Deleted" || softDeleteRequest.AnonymizedLastName != "User" {
		t.Errorf("profile placeholders = %q %q, want %q %q",
			softDeleteRequest.AnonymizedFirstName, softDeleteRequest.AnonymizedLastName, "Deleted", "User")
	}
	if _, err := harness.store.FindUserByEmail(context.Background(), testEmail); !errors.As(err, &usererrors.ErrUserNotFound{}) {
		t.Errorf("FindUserByEmail(original) err = %v, want ErrUserNotFound once the email is freed", err)
	}

	if len(harness.eventLog.recordedEvents) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(harness.eventLog.recordedEvents))
	}
	recorded := harness.eventLog.recordedEvents[0]
	if recorded.Label != events.UserDeletedLabel {
		t.Errorf("Label = %q, want %q", recorded.Label, events.UserDeletedLabel)
	}
	if recorded.UserID != (uuid.NullUUID{UUID: harness.userID, Valid: true}) {
		t.Errorf("UserID = %+v, want %s", recorded.UserID, harness.userID)
	}
	if len(harness.broker.publishedEvents) != 1 {
		t.Errorf("published events = %d, want 1", len(harness.broker.publishedEvents))
	}
}

func TestDeleteUserRejectsDeletingAnotherAccount(t *testing.T) {
	harness := newDeleteUserForTest()
	otherUserID := uuid.New()

	err := DeleteUser(context.Background(), harness.dependencies, DeleteUserParams{
		ActingUserID: otherUserID,
		TargetUserID: harness.userID,
	})

	forbidden := usererrors.ErrUserDeletionForbidden{}
	if !errors.As(err, &forbidden) {
		t.Fatalf("err = %v, want ErrUserDeletionForbidden", err)
	}
	if forbidden.ActingUserID != otherUserID || forbidden.TargetUserID != harness.userID {
		t.Errorf("forbidden = %+v, want acting %s and target %s", forbidden, otherUserID, harness.userID)
	}
	assertDeletionRejected(t, harness)
}

func TestDeleteUserReportsAnUnknownAccount(t *testing.T) {
	harness := newDeleteUserForTest()
	unknownUserID := uuid.New()

	err := DeleteUser(context.Background(), harness.dependencies, DeleteUserParams{
		ActingUserID: unknownUserID,
		TargetUserID: unknownUserID,
	})

	if !errors.As(err, &usererrors.ErrUserNotFound{}) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
	assertDeletionRejected(t, harness)
}

func TestDeleteUserPropagatesARepositoryFailure(t *testing.T) {
	harness := newDeleteUserForTest()
	storeFailure := usererrors.ErrSoftDeleteUserFailed{Cause: errors.New("connection reset")}
	harness.store.softDeleteFailure = storeFailure

	err := DeleteUser(context.Background(), harness.dependencies, DeleteUserParams{
		ActingUserID: harness.userID,
		TargetUserID: harness.userID,
	})

	if !errors.As(err, &usererrors.ErrSoftDeleteUserFailed{}) {
		t.Fatalf("err = %v, want ErrSoftDeleteUserFailed", err)
	}
	assertDeletionRejected(t, harness)
}

func TestDeleteUserSucceedsWhenTheEventCannotBeRecorded(t *testing.T) {
	harness := newDeleteUserForTest()
	harness.eventLog.failure = errors.New("event log unavailable")

	err := DeleteUser(context.Background(), harness.dependencies, DeleteUserParams{
		ActingUserID: harness.userID,
		TargetUserID: harness.userID,
	})
	if err != nil {
		t.Fatalf("DeleteUser: %v, want success despite the dispatch failure", err)
	}

	if len(harness.store.softDeletedUsers) != 1 {
		t.Errorf("soft-deleted users = %d, want 1", len(harness.store.softDeletedUsers))
	}
}

func TestCanDeleteUserAllowsOnlyTheOwnAccount(t *testing.T) {
	ownUserID := uuid.New()

	cases := []struct {
		name   string
		params DeleteUserParams
		want   bool
	}{
		{name: "own account", params: DeleteUserParams{ActingUserID: ownUserID, TargetUserID: ownUserID}, want: true},
		{name: "another account", params: DeleteUserParams{ActingUserID: ownUserID, TargetUserID: uuid.New()}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := CanDeleteUser(testCase.params); got != testCase.want {
				t.Errorf("CanDeleteUser(%+v) = %v, want %v", testCase.params, got, testCase.want)
			}
		})
	}
}
