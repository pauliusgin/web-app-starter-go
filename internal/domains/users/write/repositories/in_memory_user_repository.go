package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
)

var _ UserRepository = (*InMemoryUserRepository)(nil)

// InMemoryUser is one stored row: the write model plus the hash only credential lookups expose
// and the google subject a google sign-in matches on.
type InMemoryUser struct {
	User          models.User
	PasswordHash  string
	GoogleSubject string
}

// InMemoryUserRepository is a UserRepository that keeps live users in a map keyed by user ID.
type InMemoryUserRepository struct {
	Users map[uuid.UUID]InMemoryUser

	// SoftDeleted holds the anonymizing request each deleted user was removed from Users with.
	SoftDeleted map[uuid.UUID]models.SoftDeleteUserParams

	// LookupFailure is returned by every Find method; the other failures by their own method.
	SaveFailure           error
	LookupFailure         error
	MarkVerifiedFailure   error
	MarkSignedInFailure   error
	ChangeEmailFailure    error
	ChangePasswordFailure error
	AddPasswordFailure    error
	AddGoogleFailure      error
	SoftDeleteFailure     error
}

// NewInMemoryUserRepository builds a repository holding the given users and no failure staged.
func NewInMemoryUserRepository(users ...InMemoryUser) *InMemoryUserRepository {
	stored := make(map[uuid.UUID]InMemoryUser, len(users))
	for _, user := range users {
		stored[user.User.ID] = user
	}

	return &InMemoryUserRepository{Users: stored, SoftDeleted: map[uuid.UUID]models.SoftDeleteUserParams{}}
}

// SaveUser records the user signed in at creation, as CreateUser does, refusing an email already held
// the way the users.email unique constraint surfaces through the Postgres adapter.
func (r *InMemoryUserRepository) SaveUser(_ context.Context, newUser models.NewUser) (models.User, error) {
	if r.SaveFailure != nil {
		return models.User{}, r.SaveFailure
	}
	if _, taken := r.findByEmail(newUser.Email); taken {
		return models.User{}, e.ErrSaveUserFailed{Email: newUser.Email, Cause: e.ErrEmailTaken{Email: newUser.Email}}
	}

	savedAt := time.Now()
	saved := models.User{
		ID:        newUser.ID,
		Email:     newUser.Email,
		CreatedAt: savedAt,
		UpdatedAt: savedAt,
		SigninAt:  savedAt,
	}
	r.Users[saved.ID] = InMemoryUser{User: saved, PasswordHash: newUser.PasswordHash}

	return saved, nil
}

// SaveGoogleUser records the user verified and signed in at creation, as CreateGoogleUser does, refusing
// an email or google subject already held the way the unique constraints surface through the adapter.
func (r *InMemoryUserRepository) SaveGoogleUser(_ context.Context, newUser models.NewGoogleUser) (models.User, error) {
	if r.SaveFailure != nil {
		return models.User{}, r.SaveFailure
	}
	if _, taken := r.findByEmail(newUser.Email); taken {
		return models.User{}, e.ErrSaveUserFailed{Email: newUser.Email, Cause: e.ErrEmailTaken{Email: newUser.Email}}
	}
	if _, taken := r.findByGoogleSubject(newUser.GoogleSubject); taken {
		return models.User{}, e.ErrSaveUserFailed{
			Email: newUser.Email,
			Cause: e.ErrGoogleAccountTaken{GoogleSubject: newUser.GoogleSubject},
		}
	}

	savedAt := time.Now()
	saved := models.User{
		ID:              newUser.ID,
		Email:           newUser.Email,
		CreatedAt:       savedAt,
		UpdatedAt:       savedAt,
		EmailVerifiedAt: savedAt,
		SigninAt:        savedAt,
	}
	r.Users[saved.ID] = InMemoryUser{User: saved, GoogleSubject: newUser.GoogleSubject}

	return saved, nil
}

// FindUserByGoogleSubject returns the live user signed up with the google account, or e.ErrUserNotFound.
func (r *InMemoryUserRepository) FindUserByGoogleSubject(
	_ context.Context,
	googleSubject string,
) (models.User, error) {
	if r.LookupFailure != nil {
		return models.User{}, r.LookupFailure
	}

	stored, found := r.findByGoogleSubject(googleSubject)
	if !found {
		return models.User{}, e.ErrUserNotFound{GoogleSubject: googleSubject}
	}

	return stored.User, nil
}

// FindUserByID returns the live user, or e.ErrUserNotFound.
func (r *InMemoryUserRepository) FindUserByID(_ context.Context, id uuid.UUID) (models.User, error) {
	if r.LookupFailure != nil {
		return models.User{}, r.LookupFailure
	}

	stored, found := r.Users[id]
	if !found {
		return models.User{}, e.ErrUserNotFound{ID: id}
	}

	return stored.User, nil
}

// FindUserByEmail returns the live user holding the email, or e.ErrUserNotFound.
func (r *InMemoryUserRepository) FindUserByEmail(_ context.Context, email string) (models.User, error) {
	if r.LookupFailure != nil {
		return models.User{}, r.LookupFailure
	}

	stored, found := r.findByEmail(email)
	if !found {
		return models.User{}, e.ErrUserNotFound{Email: email}
	}

	return stored.User, nil
}

// FindUserCredentialsByEmail returns the live user's password hash, or e.ErrUserNotFound.
func (r *InMemoryUserRepository) FindUserCredentialsByEmail(
	_ context.Context,
	email string,
) (models.UserCredentials, error) {
	if r.LookupFailure != nil {
		return models.UserCredentials{}, r.LookupFailure
	}

	stored, found := r.findByEmail(email)
	if !found {
		return models.UserCredentials{}, e.ErrUserNotFound{Email: email}
	}

	return stored.credentials(), nil
}

// FindUserCredentialsByID returns the live user's password hash, or e.ErrUserNotFound.
func (r *InMemoryUserRepository) FindUserCredentialsByID(
	_ context.Context,
	id uuid.UUID,
) (models.UserCredentials, error) {
	if r.LookupFailure != nil {
		return models.UserCredentials{}, r.LookupFailure
	}

	stored, found := r.Users[id]
	if !found {
		return models.UserCredentials{}, e.ErrUserNotFound{ID: id}
	}

	return stored.credentials(), nil
}

// MarkEmailVerified stamps the wall clock as verified, or returns e.ErrUserNotFound.
func (r *InMemoryUserRepository) MarkEmailVerified(_ context.Context, id uuid.UUID) (models.User, error) {
	if r.MarkVerifiedFailure != nil {
		return models.User{}, r.MarkVerifiedFailure
	}

	return r.changeUser(id, func(user *InMemoryUser, now time.Time) {
		user.User.EmailVerifiedAt = now
	})
}

// MarkSignedIn stamps the wall clock as the last sign-in, or returns e.ErrUserNotFound.
func (r *InMemoryUserRepository) MarkSignedIn(_ context.Context, id uuid.UUID) (models.User, error) {
	if r.MarkSignedInFailure != nil {
		return models.User{}, r.MarkSignedInFailure
	}

	return r.changeUser(id, func(user *InMemoryUser, now time.Time) {
		user.User.SigninAt = now
	})
}

// ChangePassword swaps in the new hash, or returns e.ErrUserNotFound.
func (r *InMemoryUserRepository) ChangePassword(_ context.Context, params models.ChangePasswordParams) error {
	if r.ChangePasswordFailure != nil {
		return r.ChangePasswordFailure
	}

	_, err := r.changeUser(params.ID, func(user *InMemoryUser, _ time.Time) {
		user.PasswordHash = params.PasswordHash
	})

	return err
}

// AddPassword stores the hash on a live google-only user; any other user answers e.ErrPasswordAlreadySet,
// as the guarded AddUserPassword update matching no row does.
func (r *InMemoryUserRepository) AddPassword(_ context.Context, params models.AddPasswordParams) error {
	if r.AddPasswordFailure != nil {
		return r.AddPasswordFailure
	}
	stored, found := r.Users[params.ID]
	if !found || stored.PasswordHash != "" || stored.GoogleSubject == "" {
		return e.ErrPasswordAlreadySet{UserID: params.ID}
	}

	_, err := r.changeUser(params.ID, func(user *InMemoryUser, _ time.Time) {
		user.PasswordHash = params.PasswordHash
	})

	return err
}

// AddGoogleAccount stores the subject on a live user without one and stamps its email verified if not yet, as
// the guarded AddUserGoogleAccount update does; a subject another user holds answers e.ErrGoogleAccountTaken.
func (r *InMemoryUserRepository) AddGoogleAccount(_ context.Context, params models.AddGoogleAccountParams) error {
	if r.AddGoogleFailure != nil {
		return r.AddGoogleFailure
	}
	stored, found := r.Users[params.ID]
	if !found || stored.GoogleSubject != "" {
		return e.ErrGoogleAccountAlreadyConnected{UserID: params.ID}
	}
	if _, taken := r.findByGoogleSubject(params.GoogleSubject); taken {
		return e.ErrGoogleAccountTaken{GoogleSubject: params.GoogleSubject}
	}

	_, err := r.changeUser(params.ID, func(user *InMemoryUser, now time.Time) {
		user.GoogleSubject = params.GoogleSubject
		if user.User.EmailVerifiedAt.IsZero() {
			user.User.EmailVerifiedAt = now
		}
	})

	return err
}

// ChangeEmail swaps in the email stamped verified, refusing one another live user holds as the
// unique constraint does.
func (r *InMemoryUserRepository) ChangeEmail(_ context.Context, params models.ChangeEmailParams) (models.User, error) {
	if r.ChangeEmailFailure != nil {
		return models.User{}, r.ChangeEmailFailure
	}
	if holder, taken := r.findByEmail(params.Email); taken && holder.User.ID != params.ID {
		return models.User{}, e.ErrEmailTaken{Email: params.Email}
	}

	changed, err := r.changeUser(params.ID, func(user *InMemoryUser, now time.Time) {
		user.User.Email = params.Email
		user.User.EmailVerifiedAt = now
	})
	if err != nil {
		return models.User{}, e.ErrUserNotFound{ID: params.ID}
	}

	return changed, nil
}

// SoftDeleteUser moves the user from Users to SoftDeleted, so every lookup misses it as the
// deleted_at filter does and its email is free again.
func (r *InMemoryUserRepository) SoftDeleteUser(_ context.Context, params models.SoftDeleteUserParams) error {
	if r.SoftDeleteFailure != nil {
		return r.SoftDeleteFailure
	}
	if _, found := r.Users[params.ID]; !found {
		return e.ErrUserNotFound{ID: params.ID}
	}

	delete(r.Users, params.ID)
	r.SoftDeleted[params.ID] = params

	return nil
}

func (r *InMemoryUserRepository) findByEmail(email string) (InMemoryUser, bool) {
	for _, stored := range r.Users {
		if stored.User.Email == email {
			return stored, true
		}
	}

	return InMemoryUser{}, false
}

func (r *InMemoryUserRepository) findByGoogleSubject(googleSubject string) (InMemoryUser, bool) {
	for _, stored := range r.Users {
		if stored.GoogleSubject != "" && stored.GoogleSubject == googleSubject {
			return stored, true
		}
	}

	return InMemoryUser{}, false
}

func (r *InMemoryUserRepository) changeUser(
	id uuid.UUID,
	change func(user *InMemoryUser, now time.Time),
) (models.User, error) {
	stored, found := r.Users[id]
	if !found {
		return models.User{}, e.ErrUserNotFound{ID: id}
	}

	now := time.Now()
	change(&stored, now)
	stored.User.UpdatedAt = now
	r.Users[id] = stored

	return stored.User, nil
}

func (u InMemoryUser) credentials() models.UserCredentials {
	return models.UserCredentials{
		ID:            u.User.ID,
		Email:         u.User.Email,
		PasswordHash:  u.PasswordHash,
		GoogleSubject: u.GoogleSubject,
	}
}
