package integration

import (
	"testing"

	"github.com/google/uuid"

	profileusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/usecases"
	uploadmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
	uploadusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/usecases"
	userusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
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

// backdateUploadAttachment moves an upload's attached_at past the orphan retention, standing in for a day going by.
func (w *integrationWorld) backdateUploadAttachment(t *testing.T, uploadID uuid.UUID) {
	t.Helper()

	_, err := w.database.ExecContext(w.ctx,
		`UPDATE uploads SET attached_at = NOW() - INTERVAL '25 hours' WHERE id = $1`, uploadID)
	if err != nil {
		t.Fatalf("backdating upload attachment %s: %v", uploadID, err)
	}
}

// referenceUploadAsProfilePicture points the upload owner's profile at the upload, creating a bare profile when the
// owner has none, so the upload contract can seed a reference without going through the profile use cases.
func (w *integrationWorld) referenceUploadAsProfilePicture(t *testing.T, upload uploadmodels.Upload) {
	t.Helper()

	_, err := w.database.ExecContext(w.ctx,
		`INSERT INTO profiles (id, first_name, last_name, picture_upload_id) VALUES ($1, 'Ada', 'Lovelace', $2)
		ON CONFLICT (id) DO UPDATE SET picture_upload_id = EXCLUDED.picture_upload_id`,
		upload.UserID, upload.ID)
	if err != nil {
		t.Fatalf("referencing upload %s as a profile picture: %v", upload.ID, err)
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
	if _, err := world.createProfile(profileusecases.CreateProfileParams{
		UserID:          userID,
		FirstName:       "Ada",
		LastName:        "Lovelace",
		PictureUploadID: staleAttached.ID,
	}); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
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

func TestOrphanedAttachedUploadsArePurgedButReferencedAndRecentOnesKept(t *testing.T) {
	world := newIntegrationWorld(t)
	userID := world.verifiedUserID(t)

	replacedPicture := world.uploadImage(t, userID, uploadmodels.PurposeProfilePicture)
	currentPicture := world.uploadImage(t, userID, uploadmodels.PurposeProfilePicture)
	recentlyReplacedPicture := world.uploadImage(t, userID, uploadmodels.PurposeProfilePicture)
	world.createProfileWithPicture(t, createProfileWithPictureParams{UserID: userID, PictureUploadID: replacedPicture.ID})
	world.replaceProfilePicture(t, replaceProfilePictureParams{UserID: userID, PictureUploadID: recentlyReplacedPicture.ID})
	world.replaceProfilePicture(t, replaceProfilePictureParams{UserID: userID, PictureUploadID: currentPicture.ID})
	world.backdateUploadAttachment(t, replacedPicture.ID)
	world.backdateUploadAttachment(t, currentPicture.ID)

	purged, err := world.deleteOrphanedAttachedUploads()
	if err != nil {
		t.Fatalf("DeleteOrphanedAttachedUploads: %v", err)
	}

	if purged.DeletedCount != 1 {
		t.Errorf("DeletedCount = %d, want only the long-replaced picture", purged.DeletedCount)
	}
	if world.uploadExists(t, replacedPicture.ID) {
		t.Error("the long-replaced picture is still stored")
	}
	if !world.uploadExists(t, currentPicture.ID) || !world.uploadExists(t, recentlyReplacedPicture.ID) {
		t.Error("the current or the recently replaced picture was purged")
	}
	if _, stillStored := world.files.Files[replacedPicture.PreviewPath]; stillStored {
		t.Error("the replaced picture's preview file is still in storage")
	}
}

// The upload_references view skips soft-deleted profiles, so a deleted account's picture becomes an orphan.
func TestDeletedUsersProfilePictureIsPurgedAsAnOrphan(t *testing.T) {
	world := newIntegrationWorld(t)
	userID := world.verifiedUserID(t)
	picture := world.uploadImage(t, userID, uploadmodels.PurposeProfilePicture)
	world.createProfileWithPicture(t, createProfileWithPictureParams{UserID: userID, PictureUploadID: picture.ID})
	if err := world.deleteUser(userusecases.DeleteUserParams{UserID: userID}); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	world.backdateUploadAttachment(t, picture.ID)

	purged, err := world.deleteOrphanedAttachedUploads()
	if err != nil {
		t.Fatalf("DeleteOrphanedAttachedUploads: %v", err)
	}

	if purged.DeletedCount != 1 || world.uploadExists(t, picture.ID) {
		t.Errorf("DeletedCount = %d, want the deleted user's picture purged", purged.DeletedCount)
	}
}

func (w *integrationWorld) deleteOrphanedAttachedUploads() (uploadusecases.DeleteOrphanedAttachedUploadsResult, error) {
	return uploadusecases.DeleteOrphanedAttachedUploads(w.ctx, uploadusecases.DeleteOrphanedAttachedUploadsDependencies{
		UploadWriteRepository: w.uploadRepository,
		FileStorage:           w.files,
		EventDispatcher:       w.eventDispatcher,
	})
}

type replaceProfilePictureParams struct {
	UserID          uuid.UUID
	PictureUploadID uuid.UUID
}

func (w *integrationWorld) replaceProfilePicture(t *testing.T, params replaceProfilePictureParams) {
	t.Helper()

	_, err := w.updateProfile(profileusecases.UpdateProfileParams{
		UserID:          params.UserID,
		FirstName:       "Ada",
		LastName:        "Lovelace",
		PictureUploadID: params.PictureUploadID,
	})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
}
