package integration

import (
	"errors"
	"testing"

	profileerrors "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	profileusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/usecases"
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

	_, err := profileusecases.UpdateProfile(world.ctx, profileusecases.UpdateProfileDependencies{
		ProfileWriteRepository: world.profileRepository,
		EventDispatcher:        world.eventDispatcher,
	}, profileusecases.UpdateProfileParams{UserID: userID, FirstName: "Augusta", LastName: "King"})
	if !errors.As(err, &profileerrors.ErrProfileNotFound{}) {
		t.Errorf("UpdateProfile on a deleted user err = %v, want ErrProfileNotFound", err)
	}

	_, err = world.profileReadRepository.GetProfileReadModelByID(world.ctx, userID)
	if !errors.As(err, &profileerrors.ErrProfileNotFound{}) {
		t.Errorf("GetProfileReadModelByID on a deleted user err = %v, want ErrProfileNotFound", err)
	}
}
