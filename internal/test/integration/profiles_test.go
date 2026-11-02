package integration

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	profileerrors "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	profileusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/usecases"
	uploaderrors "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	uploadmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
	userusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

func TestDeletedUsersProfileCanNeitherBeUpdatedNorRead(t *testing.T) {
	world := newIntegrationWorld(t)
	userID := world.verifiedUserID(t)
	if _, err := world.createProfile(profileusecases.CreateProfileParams{
		UserID: userID, FirstName: "Ada", LastName: "Lovelace",
	}); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if err := world.deleteUser(userusecases.DeleteUserParams{UserID: userID}); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	_, err := world.updateProfile(profileusecases.UpdateProfileParams{UserID: userID, FirstName: "Augusta", LastName: "King"})
	if !errors.As(err, &profileerrors.ErrProfileNotFound{}) {
		t.Errorf("UpdateProfile on a deleted user err = %v, want ErrProfileNotFound", err)
	}

	_, err = world.profileReadRepository.GetProfileReadModelByID(world.ctx, userID)
	if !errors.As(err, &profileerrors.ErrProfileNotFound{}) {
		t.Errorf("GetProfileReadModelByID on a deleted user err = %v, want ErrProfileNotFound", err)
	}
}

func TestProfilePictureIsReadBackWithTheUploadsURLs(t *testing.T) {
	world := newIntegrationWorld(t)
	userID := world.verifiedUserID(t)
	picture := world.uploadImage(t, userID, uploadmodels.PurposeProfilePicture)
	world.createProfileWithPicture(t, createProfileWithPictureParams{UserID: userID, PictureUploadID: picture.ID})

	read, err := world.profileReadRepository.GetProfileReadModelByID(world.ctx, userID)
	if err != nil {
		t.Fatalf("GetProfileReadModelByID: %v", err)
	}

	if read.Picture.UploadID == nil || *read.Picture.UploadID != picture.ID {
		t.Errorf("Picture.UploadID = %v, want %s", read.Picture.UploadID, picture.ID)
	}
	if read.Picture.OriginalURL != nil {
		t.Errorf("Picture.OriginalURL = %q, want nil, as a profile picture keeps only its preview", *read.Picture.OriginalURL)
	}
	if read.Picture.PreviewURL == nil || *read.Picture.PreviewURL != picture.PreviewURL {
		t.Errorf("Picture.PreviewURL = %v, want %q", read.Picture.PreviewURL, picture.PreviewURL)
	}
}

func TestProfileWithoutPictureReadsBackNilPictureFields(t *testing.T) {
	world := newIntegrationWorld(t)
	userID := world.verifiedUserID(t)
	world.createProfileWithPicture(t, createProfileWithPictureParams{UserID: userID, PictureUploadID: uuid.Nil})

	read, err := world.profileReadRepository.GetProfileReadModelByID(world.ctx, userID)
	if err != nil {
		t.Fatalf("GetProfileReadModelByID: %v", err)
	}

	if read.Picture.UploadID != nil || read.Picture.PreviewURL != nil || read.Picture.OriginalURL != nil {
		t.Errorf("Picture = %+v, want every field nil", read.Picture)
	}
}

func TestProfilePictureMustBeTheOwnersProfilePictureUpload(t *testing.T) {
	world := newIntegrationWorld(t)
	userID := world.verifiedUserID(t)
	strangerID := world.verifiedUserID(t)

	cases := []struct {
		name   string
		upload uploadmodels.Upload
	}{
		{name: "another user's upload", upload: world.uploadImage(t, strangerID, uploadmodels.PurposeProfilePicture)},
		{name: "uploaded for another purpose", upload: world.uploadImage(t, userID, uploadmodels.PurposeDocumentAttachment)},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := world.createProfile(profileusecases.CreateProfileParams{
				UserID:          userID,
				FirstName:       "Ada",
				LastName:        "Lovelace",
				PictureUploadID: testCase.upload.ID,
			})

			if !errors.As(err, &uploaderrors.ErrUploadNotFound{}) {
				t.Errorf("err = %v, want ErrUploadNotFound", err)
			}
		})
	}
}

type createProfileWithPictureParams struct {
	UserID          uuid.UUID
	PictureUploadID uuid.UUID
}

func (w *integrationWorld) createProfileWithPicture(t *testing.T, params createProfileWithPictureParams) {
	t.Helper()

	_, err := w.createProfile(profileusecases.CreateProfileParams{
		UserID:          params.UserID,
		FirstName:       "Ada",
		LastName:        "Lovelace",
		PictureUploadID: params.PictureUploadID,
	})
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
}

func assertTextColumn(t *testing.T, column string, got *string, want string) {
	t.Helper()

	if got == nil || *got != want {
		t.Errorf("%s = %v, want %q", column, got, want)
	}
}
