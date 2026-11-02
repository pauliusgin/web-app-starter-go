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

type createProfileFixture struct {
	repository *repositories.InMemoryProfileRepository
	uploads    *uploadrepositories.InMemoryUploadRepository
	captured   *eventstest.CapturedEvents
}

func newCreateProfileFixture(profiles ...models.Profile) createProfileFixture {
	return createProfileFixture{
		repository: repositories.NewInMemoryProfileRepository(profiles...),
		uploads:    pendingPictureUploads(),
		captured:   &eventstest.CapturedEvents{},
	}
}

func (f createProfileFixture) create(params CreateProfileParams) (CreateProfileResult, error) {
	return CreateProfile(context.Background(), CreateProfileDependencies{
		ProfileWriteRepository: f.repository,
		EventDispatcher:        f.captured.Dispatcher(),
		AttachUpload:           f.uploads.AttachUpload,
	}, params)
}

func adaParams() CreateProfileParams {
	return CreateProfileParams{UserID: testUserID, FirstName: "Ada", LastName: "Lovelace"}
}

func TestCreateProfileRecordsTheProfileUnderTheUserID(t *testing.T) {
	fixture := newCreateProfileFixture()

	result, err := fixture.create(adaParams())
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	if result.Profile.ID != testUserID || result.Profile.FirstName != "Ada" || result.Profile.LastName != "Lovelace" {
		t.Errorf("profile = %+v, want Ada Lovelace under %s", result.Profile, testUserID)
	}
	stored, ok := fixture.repository.Profiles[testUserID]
	if !ok {
		t.Fatalf("store holds no profile for %s", testUserID)
	}
	if stored != result.Profile {
		t.Errorf("stored = %+v, want the returned %+v", stored, result.Profile)
	}
}

func TestCreateProfileAnnouncesTheCreatedProfile(t *testing.T) {
	fixture := newCreateProfileFixture()

	if _, err := fixture.create(adaParams()); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	if len(fixture.captured.Recorded) != 1 || len(fixture.captured.Published) != 1 {
		t.Fatalf("recorded/published events = %d/%d, want 1/1",
			len(fixture.captured.Recorded), len(fixture.captured.Published))
	}
	if label := fixture.captured.Recorded[0].Label; label != events.ProfileCreatedLabel {
		t.Errorf("Label = %q, want %q", label, events.ProfileCreatedLabel)
	}
}

func TestCreateProfileRejectsASecondProfileForTheSameUser(t *testing.T) {
	existing := storedProfile()
	fixture := newCreateProfileFixture(existing)

	_, err := fixture.create(CreateProfileParams{UserID: testUserID, FirstName: "Augusta", LastName: "King"})

	if !errors.As(err, &e.ErrProfileAlreadyExists{}) {
		t.Fatalf("err = %v, want ErrProfileAlreadyExists", err)
	}
	if stored := fixture.repository.Profiles[testUserID]; stored != existing {
		t.Errorf("stored = %+v, want the existing %+v untouched", stored, existing)
	}
	if len(fixture.captured.Recorded) != 0 {
		t.Errorf("recorded events = %d, want none for a rejected creation", len(fixture.captured.Recorded))
	}
}

func TestCreateProfileReportsASaveFailureWithoutAnnouncingIt(t *testing.T) {
	fixture := newCreateProfileFixture()
	fixture.repository.SaveFailure = e.ErrSaveProfileFailed{UserID: testUserID, Cause: errDatabaseDown}

	_, err := fixture.create(adaParams())

	if !errors.Is(err, errDatabaseDown) {
		t.Fatalf("err = %v, want it to wrap %v", err, errDatabaseDown)
	}
	if len(fixture.repository.Profiles) != 0 {
		t.Errorf("stored profiles = %d, want none", len(fixture.repository.Profiles))
	}
	if len(fixture.captured.Recorded) != 0 {
		t.Errorf("recorded events = %d, want none for a failed save", len(fixture.captured.Recorded))
	}
}

func TestCreateProfileAttachesTheOwnersPictureUpload(t *testing.T) {
	fixture := newCreateProfileFixture()
	params := adaParams()
	params.PictureUploadID = pendingPictureID

	result, err := fixture.create(params)
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	if result.Profile.PictureUploadID != pendingPictureID {
		t.Errorf("PictureUploadID = %s, want %s", result.Profile.PictureUploadID, pendingPictureID)
	}
	if !isAttached(fixture.uploads, pendingPictureID) {
		t.Error("the picture upload is still pending, want it attached so the cleanup keeps it")
	}
}

func TestCreateProfileRefusesAPictureThatIsNotTheOwnersProfilePicture(t *testing.T) {
	cases := []struct {
		name     string
		uploadID uuid.UUID
	}{
		{name: "unknown upload", uploadID: uuid.MustParse("0b6f3a52-1c4d-4e8f-9a2b-7c5d3e1f0a94")},
		{name: "upload of another purpose", uploadID: attachmentID},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newCreateProfileFixture()
			params := adaParams()
			params.PictureUploadID = testCase.uploadID

			_, err := fixture.create(params)

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
