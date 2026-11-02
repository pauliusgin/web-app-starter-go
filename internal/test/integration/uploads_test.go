package integration

import (
	"testing"

	"github.com/google/uuid"

	uploadmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
	uploadusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/usecases"
)

// backdateUpload moves an upload's created_at past the pending retention, standing in for a day going by.
func (w *integrationWorld) backdateUpload(t *testing.T, uploadID uuid.UUID) {
	t.Helper()

	_, err := w.database.ExecContext(w.ctx,
		`UPDATE uploads SET created_at = NOW() - INTERVAL '25 hours' WHERE id = $1`, uploadID)
	if err != nil {
		t.Fatalf("backdating upload %s: %v", uploadID, err)
	}
}

// attachUpload stamps attached_at the way a domain referencing the upload would; the starter ships no such domain.
func (w *integrationWorld) attachUpload(t *testing.T, uploadID uuid.UUID) {
	t.Helper()

	_, err := w.database.ExecContext(w.ctx, `UPDATE uploads SET attached_at = NOW() WHERE id = $1`, uploadID)
	if err != nil {
		t.Fatalf("attaching upload %s: %v", uploadID, err)
	}
}

func (w *integrationWorld) uploadExists(t *testing.T, uploadID uuid.UUID) bool {
	t.Helper()

	var exists bool
	err := w.database.QueryRowContext(w.ctx, `SELECT EXISTS (SELECT 1 FROM uploads WHERE id = $1)`, uploadID).Scan(&exists)
	if err != nil {
		t.Fatalf("checking upload %s: %v", uploadID, err)
	}

	return exists
}

func TestStalePendingUploadsArePurgedButAttachedAndRecentOnesKept(t *testing.T) {
	world := newIntegrationWorld(t)
	userID := world.verifiedUserID(t)

	stalePending := world.uploadImage(t, userID, uploadmodels.PurposeProfilePicture)
	staleAttached := world.uploadImage(t, userID, uploadmodels.PurposeProfilePicture)
	recentPending := world.uploadImage(t, userID, uploadmodels.PurposeProfilePicture)
	world.attachUpload(t, staleAttached.ID)
	world.backdateUpload(t, stalePending.ID)
	world.backdateUpload(t, staleAttached.ID)

	purged, err := uploadusecases.DeleteStalePendingUploads(world.ctx, uploadusecases.DeleteStalePendingUploadsDependencies{
		UploadWriteRepository: world.uploadRepository,
		FileStorage:           world.files,
		EventDispatcher:       world.eventDispatcher,
	})
	if err != nil {
		t.Fatalf("DeleteStalePendingUploads: %v", err)
	}

	if purged.DeletedCount != 1 {
		t.Errorf("DeletedCount = %d, want only the stale pending upload", purged.DeletedCount)
	}
	if world.uploadExists(t, stalePending.ID) {
		t.Error("the stale pending upload is still stored")
	}
	if !world.uploadExists(t, staleAttached.ID) || !world.uploadExists(t, recentPending.ID) {
		t.Error("an attached or recent upload was purged")
	}
	if _, stillStored := world.files.Files[stalePending.PreviewPath]; stillStored {
		t.Error("the stale upload's preview file is still in storage")
	}
}
