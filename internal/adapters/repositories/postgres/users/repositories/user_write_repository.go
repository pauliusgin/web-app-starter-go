// Package repositories adapts the generated sqlc queries to the user ports the domain declares.
package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/users/mappers"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	write "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
)

var _ write.UserRepository = UserWriteRepository{}

// UserWriteRepository stores users through the generated sqlc queries.
type UserWriteRepository struct {
	queries *sqlc.Queries
}

// NewUserWriteRepository builds a repository that writes through the given sqlc queries.
func NewUserWriteRepository(queries *sqlc.Queries) UserWriteRepository {
	return UserWriteRepository{queries: queries}
}

// Save inserts the user and returns the row the database produced.
func (r UserWriteRepository) Save(ctx context.Context, newUser models.NewUser) (models.User, error) {
	row, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           newUser.ID,
		Email:        newUser.Email,
		PasswordHash: newUser.PasswordHash,
	})
	if err != nil {
		return models.User{}, usererrors.ErrSaveUserFailed{Cause: err}
	}

	return mappers.CreateUserMapper(row), nil
}

// FindUserByID returns the user's write model, or usererrors.ErrUserNotFound.
func (r UserWriteRepository) FindUserByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	row, err := r.queries.FindUserByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, usererrors.ErrUserNotFound{}
	}
	if err != nil {
		return models.User{}, usererrors.ErrIDLookupFailed{Cause: err}
	}

	return mappers.UserFromIDMapper(row), nil
}

// FindUserByEmail returns the user's write model, or usererrors.ErrUserNotFound.
func (r UserWriteRepository) FindUserByEmail(ctx context.Context, email string) (models.User, error) {
	row, err := r.queries.FindUserByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, usererrors.ErrUserNotFound{}
	}
	if err != nil {
		return models.User{}, usererrors.ErrEmailLookupFailed{Cause: err}
	}

	return mappers.UserFromEmailMapper(row), nil
}

// MarkEmailVerified stamps email_verified_at and returns the stored user.
func (r UserWriteRepository) MarkEmailVerified(ctx context.Context, id uuid.UUID) (models.User, error) {
	row, err := r.queries.MarkUserEmailAsVerified(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, usererrors.ErrUserNotFound{}
	}
	if err != nil {
		return models.User{}, usererrors.ErrMarkEmailVerifiedFailed{Cause: err}
	}

	return mappers.MarkUserEmailAsVerifiedMapper(row), nil
}

// FindUserCredentialsByEmail returns the stored password hash, or usererrors.ErrUserNotFound.
func (r UserWriteRepository) FindUserCredentialsByEmail(
	ctx context.Context,
	email string,
) (models.UserCredentials, error) {
	row, err := r.queries.FindUserCredentialsByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return models.UserCredentials{}, usererrors.ErrUserNotFound{}
	}
	if err != nil {
		return models.UserCredentials{}, usererrors.ErrEmailLookupFailed{Cause: err}
	}

	return mappers.UserCredentialsFromEmailMapper(row), nil
}

// FindUserCredentialsByID returns the stored password hash, or usererrors.ErrUserNotFound.
func (r UserWriteRepository) FindUserCredentialsByID(
	ctx context.Context,
	id uuid.UUID,
) (models.UserCredentials, error) {
	row, err := r.queries.FindUserCredentialsByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.UserCredentials{}, usererrors.ErrUserNotFound{ID: id}
	}
	if err != nil {
		return models.UserCredentials{}, usererrors.ErrIDLookupFailed{Cause: err}
	}

	return mappers.UserCredentialsFromIDMapper(row), nil
}

// ChangePassword stores the new password hash, or returns usererrors.ErrUserNotFound.
func (r UserWriteRepository) ChangePassword(ctx context.Context, params models.ChangePasswordParams) error {
	_, err := r.queries.ChangeUserPassword(ctx, sqlc.ChangeUserPasswordParams{
		ID:           params.ID,
		PasswordHash: params.PasswordHash,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return usererrors.ErrUserNotFound{ID: params.ID}
	}
	if err != nil {
		return usererrors.ErrChangePasswordFailed{Cause: err}
	}

	return nil
}

// ChangeEmail swaps the email, stamps it verified, or returns usererrors.ErrEmailTaken on a clash.
func (r UserWriteRepository) ChangeEmail(ctx context.Context, params models.ChangeEmailParams) (models.User, error) {
	row, err := r.queries.ChangeUserEmail(ctx, sqlc.ChangeUserEmailParams{
		ID:    params.ID,
		Email: params.Email,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, usererrors.ErrUserNotFound{ID: params.ID}
	}
	if postgres.IsUniqueViolation(err) {
		return models.User{}, usererrors.ErrEmailTaken{Email: params.Email}
	}
	if err != nil {
		return models.User{}, usererrors.ErrChangeEmailFailed{Cause: err}
	}

	return mappers.ChangeUserEmailMapper(row), nil
}

// MarkSignedIn stamps signin_at and returns the stored user.
func (r UserWriteRepository) MarkSignedIn(ctx context.Context, id uuid.UUID) (models.User, error) {
	row, err := r.queries.MarkUserSignedIn(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, usererrors.ErrUserNotFound{}
	}
	if err != nil {
		return models.User{}, usererrors.ErrMarkSignedInFailed{Cause: err}
	}

	return mappers.MarkUserSignedInMapper(row), nil
}

// SoftDeleteUser marks the user and its profile deleted in one statement, anonymizing both,
// or returns usererrors.ErrUserNotFound.
func (r UserWriteRepository) SoftDeleteUser(ctx context.Context, params models.SoftDeleteUserParams) error {
	_, err := r.queries.SoftDeleteUser(ctx, sqlc.SoftDeleteUserParams{
		ID:        params.ID,
		Email:     params.AnonymizedEmail,
		FirstName: params.AnonymizedFirstName,
		LastName:  params.AnonymizedLastName,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return usererrors.ErrUserNotFound{ID: params.ID}
	}
	if err != nil {
		return usererrors.ErrSoftDeleteUserFailed{Cause: err}
	}

	return nil
}
