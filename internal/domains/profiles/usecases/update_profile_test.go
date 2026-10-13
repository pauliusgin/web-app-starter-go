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

type updateProfileFixture struct {
	repository *repositories.InMemoryProfileRepository
	captured   *capturedEvents
}

func newUpdateProfileFixture(profiles ...models.Profile) updateProfileFixture {
	return updateProfileFixture{
		repository: repositories.NewInMemoryProfileRepository(profiles...),
		captured:   &capturedEvents{},
	}
}

func (f updateProfileFixture) update(params UpdateProfileParams) (UpdateProfileResult, error) {
	return UpdateProfile(context.Background(), UpdateProfileDependencies{
		ProfileRepository: f.repository,
		EventDispatcher:   f.captured.dispatcher(),
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

	if len(fixture.captured.recorded) != 1 || len(fixture.captured.published) != 1 {
		t.Fatalf("recorded/published events = %d/%d, want 1/1",
			len(fixture.captured.recorded), len(fixture.captured.published))
	}
	if label := fixture.captured.recorded[0].Label; label != events.ProfileUpdatedLabel {
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
	if len(fixture.captured.recorded) != 0 {
		t.Errorf("recorded events = %d, want none for a rejected update", len(fixture.captured.recorded))
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
	if len(fixture.captured.recorded) != 0 {
		t.Errorf("recorded events = %d, want none for a failed update", len(fixture.captured.recorded))
	}
}
