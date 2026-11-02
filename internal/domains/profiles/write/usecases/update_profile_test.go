package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/repositories"
	uploaderrors "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	uploadrepositories "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

type updateProfileFixture struct {
	repository *repositories.InMemoryProfileRepository
	uploads    *uploadrepositories.InMemoryUploadRepository
	captured   *eventstest.CapturedEvents
}

func newUpdateProfileFixture(profiles ...models.Profile) updateProfileFixture {
	return updateProfileFixture{
		repository: repositories.NewInMemoryProfileRepository(profiles...),
		uploads:    pendingPictureUploads(),
		captured:   &eventstest.CapturedEvents{},
	}
}

func (f updateProfileFixture) update(params UpdateProfileParams) (UpdateProfileResult, error) {
	return UpdateProfile(context.Background(), UpdateProfileDependencies{
		ProfileWriteRepository: f.repository,
		EventDispatcher:        f.captured.Dispatcher(),
		AttachUpload:           f.uploads.AttachUpload,
	}, params)
}

func augustaParams() UpdateProfileParams {
	return UpdateProfileParams{UserID: testUserID, FirstName: "Augusta", LastName: "King"}
}

func TestUpdateProfileReplacesBothNames(t *testing.T) {
	fixture := newUpdateProfileFixture(storedProfile())

	result, err := fixture.update(augustaParams())
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}

	if result.Profile.FirstName != "Augusta" || result.Profile.LastName != "King" {
		t.Errorf("profile names = %q %q, want Augusta King", result.Profile.FirstName, result.Profile.LastName)
	}
	if !result.Profile.CreatedAt.Equal(storedAt) {
		t.Errorf("CreatedAt = %v, want the original %v", result.Profile.CreatedAt, storedAt)
	}
	if stored := fixture.repository.Profiles[testUserID]; stored != result.Profile {
		t.Errorf("stored = %+v, want the returned %+v", stored, result.Profile)
	}
}

func TestUpdateProfileAnnouncesTheUpdatedProfile(t *testing.T) {
	fixture := newUpdateProfileFixture(storedProfile())

	if _, err := fixture.update(augustaParams()); err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}

	if len(fixture.captured.Recorded) != 1 || len(fixture.captured.Published) != 1 {
		t.Fatalf("recorded/published events = %d/%d, want 1/1",
			len(fixture.captured.Recorded), len(fixture.captured.Published))
	}
	if label := fixture.captured.Recorded[0].Label; label != events.ProfileUpdatedLabel {
		t.Errorf("Label = %q, want %q", label, events.ProfileUpdatedLabel)
	}
}

func TestUpdateProfileRejectsAUserWithoutAProfile(t *testing.T) {
	fixture := newUpdateProfileFixture()

	_, err := fixture.update(augustaParams())

	if !errors.As(err, &e.ErrProfileNotFound{}) {
		t.Fatalf("err = %v, want ErrProfileNotFound", err)
	}
	if len(fixture.repository.Profiles) != 0 {
		t.Errorf("stored profiles = %d, want none created by an update", len(fixture.repository.Profiles))
	}
	if len(fixture.captured.Recorded) != 0 {
		t.Errorf("recorded events = %d, want none for a rejected update", len(fixture.captured.Recorded))
	}
}

func TestUpdateProfileReportsAnUpdateFailureWithoutAnnouncingIt(t *testing.T) {
	existing := storedProfile()
	fixture := newUpdateProfileFixture(existing)
	fixture.repository.UpdateFailure = e.ErrUpdateProfileFailed{UserID: testUserID, Cause: errDatabaseDown}

	_, err := fixture.update(augustaParams())

	if !errors.Is(err, errDatabaseDown) {
		t.Fatalf("err = %v, want it to wrap %v", err, errDatabaseDown)
	}
	if stored := fixture.repository.Profiles[testUserID]; stored != existing {
		t.Errorf("stored = %+v, want the existing %+v untouched", stored, existing)
	}
	if len(fixture.captured.Recorded) != 0 {
		t.Errorf("recorded events = %d, want none for a failed update", len(fixture.captured.Recorded))
	}
}

func TestUpdateProfileAttachesTheOwnersPictureUpload(t *testing.T) {
	fixture := newUpdateProfileFixture(storedProfile())
	params := augustaParams()
	params.PictureUploadID = pendingPictureID

	result, err := fixture.update(params)
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}

	if result.Profile.PictureUploadID != pendingPictureID {
		t.Errorf("PictureUploadID = %s, want %s", result.Profile.PictureUploadID, pendingPictureID)
	}
	if !isAttached(fixture.uploads, pendingPictureID) {
		t.Error("the picture upload is still pending, want it attached so the cleanup keeps it")
	}
}

func TestUpdateProfileRefusesAPictureThatIsNotTheOwnersProfilePicture(t *testing.T) {
	cases := []struct {
		name     string
		uploadID uuid.UUID
	}{
		{name: "unknown upload", uploadID: uuid.MustParse("0b6f3a52-1c4d-4e8f-9a2b-7c5d3e1f0a94")},
		{name: "upload of another purpose", uploadID: attachmentID},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newUpdateProfileFixture(storedProfile())
			params := augustaParams()
			params.PictureUploadID = testCase.uploadID

			_, err := fixture.update(params)

			if !errors.As(err, &uploaderrors.ErrUploadNotFound{}) {
				t.Errorf("err = %v, want ErrUploadNotFound", err)
			}
			if fixture.repository.Profiles[testUserID].PictureUploadID != uuid.Nil {
				t.Error("the refused picture was stored on the profile")
			}
			if len(fixture.captured.Published) != 0 {
				t.Errorf("published = %d, want no event for a refused picture", len(fixture.captured.Published))
			}
		})
	}
}
