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
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

// cleanupCutoff mirrors the cutoff the use case derives from the wall clock; fixtures sit hours away
// from it, so the few milliseconds between the two clock reads cannot flip a verdict.
var cleanupCutoff = time.Now().Add(-models.PendingUploadRetention)

var staleImageID = uuid.MustParse("a1a1a1a1-0000-4000-8000-000000000001")

var staleDocumentID = uuid.MustParse("a1a1a1a1-0000-4000-8000-000000000002")

var freshImageID = uuid.MustParse("a1a1a1a1-0000-4000-8000-000000000003")

var attachedImageID = uuid.MustParse("a1a1a1a1-0000-4000-8000-000000000004")

func storedImage(id uuid.UUID, createdAt time.Time) models.Upload {
	return models.Upload{
		ID:           id,
		UserID:       testUserID,
		Purpose:      models.PurposeProfilePicture,
		Kind:         models.KindImage,
		OriginalPath: id.String() + "/original.jpg",
		PreviewPath:  id.String() + "/preview.webp",
		CreatedAt:    createdAt,
	}
}

func storedDocument(id uuid.UUID, createdAt time.Time) models.Upload {
	return models.Upload{
		ID:           id,
		UserID:       testUserID,
		Purpose:      models.PurposeDocumentAttachment,
		Kind:         models.KindDocument,
		OriginalPath: id.String() + "/original.pdf",
		CreatedAt:    createdAt,
	}
}

// deleteStalePendingUploadsFixture holds a stale image, a stale document, a fresh image and an
// attached stale image, each with its files present in storage.
type deleteStalePendingUploadsFixture struct {
	repository  *repositories.InMemoryUploadRepository
	fileStorage *storage.InMemoryFileStorage
	captured    *capturedEvents
}

func newDeleteStalePendingUploadsFixture() deleteStalePendingUploadsFixture {
	attached := storedImage(attachedImageID, cleanupCutoff.Add(-72*time.Hour))
	attached.AttachedAt = cleanupCutoff.Add(-71 * time.Hour)

	uploads := []models.Upload{
		storedImage(staleImageID, cleanupCutoff.Add(-2*time.Hour)),
		storedDocument(staleDocumentID, cleanupCutoff.Add(-1*time.Hour)),
		storedImage(freshImageID, cleanupCutoff.Add(time.Hour)),
		attached,
	}

	fileStorage := storage.NewInMemoryFileStorage()
	for _, upload := range uploads {
		for _, path := range upload.StoredPaths() {
			fileStorage.Files[path] = storage.StoreFileParams{Path: path}
		}
	}

	return deleteStalePendingUploadsFixture{
		repository:  repositories.NewInMemoryUploadRepository(uploads...),
		fileStorage: fileStorage,
		captured:    &capturedEvents{},
	}
}

func (f deleteStalePendingUploadsFixture) deleteStale() (DeleteStalePendingUploadsResult, error) {
	return DeleteStalePendingUploads(context.Background(), DeleteStalePendingUploadsDependencies{
		UploadRepository: f.repository,
		FileStorage:      f.fileStorage,
		EventDispatcher:  f.captured.dispatcher(),
	})
}

func (f deleteStalePendingUploadsFixture) remainingUploadIDs() []uuid.UUID {
	var ids []uuid.UUID
	for _, upload := range f.repository.Uploads {
		ids = append(ids, upload.ID)
	}

	return ids
}

func TestDeleteStalePendingUploadsRemovesOnlyUnattachedUploadsOlderThanTheCutoff(t *testing.T) {
	fixture := newDeleteStalePendingUploadsFixture()

	result, err := fixture.deleteStale()
	if err != nil {
		t.Fatalf("DeleteStalePendingUploads: %v", err)
	}

	if result.DeletedCount != 2 {
		t.Errorf("deleted = %d, want the stale image and the stale document", result.DeletedCount)
	}
	if remaining := fixture.remainingUploadIDs(); !slices.Equal(remaining, []uuid.UUID{freshImageID, attachedImageID}) {
		t.Errorf("remaining uploads = %v, want the fresh and the attached one", remaining)
	}

	wantDeletedFiles := []string{
		staleImageID.String() + "/original.jpg",
		staleImageID.String() + "/preview.webp",
		staleDocumentID.String() + "/original.pdf",
	}
	if !slices.Equal(fixture.fileStorage.Deleted, wantDeletedFiles) {
		t.Errorf("deleted files = %v, want %v", fixture.fileStorage.Deleted, wantDeletedFiles)
	}
	if len(fixture.fileStorage.Files) != 4 {
		t.Errorf("stored files left = %d, want the fresh and the attached image's 4", len(fixture.fileStorage.Files))
	}
}

func TestDeleteStalePendingUploadsAnnouncesEachRemovedUpload(t *testing.T) {
	fixture := newDeleteStalePendingUploadsFixture()

	if _, err := fixture.deleteStale(); err != nil {
		t.Fatalf("DeleteStalePendingUploads: %v", err)
	}

	if len(fixture.captured.recorded) != 2 || len(fixture.captured.published) != 2 {
		t.Fatalf("recorded = %d, published = %d, want 2 each",
			len(fixture.captured.recorded), len(fixture.captured.published))
	}
	for index, event := range fixture.captured.published {
		if event.Label != events.UploadDeletedLabel {
			t.Errorf("event %d label = %q, want %q", index, event.Label, events.UploadDeletedLabel)
		}
	}
}

func TestDeleteStalePendingUploadsTreatsAnAlreadyMissingFileAsDeleted(t *testing.T) {
	fixture := newDeleteStalePendingUploadsFixture()
	delete(fixture.fileStorage.Files, staleImageID.String()+"/preview.webp")

	result, err := fixture.deleteStale()
	if err != nil {
		t.Fatalf("DeleteStalePendingUploads: %v", err)
	}

	if result.DeletedCount != 2 {
		t.Errorf("deleted = %d, want 2 despite the missing preview", result.DeletedCount)
	}
}

func TestDeleteStalePendingUploadsKeepsRowsWhoseFilesCannotBeDeleted(t *testing.T) {
	fixture := newDeleteStalePendingUploadsFixture()
	storageFailure := errors.New("storage unreachable")
	fixture.fileStorage.DeleteFailure = storageFailure

	result, err := fixture.deleteStale()

	if _, ok := errors.AsType[e.ErrDeleteStoredFileFailed](err); !ok || !errors.Is(err, storageFailure) {
		t.Fatalf("err = %v, want ErrDeleteStoredFileFailed wrapping the storage failure", err)
	}
	if result.DeletedCount != 0 {
		t.Errorf("deleted = %d, want 0", result.DeletedCount)
	}
	if len(fixture.repository.Uploads) != 4 {
		t.Errorf("uploads left = %d, want all 4 kept for the next run", len(fixture.repository.Uploads))
	}
	if len(fixture.captured.published) != 0 {
		t.Errorf("published = %d, want no events", len(fixture.captured.published))
	}
}

func TestDeleteStalePendingUploadsReportsRowsTheDatabaseCannotDelete(t *testing.T) {
	fixture := newDeleteStalePendingUploadsFixture()
	fixture.repository.DeleteFailure = errDatabaseDown

	result, err := fixture.deleteStale()

	if !errors.Is(err, errDatabaseDown) {
		t.Fatalf("err = %v, want the delete failure", err)
	}
	if result.DeletedCount != 0 {
		t.Errorf("deleted = %d, want 0", result.DeletedCount)
	}
	if len(fixture.captured.published) != 0 {
		t.Errorf("published = %d, want no events", len(fixture.captured.published))
	}
}

func TestDeleteStalePendingUploadsReportsAFailedLookup(t *testing.T) {
	fixture := newDeleteStalePendingUploadsFixture()
	fixture.repository.ListFailure = errDatabaseDown

	result, err := fixture.deleteStale()

	if !errors.Is(err, errDatabaseDown) || result.DeletedCount != 0 {
		t.Fatalf("deleted = %d, err = %v, want 0 and the lookup failure", result.DeletedCount, err)
	}
	if len(fixture.fileStorage.Deleted) != 0 || len(fixture.repository.Uploads) != 4 {
		t.Errorf("deleted files = %v, uploads = %d, want nothing touched",
			fixture.fileStorage.Deleted, len(fixture.repository.Uploads))
	}
}
