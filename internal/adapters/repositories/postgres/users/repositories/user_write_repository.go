// Package repositories adapts the generated sqlc queries to the user ports the domain declares.
package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlnull"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/users/mappers"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
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

// SaveUser inserts the user and returns the row the database produced.
func (r UserWriteRepository) SaveUser(ctx context.Context, newUser models.NewUser) (models.User, error) {
	row, err := r.queries.CreateUser(ctx, mappers.CreateUserParamsMapper(newUser))
	if err != nil {
		return models.User{}, e.ErrSaveUserFailed{Email: newUser.Email, Cause: err}
	}

	return mappers.CreateUserMapper(row), nil
}

// SaveGoogleUser inserts the google user, verified and signed in, and returns the row the database produced.
func (r UserWriteRepository) SaveGoogleUser(ctx context.Context, newUser models.NewGoogleUser) (models.User, error) {
	row, err := r.queries.CreateGoogleUser(ctx, mappers.CreateGoogleUserParamsMapper(newUser))
	if err != nil {
		return models.User{}, e.ErrSaveUserFailed{Email: newUser.Email, Cause: err}
	}

	return mappers.CreateGoogleUserMapper(row), nil
}

// FindUserByGoogleSubject returns the write model of the user signed up with the google account,
// or e.ErrUserNotFound.
func (r UserWriteRepository) FindUserByGoogleSubject(ctx context.Context, googleSubject string) (models.User, error) {
	row, err := r.queries.FindUserByGoogleSubject(ctx, sqlnull.StringUnlessEmpty(googleSubject))
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, e.ErrUserNotFound{GoogleSubject: googleSubject}
	}
	if err != nil {
		return models.User{}, e.ErrFindUserFailed{GoogleSubject: googleSubject, Cause: err}
	}

	return mappers.UserFromGoogleSubjectMapper(row), nil
}

// FindUserByID returns the user's write model, or e.ErrUserNotFound.
func (r UserWriteRepository) FindUserByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	row, err := r.queries.FindUserByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, e.ErrUserNotFound{ID: id}
	}
	if err != nil {
		return models.User{}, e.ErrFindUserFailed{ID: id, Cause: err}
	}

	return mappers.UserFromIDMapper(row), nil
}

// FindUserByEmail returns the user's write model, or e.ErrUserNotFound.
func (r UserWriteRepository) FindUserByEmail(ctx context.Context, email string) (models.User, error) {
	row, err := r.queries.FindUserByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, e.ErrUserNotFound{Email: email}
	}
	if err != nil {
		return models.User{}, e.ErrFindUserFailed{Email: email, Cause: err}
	}

	return mappers.UserFromEmailMapper(row), nil
}

// MarkEmailVerified stamps email_verified_at and returns the stored user.
func (r UserWriteRepository) MarkEmailVerified(ctx context.Context, id uuid.UUID) (models.User, error) {
	row, err := r.queries.MarkUserEmailAsVerified(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, e.ErrUserNotFound{ID: id}
	}
	if err != nil {
		return models.User{}, e.ErrMarkEmailVerifiedFailed{UserID: id, Cause: err}
	}

	return mappers.MarkUserEmailAsVerifiedMapper(row), nil
}

// FindUserCredentialsByEmail returns the stored password hash, or e.ErrUserNotFound.
func (r UserWriteRepository) FindUserCredentialsByEmail(
	ctx context.Context,
	email string,
) (models.UserCredentials, error) {
	row, err := r.queries.FindUserCredentialsByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return models.UserCredentials{}, e.ErrUserNotFound{Email: email}
	}
	if err != nil {
		return models.UserCredentials{}, e.ErrFindUserFailed{Email: email, Cause: err}
	}

	return mappers.UserCredentialsFromEmailMapper(row), nil
}

// FindUserCredentialsByID returns the stored password hash, or e.ErrUserNotFound.
func (r UserWriteRepository) FindUserCredentialsByID(
	ctx context.Context,
	id uuid.UUID,
) (models.UserCredentials, error) {
	row, err := r.queries.FindUserCredentialsByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.UserCredentials{}, e.ErrUserNotFound{ID: id}
	}
	if err != nil {
		return models.UserCredentials{}, e.ErrFindUserFailed{ID: id, Cause: err}
	}

	return mappers.UserCredentialsFromIDMapper(row), nil
}

// ChangePassword stores the new password hash, or returns e.ErrUserNotFound.
func (r UserWriteRepository) ChangePassword(ctx context.Context, params models.ChangePasswordParams) error {
	_, err := r.queries.ChangeUserPassword(ctx, sqlc.ChangeUserPasswordParams{
		ID:           params.ID,
		PasswordHash: sqlnull.StringUnlessEmpty(params.PasswordHash),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return e.ErrUserNotFound{ID: params.ID}
	}
	if err != nil {
		return e.ErrChangePasswordFailed{UserID: params.ID, Cause: err}
	}

	return nil
}

// AddPassword stores the hash on a google-only user; a user that is missing, already has a password or has
// no google account matches no row and answers e.ErrPasswordAlreadySet.
func (r UserWriteRepository) AddPassword(ctx context.Context, params models.AddPasswordParams) error {
	_, err := r.queries.AddUserPassword(ctx, sqlc.AddUserPasswordParams{
		ID:           params.ID,
		PasswordHash: sqlnull.StringUnlessEmpty(params.PasswordHash),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return e.ErrPasswordAlreadySet{UserID: params.ID}
	}
	if err != nil {
		return e.ErrAddPasswordFailed{UserID: params.ID, Cause: err}
	}

	return nil
}

// AddGoogleAccount stores the google subject and stamps the email verified if not yet; a user that is missing or
// already has a google account matches no row and answers e.ErrGoogleAccountAlreadyConnected.
func (r UserWriteRepository) AddGoogleAccount(ctx context.Context, params models.AddGoogleAccountParams) error {
	_, err := r.queries.AddUserGoogleAccount(ctx, sqlc.AddUserGoogleAccountParams{
		ID:            params.ID,
		GoogleSubject: sqlnull.StringUnlessEmpty(params.GoogleSubject),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return e.ErrGoogleAccountAlreadyConnected{UserID: params.ID}
	}
	if postgres.IsUniqueViolation(err) {
		return e.ErrGoogleAccountTaken{GoogleSubject: params.GoogleSubject}
	}
	if err != nil {
		return e.ErrAddGoogleAccountFailed{UserID: params.ID, Cause: err}
	}

	return nil
}

// ChangeEmail swaps the email, stamps it verified, or returns e.ErrEmailTaken on a clash.
func (r UserWriteRepository) ChangeEmail(ctx context.Context, params models.ChangeEmailParams) (models.User, error) {
	row, err := r.queries.ChangeUserEmail(ctx, sqlc.ChangeUserEmailParams{
		ID:    params.ID,
		Email: params.Email,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, e.ErrUserNotFound{ID: params.ID}
	}
	if postgres.IsUniqueViolation(err) {
		return models.User{}, e.ErrEmailTaken{Email: params.Email}
	}
	if err != nil {
		return models.User{}, e.ErrChangeEmailFailed{UserID: params.ID, Cause: err}
	}

	return mappers.ChangeUserEmailMapper(row), nil
}

// MarkSignedIn stamps signin_at and returns the stored user.
func (r UserWriteRepository) MarkSignedIn(ctx context.Context, id uuid.UUID) (models.User, error) {
	row, err := r.queries.MarkUserSignedIn(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, e.ErrUserNotFound{ID: id}
	}
	if err != nil {
		return models.User{}, e.ErrMarkSignedInFailed{UserID: id, Cause: err}
	}

	return mappers.MarkUserSignedInMapper(row), nil
}

// SoftDeleteUser marks the user and its profile deleted in one statement, anonymizing both,
// or returns e.ErrUserNotFound.
func (r UserWriteRepository) SoftDeleteUser(ctx context.Context, params models.SoftDeleteUserParams) error {
	_, err := r.queries.SoftDeleteUser(ctx, sqlc.SoftDeleteUserParams{
		ID:        params.ID,
		Email:     params.AnonymizedEmail,
		FirstName: params.AnonymizedFirstName,
		LastName:  params.AnonymizedLastName,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return e.ErrUserNotFound{ID: params.ID}
	}
	if err != nil {
		return e.ErrSoftDeleteUserFailed{UserID: params.ID, Cause: err}
	}

	return nil
}
