package usecases

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	userevents "github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

// DeleteUserParams is the account being deleted.
type DeleteUserParams struct {
	UserID uuid.UUID
}

// DeleteUserDependencies carries the user store anonymizing the account and the dispatcher announcing the deletion.
type DeleteUserDependencies struct {
	UserWriteRepository repositories.UserRepository
	EventDispatcher     events.Dispatcher
}

// DeleteUser soft-deletes the caller's own account and profile, anonymizing both, and announces it.
func DeleteUser(ctx context.Context, d DeleteUserDependencies, params DeleteUserParams) error {
	err := d.UserWriteRepository.SoftDeleteUser(ctx, models.SoftDeleteUserParams{
		ID:                  params.UserID,
		AnonymizedEmail:     models.AnonymizedEmail(params.UserID),
		AnonymizedFirstName: models.AnonymizedFirstName,
		AnonymizedLastName:  models.AnonymizedLastName,
	})
	if err != nil {
		return err
	}

	userevents.DispatchUserDeleted(ctx, d.EventDispatcher, userevents.NewUserDeletedParams{
		UserID: params.UserID,
	})

	return nil
}
