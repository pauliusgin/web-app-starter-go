package usecases

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

var pictureImageID = uuid.MustParse("b2b2b2b2-0000-4000-8000-000000000001")

var bystanderImageID = uuid.MustParse("b2b2b2b2-0000-4000-8000-000000000002")

// deleteUnreferencedUploadFixture holds an attached picture and an attached bystander image, each with its files
// present in storage and neither referenced.
type deleteUnreferencedUploadFixture struct {
	repository  *repositories.InMemoryUploadRepository
	fileStorage *storage.InMemoryFileStorage
	captured    *eventstest.CapturedEvents
}

func newDeleteUnreferencedUploadFixture() deleteUnreferencedUploadFixture {
	attachedAt := time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC)
	picture := storedImage(pictureImageID, attachedAt.Add(-time.Hour))
	picture.AttachedAt = attachedAt
	bystander := storedImage(bystanderImageID, attachedAt.Add(-time.Hour))
	bystander.AttachedAt = attachedAt

	return deleteUnreferencedUploadFixture{
		repository:  repositories.NewInMemoryUploadRepository(picture, bystander),
		fileStorage: fileStorageHolding(picture, bystander),
		captured:    &eventstest.CapturedEvents{},
	}
}

func (f deleteUnreferencedUploadFixture) deleteUnreferenced(uploadID uuid.UUID) error {
	return DeleteUnreferencedUpload(context.Background(), DeleteUnreferencedUploadDependencies{
		UploadWriteRepository: f.repository,
		FileStorage:           f.fileStorage,
		EventDispatcher:       f.captured.Dispatcher(),
	}, DeleteUnreferencedUploadParams{UploadID: uploadID})
}

func (f deleteUnreferencedUploadFixture) remainingUploadIDs() []uuid.UUID {
	return uploadIDsOf(f.repository.Uploads)
}

func TestDeleteUnreferencedUploadRemovesTheUploadAndItsFiles(t *testing.T) {
	fixture := newDeleteUnreferencedUploadFixture()

	if err := fixture.deleteUnreferenced(pictureImageID); err != nil {
		t.Fatalf("DeleteUnreferencedUpload: %v", err)
	}

	if remaining := fixture.remainingUploadIDs(); !slices.Equal(remaining, []uuid.UUID{bystanderImageID}) {
		t.Errorf("remaining uploads = %v, want only the bystander", remaining)
	}
	wantDeletedFiles := []string{pictureImageID.String() + "/original.jpg", pictureImageID.String() + "/preview.webp"}
	if !slices.Equal(fixture.fileStorage.Deleted, wantDeletedFiles) {
		t.Errorf("deleted files = %v, want %v", fixture.fileStorage.Deleted, wantDeletedFiles)
	}
	event := fixture.captured.OnlyDispatched(t, events.UploadDeletedLabel)
	if payload := eventstest.DecodePayload(t, event); payload["uploadId"] != pictureImageID.String() {
		t.Errorf("payload = %v, want the picture's uploadId", payload)
	}
}

func TestDeleteUnreferencedUploadKeepsAnUploadAnotherResourcePointsAt(t *testing.T) {
	fixture := newDeleteUnreferencedUploadFixture()
	fixture.repository.ReferenceUpload(pictureImageID)

	if err := fixture.deleteUnreferenced(pictureImageID); err != nil {
		t.Fatalf("DeleteUnreferencedUpload: %v", err)
	}

	if len(fixture.repository.Uploads) != 2 || len(fixture.fileStorage.Deleted) != 0 {
		t.Errorf("uploads = %d, deleted files = %v, want nothing touched",
			len(fixture.repository.Uploads), fixture.fileStorage.Deleted)
	}
	fixture.captured.AssertNoneDispatched(t)
}

func TestDeleteUnreferencedUploadIgnoresAnUploadAlreadyGone(t *testing.T) {
	fixture := newDeleteUnreferencedUploadFixture()

	if err := fixture.deleteUnreferenced(uuid.New()); err != nil {
		t.Fatalf("DeleteUnreferencedUpload(missing) = %v, want nil", err)
	}

	if len(fixture.repository.Uploads) != 2 || len(fixture.fileStorage.Deleted) != 0 {
		t.Errorf("uploads = %d, deleted files = %v, want nothing touched",
			len(fixture.repository.Uploads), fixture.fileStorage.Deleted)
	}
	fixture.captured.AssertNoneDispatched(t)
}

func TestDeleteUnreferencedUploadTreatsAnAlreadyMissingFileAsDeleted(t *testing.T) {
	fixture := newDeleteUnreferencedUploadFixture()
	delete(fixture.fileStorage.Files, pictureImageID.String()+"/preview.webp")

	if err := fixture.deleteUnreferenced(pictureImageID); err != nil {
		t.Fatalf("DeleteUnreferencedUpload: %v", err)
	}

	if remaining := fixture.remainingUploadIDs(); !slices.Equal(remaining, []uuid.UUID{bystanderImageID}) {
		t.Errorf("remaining uploads = %v, want only the bystander", remaining)
	}
}

func TestDeleteUnreferencedUploadKeepsTheRowWhenItsFilesCannotBeDeleted(t *testing.T) {
	fixture := newDeleteUnreferencedUploadFixture()
	storageFailure := errors.New("storage unreachable")
	fixture.fileStorage.DeleteFailure = storageFailure

	err := fixture.deleteUnreferenced(pictureImageID)

	if _, ok := errors.AsType[e.ErrDeleteStoredFileFailed](err); !ok || !errors.Is(err, storageFailure) {
		t.Fatalf("err = %v, want ErrDeleteStoredFileFailed wrapping the storage failure", err)
	}
	if len(fixture.repository.Uploads) != 2 {
		t.Errorf("uploads left = %d, want the picture kept for the orphan cleanup", len(fixture.repository.Uploads))
	}
	fixture.captured.AssertNoneDispatched(t)
}

func TestDeleteUnreferencedUploadReportsAFailedLookup(t *testing.T) {
	fixture := newDeleteUnreferencedUploadFixture()
	fixture.repository.FindFailure = errDatabaseDown

	err := fixture.deleteUnreferenced(pictureImageID)

	if !errors.Is(err, errDatabaseDown) {
		t.Fatalf("err = %v, want the lookup failure", err)
	}
	if len(fixture.repository.Uploads) != 2 || len(fixture.fileStorage.Deleted) != 0 {
		t.Errorf("uploads = %d, deleted files = %v, want nothing touched",
			len(fixture.repository.Uploads), fixture.fileStorage.Deleted)
	}
}

func TestDeleteUnreferencedUploadReportsARowTheDatabaseCannotDelete(t *testing.T) {
	fixture := newDeleteUnreferencedUploadFixture()
	fixture.repository.DeleteFailure = errDatabaseDown

	err := fixture.deleteUnreferenced(pictureImageID)

	if !errors.Is(err, errDatabaseDown) {
		t.Fatalf("err = %v, want the delete failure", err)
	}
	if len(fixture.repository.Uploads) != 2 {
		t.Errorf("uploads left = %d, want the picture row kept", len(fixture.repository.Uploads))
	}
	fixture.captured.AssertNoneDispatched(t)
}
