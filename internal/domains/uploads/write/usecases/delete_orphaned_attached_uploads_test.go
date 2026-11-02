package usecases

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

var oldOrphanImageID = uuid.MustParse("c3c3c3c3-0000-4000-8000-000000000001")

var olderOrphanDocumentID = uuid.MustParse("c3c3c3c3-0000-4000-8000-000000000002")

var freshOrphanImageID = uuid.MustParse("c3c3c3c3-0000-4000-8000-000000000003")

var referencedImageID = uuid.MustParse("c3c3c3c3-0000-4000-8000-000000000004")

var stalePendingImageID = uuid.MustParse("c3c3c3c3-0000-4000-8000-000000000005")

func attachedUpload(upload models.Upload, attachedAt time.Time) models.Upload {
	upload.AttachedAt = attachedAt
	return upload
}

// deleteOrphanedAttachedUploadsFixture holds an old orphaned image, an older orphaned document, a freshly
// attached unreferenced image, an old referenced image and a stale pending image, all with files in storage.
type deleteOrphanedAttachedUploadsFixture struct {
	repository  *repositories.InMemoryUploadRepository
	fileStorage *storage.InMemoryFileStorage
	captured    *eventstest.CapturedEvents
}

func newDeleteOrphanedAttachedUploadsFixture() deleteOrphanedAttachedUploadsFixture {
	createdAt := cleanupCutoff.Add(-96 * time.Hour)
	uploads := []models.Upload{
		attachedUpload(storedImage(oldOrphanImageID, createdAt), cleanupCutoff.Add(-time.Hour)),
		attachedUpload(storedDocument(olderOrphanDocumentID, createdAt), cleanupCutoff.Add(-2*time.Hour)),
		attachedUpload(storedImage(freshOrphanImageID, createdAt), cleanupCutoff.Add(time.Hour)),
		attachedUpload(storedImage(referencedImageID, createdAt), cleanupCutoff.Add(-72*time.Hour)),
		storedImage(stalePendingImageID, createdAt),
	}

	repository := repositories.NewInMemoryUploadRepository(uploads...)
	repository.ReferenceUpload(referencedImageID)

	return deleteOrphanedAttachedUploadsFixture{
		repository:  repository,
		fileStorage: fileStorageHolding(uploads...),
		captured:    &eventstest.CapturedEvents{},
	}
}

func (f deleteOrphanedAttachedUploadsFixture) deleteOrphaned() (DeleteOrphanedAttachedUploadsResult, error) {
	return DeleteOrphanedAttachedUploads(context.Background(), DeleteOrphanedAttachedUploadsDependencies{
		UploadWriteRepository: f.repository,
		FileStorage:           f.fileStorage,
		EventDispatcher:       f.captured.Dispatcher(),
	})
}

func (f deleteOrphanedAttachedUploadsFixture) remainingUploadIDs() []uuid.UUID {
	return uploadIDsOf(f.repository.Uploads)
}

func TestDeleteOrphanedAttachedUploadsRemovesOnlyUnreferencedUploadsAttachedBeforeTheCutoff(t *testing.T) {
	fixture := newDeleteOrphanedAttachedUploadsFixture()

	result, err := fixture.deleteOrphaned()
	if err != nil {
		t.Fatalf("DeleteOrphanedAttachedUploads: %v", err)
	}

	if result.DeletedCount != 2 {
		t.Errorf("deleted = %d, want the two old orphans", result.DeletedCount)
	}
	wantRemaining := []uuid.UUID{freshOrphanImageID, referencedImageID, stalePendingImageID}
	if remaining := fixture.remainingUploadIDs(); !slices.Equal(remaining, wantRemaining) {
		t.Errorf("remaining uploads = %v, want %v", remaining, wantRemaining)
	}

	wantDeletedFiles := []string{
		olderOrphanDocumentID.String() + "/original.pdf",
		oldOrphanImageID.String() + "/original.jpg",
		oldOrphanImageID.String() + "/preview.webp",
	}
	if !slices.Equal(fixture.fileStorage.Deleted, wantDeletedFiles) {
		t.Errorf("deleted files = %v, want %v, oldest attachment first", fixture.fileStorage.Deleted, wantDeletedFiles)
	}
}

func TestDeleteOrphanedAttachedUploadsAnnouncesEachRemovedUpload(t *testing.T) {
	fixture := newDeleteOrphanedAttachedUploadsFixture()

	if _, err := fixture.deleteOrphaned(); err != nil {
		t.Fatalf("DeleteOrphanedAttachedUploads: %v", err)
	}

	if len(fixture.captured.Recorded) != 2 || len(fixture.captured.Published) != 2 {
		t.Fatalf("recorded = %d, published = %d, want 2 each",
			len(fixture.captured.Recorded), len(fixture.captured.Published))
	}
	for index, event := range fixture.captured.Published {
		if event.Label != events.UploadDeletedLabel {
			t.Errorf("event %d label = %q, want %q", index, event.Label, events.UploadDeletedLabel)
		}
	}
}

func TestDeleteOrphanedAttachedUploadsKeepsRowsWhoseFilesCannotBeDeleted(t *testing.T) {
	fixture := newDeleteOrphanedAttachedUploadsFixture()
	storageFailure := errors.New("storage unreachable")
	fixture.fileStorage.DeleteFailure = storageFailure

	result, err := fixture.deleteOrphaned()

	if !errors.Is(err, storageFailure) {
		t.Fatalf("err = %v, want the storage failure", err)
	}
	if result.DeletedCount != 0 || len(fixture.repository.Uploads) != 5 {
		t.Errorf("deleted = %d, uploads left = %d, want 0 and all 5 kept for the next run",
			result.DeletedCount, len(fixture.repository.Uploads))
	}
	fixture.captured.AssertNoneDispatched(t)
}

func TestDeleteOrphanedAttachedUploadsReportsAFailedLookup(t *testing.T) {
	fixture := newDeleteOrphanedAttachedUploadsFixture()
	fixture.repository.ListFailure = errDatabaseDown

	result, err := fixture.deleteOrphaned()

	if !errors.Is(err, errDatabaseDown) || result.DeletedCount != 0 {
		t.Fatalf("deleted = %d, err = %v, want 0 and the lookup failure", result.DeletedCount, err)
	}
	if len(fixture.fileStorage.Deleted) != 0 || len(fixture.repository.Uploads) != 5 {
		t.Errorf("deleted files = %v, uploads = %d, want nothing touched",
			fixture.fileStorage.Deleted, len(fixture.repository.Uploads))
	}
}
