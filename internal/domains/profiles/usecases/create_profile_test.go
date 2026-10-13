package usecases

import (
	"context"
	"errors"
	"testing"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type createProfileFixture struct {
	repository *repositories.InMemoryProfileRepository
	captured   *capturedEvents
}

func newCreateProfileFixture(profiles ...models.Profile) createProfileFixture {
	return createProfileFixture{
		repository: repositories.NewInMemoryProfileRepository(profiles...),
		captured:   &capturedEvents{},
	}
}

func (f createProfileFixture) create(params CreateProfileParams) (CreateProfileResult, error) {
	return CreateProfile(context.Background(), CreateProfileDependencies{
		ProfileRepository: f.repository,
		EventDispatcher:   f.captured.dispatcher(),
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

	if len(fixture.captured.recorded) != 1 || len(fixture.captured.published) != 1 {
		t.Fatalf("recorded/published events = %d/%d, want 1/1",
			len(fixture.captured.recorded), len(fixture.captured.published))
	}
	if label := fixture.captured.recorded[0].Label; label != events.ProfileCreatedLabel {
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
	if len(fixture.captured.recorded) != 0 {
		t.Errorf("recorded events = %d, want none for a rejected creation", len(fixture.captured.recorded))
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
	if len(fixture.captured.recorded) != 0 {
		t.Errorf("recorded events = %d, want none for a failed save", len(fixture.captured.recorded))
	}
}
