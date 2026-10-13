package usecases

import (
	"context"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	writerepo "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	userevents "github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

// DeleteUserParams names who asks for the deletion and which account they want deleted.
type DeleteUserParams struct {
	ActingUserID uuid.UUID
	TargetUserID uuid.UUID
}

// DeleteUserDependencies carries the collaborators deleting a user needs.
type DeleteUserDependencies struct {
	UserWriteRepository writerepo.UserRepository
	EventDispatcher     events.Dispatcher
}

// CanDeleteUser reports whether the acting user may delete the target account.
func CanDeleteUser(params DeleteUserParams) bool {
	return params.ActingUserID == params.TargetUserID
}

// DeleteUser soft-deletes the caller's own account and profile, anonymizing both, and announces it.
func DeleteUser(ctx context.Context, d DeleteUserDependencies, params DeleteUserParams) error {
	if !CanDeleteUser(params) {
		return e.ErrUserDeletionForbidden{
			ActingUserID: params.ActingUserID,
			TargetUserID: params.TargetUserID,
		}
	}

	err := d.UserWriteRepository.SoftDeleteUser(ctx, models.SoftDeleteUserParams{
		ID:                  params.TargetUserID,
		AnonymizedEmail:     models.AnonymizedEmail(params.TargetUserID),
		AnonymizedFirstName: models.AnonymizedFirstName,
		AnonymizedLastName:  models.AnonymizedLastName,
	})
	if err != nil {
		return err
	}

	userevents.DispatchUserDeleted(ctx, d.EventDispatcher, userevents.NewUserDeletedParams{
		UserID: params.TargetUserID,
	})

	return nil
}
