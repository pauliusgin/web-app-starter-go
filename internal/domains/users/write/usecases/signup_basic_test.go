package usecases

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	user "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const testEmail = "accountant@example.com"

const testPassword = "correct horse battery staple"

const seededHash = "$argon2id$seeded"

var fixedCreatedAt = time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)

var fixedUpdatedAt = time.Date(2026, time.September, 1, 12, 0, 1, 0, time.UTC)

var fixedOccurredAt = time.Date(2026, time.September, 1, 12, 0, 2, 0, time.UTC)

var fixedExpiresAt = time.Date(2026, time.September, 1, 12, 5, 0, 0, time.UTC)

var fixedVerifiedAt = time.Date(2026, time.September, 1, 12, 3, 0, 0, time.UTC)

var fixedSignedInAt = time.Date(2026, time.September, 2, 8, 30, 0, 0, time.UTC)

type inMemoryUserStore struct {
	savedUsers            []user.NewUser
	failure               error
	lookupFailure         error
	returnedUser          *user.User
	verifiedAt            map[uuid.UUID]time.Time
	markVerifiedFailure   error
	signedInAt            map[uuid.UUID]time.Time
	markSignedInFailure   error
	softDeletedUsers      []user.NewUser
	softDeleteRequests    []user.SoftDeleteUserParams
	softDeleteFailure     error
	changeEmailFailure    error
	changePasswordFailure error
}

func newInMemoryUserStore() *inMemoryUserStore {
	return &inMemoryUserStore{
		verifiedAt: map[uuid.UUID]time.Time{},
		signedInAt: map[uuid.UUID]time.Time{},
	}
}

func (s *inMemoryUserStore) MarkEmailVerified(_ context.Context, id uuid.UUID) (user.User, error) {
	if s.markVerifiedFailure != nil {
		return user.User{}, s.markVerifiedFailure
	}

	for _, savedUser := range s.savedUsers {
		if savedUser.ID != id {
			continue
		}

		s.verifiedAt[id] = fixedVerifiedAt

		return s.userRow(savedUser), nil
	}

	return user.User{}, usererrors.ErrUserNotFound{}
}

func (s *inMemoryUserStore) FindUserCredentialsByEmail(
	_ context.Context,
	email string,
) (user.UserCredentials, error) {
	if s.lookupFailure != nil {
		return user.UserCredentials{}, s.lookupFailure
	}

	for _, savedUser := range s.savedUsers {
		if savedUser.Email == email {
			return user.UserCredentials{
				ID:           savedUser.ID,
				Email:        savedUser.Email,
				PasswordHash: savedUser.PasswordHash,
			}, nil
		}
	}

	return user.UserCredentials{}, usererrors.ErrUserNotFound{}
}

func (s *inMemoryUserStore) FindUserCredentialsByID(
	_ context.Context,
	id uuid.UUID,
) (user.UserCredentials, error) {
	if s.lookupFailure != nil {
		return user.UserCredentials{}, s.lookupFailure
	}

	for _, savedUser := range s.savedUsers {
		if savedUser.ID == id {
			return user.UserCredentials{
				ID:           savedUser.ID,
				Email:        savedUser.Email,
				PasswordHash: savedUser.PasswordHash,
			}, nil
		}
	}

	return user.UserCredentials{}, usererrors.ErrUserNotFound{ID: id}
}

func (s *inMemoryUserStore) ChangePassword(_ context.Context, params user.ChangePasswordParams) error {
	if s.changePasswordFailure != nil {
		return s.changePasswordFailure
	}

	for index, savedUser := range s.savedUsers {
		if savedUser.ID != params.ID {
			continue
		}

		s.savedUsers[index].PasswordHash = params.PasswordHash

		return nil
	}

	return usererrors.ErrUserNotFound{ID: params.ID}
}

// ChangeEmail refuses an email another user holds, as the users.email unique constraint does.
func (s *inMemoryUserStore) ChangeEmail(_ context.Context, params user.ChangeEmailParams) (user.User, error) {
	if s.changeEmailFailure != nil {
		return user.User{}, s.changeEmailFailure
	}

	for _, savedUser := range s.savedUsers {
		if savedUser.Email == params.Email && savedUser.ID != params.ID {
			return user.User{}, usererrors.ErrEmailTaken{Email: params.Email}
		}
	}

	for index, savedUser := range s.savedUsers {
		if savedUser.ID != params.ID {
			continue
		}

		s.savedUsers[index].Email = params.Email
		s.verifiedAt[params.ID] = fixedVerifiedAt

		return s.userRow(s.savedUsers[index]), nil
	}

	return user.User{}, usererrors.ErrUserNotFound{ID: params.ID}
}

func (s *inMemoryUserStore) MarkSignedIn(_ context.Context, id uuid.UUID) (user.User, error) {
	if s.markSignedInFailure != nil {
		return user.User{}, s.markSignedInFailure
	}

	for _, savedUser := range s.savedUsers {
		if savedUser.ID != id {
			continue
		}

		s.signedInAt[id] = fixedSignedInAt

		return s.userRow(savedUser), nil
	}

	return user.User{}, usererrors.ErrUserNotFound{}
}

// SoftDeleteUser moves the user out of savedUsers, so every lookup misses it as the deleted_at filter does.
func (s *inMemoryUserStore) SoftDeleteUser(_ context.Context, params user.SoftDeleteUserParams) error {
	if s.softDeleteFailure != nil {
		return s.softDeleteFailure
	}

	for index, savedUser := range s.savedUsers {
		if savedUser.ID != params.ID {
			continue
		}

		savedUser.Email = params.AnonymizedEmail
		s.softDeleteRequests = append(s.softDeleteRequests, params)
		s.softDeletedUsers = append(s.softDeletedUsers, savedUser)
		s.savedUsers = append(s.savedUsers[:index], s.savedUsers[index+1:]...)

		return nil
	}

	return usererrors.ErrUserNotFound{ID: params.ID}
}

func (s *inMemoryUserStore) userRow(savedUser user.NewUser) user.User {
	return user.User{
		ID:              savedUser.ID,
		Email:           savedUser.Email,
		CreatedAt:       fixedCreatedAt,
		UpdatedAt:       fixedUpdatedAt,
		EmailVerifiedAt: s.verifiedAt[savedUser.ID],
		SigninAt:        s.signedInAt[savedUser.ID],
	}
}

func (s *inMemoryUserStore) Save(_ context.Context, newUser user.NewUser) (user.User, error) {
	if s.failure != nil {
		return user.User{}, s.failure
	}

	s.savedUsers = append(s.savedUsers, newUser)

	if s.returnedUser != nil {
		return *s.returnedUser, nil
	}

	return user.User{
		ID:        newUser.ID,
		Email:     newUser.Email,
		CreatedAt: fixedCreatedAt,
		UpdatedAt: fixedUpdatedAt,
	}, nil
}

func (s *inMemoryUserStore) FindUserByEmail(_ context.Context, email string) (user.User, error) {
	if s.lookupFailure != nil {
		return user.User{}, s.lookupFailure
	}

	for _, savedUser := range s.savedUsers {
		if savedUser.Email == email {
			return s.userRow(savedUser), nil
		}
	}

	return user.User{}, usererrors.ErrUserNotFound{}
}

func (s *inMemoryUserStore) FindUserByID(_ context.Context, id uuid.UUID) (user.User, error) {
	if s.lookupFailure != nil {
		return user.User{}, s.lookupFailure
	}

	for _, savedUser := range s.savedUsers {
		if savedUser.ID == id {
			return s.userRow(savedUser), nil
		}
	}

	return user.User{}, usererrors.ErrUserNotFound{}
}

func (s *inMemoryUserStore) saveUserForTest(t *testing.T, email string) {
	t.Helper()

	_, err := s.Save(context.Background(), user.NewUser{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: seededHash,
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
}

type inMemoryEventLog struct {
	recordedEvents []events.NewEvent
	failure        error
}

func newInMemoryEventLog() *inMemoryEventLog {
	return &inMemoryEventLog{}
}

func (l *inMemoryEventLog) Save(_ context.Context, newEvent events.NewEvent) (events.Event, error) {
	if l.failure != nil {
		return events.Event{}, l.failure
	}

	l.recordedEvents = append(l.recordedEvents, newEvent)

	return events.Event{
		ID:         newEvent.ID,
		UserID:     newEvent.UserID,
		Label:      newEvent.Label,
		Data:       newEvent.Data,
		OccurredAt: fixedOccurredAt,
	}, nil
}

type inMemoryBroker struct {
	publishedEvents []events.NewEvent
	failure         error
}

func newInMemoryBroker() *inMemoryBroker {
	return &inMemoryBroker{}
}

func (b *inMemoryBroker) publish(_ context.Context, event events.NewEvent) error {
	if b.failure != nil {
		return b.failure
	}

	b.publishedEvents = append(b.publishedEvents, event)

	return nil
}

type inMemoryChallengeGenerator struct {
	requests []challenge.GenerateParams
	failure  error
}

func newInMemoryChallengeGenerator() *inMemoryChallengeGenerator {
	return &inMemoryChallengeGenerator{}
}

func (g *inMemoryChallengeGenerator) generate(
	_ context.Context,
	params challenge.GenerateParams,
) (challenge.Generated, error) {
	if g.failure != nil {
		return challenge.Generated{}, g.failure
	}

	g.requests = append(g.requests, params)

	return challenge.Generated{ID: uuid.New(), ExpiresAt: fixedExpiresAt}, nil
}

type basicSignupForTest struct {
	dependencies SignupUserDependencies
	store        *inMemoryUserStore
	eventLog     *inMemoryEventLog
	broker       *inMemoryBroker
	challenges   *inMemoryChallengeGenerator
}

func newBasicSignupForTest() basicSignupForTest {
	store := newInMemoryUserStore()
	eventLog := newInMemoryEventLog()
	broker := newInMemoryBroker()
	challenges := newInMemoryChallengeGenerator()

	recorder := func(ctx context.Context, event events.NewEvent) error {
		_, err := events.Record(ctx, events.RecordDependencies{EventWriteRepository: eventLog}, event)
		return err
	}

	return basicSignupForTest{
		dependencies: SignupUserDependencies{
			UserWriteRepository:   store,
			EventDispatcher:       events.Dispatcher{Recorder: recorder, Publisher: broker.publish},
			GenerateAuthChallenge: challenges.generate,
		},
		store:      store,
		eventLog:   eventLog,
		broker:     broker,
		challenges: challenges,
	}
}

func silenceLogger(t *testing.T) *bytes.Buffer {
	t.Helper()

	captured := &bytes.Buffer{}
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(captured, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	return captured
}

func TestBasicSignupStoresTheUser(t *testing.T) {
	harness := newBasicSignupForTest()

	result, err := SignupUser(context.Background(), harness.dependencies, SignupUserParams{
		Email:    testEmail,
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("SignupUser: %v", err)
	}

	if len(harness.store.savedUsers) != 1 {
		t.Fatalf("stored rows = %d, want 1", len(harness.store.savedUsers))
	}
	storedRow := harness.store.savedUsers[0]

	if result.User.ID != storedRow.ID {
		t.Errorf("ID = %s, want the stored id %s", result.User.ID, storedRow.ID)
	}
	if result.User.ID == uuid.Nil {
		t.Error("ID is the nil uuid, want a generated one")
	}
	if result.User.Email != testEmail {
		t.Errorf("Email = %q, want %q", result.User.Email, testEmail)
	}
	if !result.User.CreatedAt.Equal(fixedCreatedAt) || !result.User.UpdatedAt.Equal(fixedUpdatedAt) {
		t.Errorf("timestamps = %s / %s, want the ones the store returned %s / %s",
			result.User.CreatedAt, result.User.UpdatedAt, fixedCreatedAt, fixedUpdatedAt)
	}
}

func TestBasicSignupStoresAnArgon2idHashAndNeverThePlaintext(t *testing.T) {
	harness := newBasicSignupForTest()

	_, err := SignupUser(context.Background(), harness.dependencies, SignupUserParams{
		Email:    testEmail,
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("SignupUser: %v", err)
	}

	storedHash := harness.store.savedUsers[0].PasswordHash

	if !strings.HasPrefix(storedHash, "$argon2id$") {
		t.Errorf("stored hash = %q, want the $argon2id$ prefix", storedHash)
	}
	if strings.Contains(storedHash, testPassword) {
		t.Error("the repository received the plaintext password")
	}

	matches, err := password.Verify(password.VerificationParams{
		Password: testPassword,
		Hash:     storedHash,
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !matches {
		t.Error("Verify = false, want the stored hash to verify against the signup password")
	}
}

func TestBasicSignupSurfacesAStorageFailureUnchanged(t *testing.T) {
	cause := errors.New("pq: duplicate key value violates unique constraint")
	harness := newBasicSignupForTest()
	harness.store.failure = usererrors.ErrSaveUserFailed{Cause: cause}

	result, err := SignupUser(context.Background(), harness.dependencies, SignupUserParams{
		Email:    testEmail,
		Password: testPassword,
	})

	saveFailed := usererrors.ErrSaveUserFailed{}
	if !errors.As(err, &saveFailed) {
		t.Fatalf("error = %v, want ErrSaveUserFailed", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the storage failure does not unwrap to the driver error")
	}
	if errors.As(err, &usererrors.ErrPasswordHashingFailed{}) {
		t.Error("a storage failure reports as a hashing failure, want the two kept apart")
	}
	if result.User != (user.User{}) {
		t.Errorf("user = %+v, want the zero value on failure", result.User)
	}
	if len(harness.store.savedUsers) != 0 {
		t.Errorf("stored rows = %d, want nothing stored when the insert fails", len(harness.store.savedUsers))
	}
	if len(harness.eventLog.recordedEvents) != 0 {
		t.Errorf("stored events = %d, want no signup event for a user that was never stored",
			len(harness.eventLog.recordedEvents))
	}
}

func TestBasicSignupRejectsAnEmailThatAlreadyHasAnAccount(t *testing.T) {
	harness := newBasicSignupForTest()
	harness.store.saveUserForTest(t, testEmail)

	result, err := SignupUser(context.Background(), harness.dependencies, SignupUserParams{
		Email:    testEmail,
		Password: testPassword,
	})

	emailTaken := usererrors.ErrEmailTaken{}
	if !errors.As(err, &emailTaken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
	if emailTaken.Email != testEmail {
		t.Errorf("Email = %q, want %q", emailTaken.Email, testEmail)
	}
	if result.User != (user.User{}) {
		t.Errorf("user = %+v, want the zero value on failure", result.User)
	}
	if len(harness.store.savedUsers) != 1 {
		t.Errorf("stored rows = %d, want only the account that was already there",
			len(harness.store.savedUsers))
	}
	if len(harness.eventLog.recordedEvents) != 0 {
		t.Errorf("stored events = %d, want no signup event for a rejected signup",
			len(harness.eventLog.recordedEvents))
	}
}

func TestBasicSignupSurfacesALookupFailureRatherThanATakenEmail(t *testing.T) {
	cause := errors.New("connection refused")
	harness := newBasicSignupForTest()
	harness.store.lookupFailure = usererrors.ErrEmailLookupFailed{Cause: cause}

	result, err := SignupUser(context.Background(), harness.dependencies, SignupUserParams{
		Email:    testEmail,
		Password: testPassword,
	})

	lookupFailed := usererrors.ErrEmailLookupFailed{}
	if !errors.As(err, &lookupFailed) {
		t.Fatalf("error = %v, want ErrEmailLookupFailed", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the lookup failure does not unwrap to the driver error")
	}
	if errors.As(err, &usererrors.ErrEmailTaken{}) {
		t.Error("a lookup failure reports as a taken email, want an honest signup left unblamed")
	}
	if result.User != (user.User{}) {
		t.Errorf("user = %+v, want the zero value on failure", result.User)
	}
	if len(harness.store.savedUsers) != 0 {
		t.Errorf("stored rows = %d, want nothing stored when the lookup fails",
			len(harness.store.savedUsers))
	}
	if len(harness.eventLog.recordedEvents) != 0 {
		t.Errorf("stored events = %d, want no signup event when the lookup fails",
			len(harness.eventLog.recordedEvents))
	}
}

func TestBasicSignupSucceedsWhenTheEventLogRefusesTheEvent(t *testing.T) {
	silenceLogger(t)

	cause := errors.New("pq: relation \"event_log\" does not exist")
	harness := newBasicSignupForTest()
	harness.eventLog.failure = events.ErrRecordEventFailed{Label: events.UserSignedUpLabel, Cause: cause}

	result, err := SignupUser(context.Background(), harness.dependencies, SignupUserParams{
		Email:    testEmail,
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("SignupUser = %v, want a failed event write to leave the signup successful", err)
	}

	if result.User.ID == uuid.Nil {
		t.Error("ID is the nil uuid, want the created user returned despite the event failure")
	}
	if result.User.Email != testEmail {
		t.Errorf("Email = %q, want %q", result.User.Email, testEmail)
	}
	if len(harness.store.savedUsers) != 1 {
		t.Errorf("stored rows = %d, want the user kept when only the event failed",
			len(harness.store.savedUsers))
	}
	if len(harness.eventLog.recordedEvents) != 0 {
		t.Errorf("stored events = %d, want none from a refusing event log",
			len(harness.eventLog.recordedEvents))
	}
}

func TestBasicSignupSucceedsWhenTheChallengeGeneratorFails(t *testing.T) {
	captured := silenceLogger(t)

	harness := newBasicSignupForTest()
	harness.challenges.failure = errors.New("pq: relation \"auth_challenges\" does not exist")

	result, err := SignupUser(context.Background(), harness.dependencies, SignupUserParams{
		Email:    testEmail,
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("SignupUser = %v, want a failed challenge to leave the signup successful", err)
	}

	if result.User.ID == uuid.Nil {
		t.Error("ID is the nil uuid, want the created user returned despite the challenge failure")
	}
	if result.User.Email != testEmail {
		t.Errorf("Email = %q, want %q", result.User.Email, testEmail)
	}
	if len(harness.store.savedUsers) != 1 {
		t.Errorf("stored rows = %d, want the user kept when only the challenge failed",
			len(harness.store.savedUsers))
	}
	if len(harness.challenges.requests) != 0 {
		t.Errorf("challenge requests = %d, want none held by a refusing generator",
			len(harness.challenges.requests))
	}

	logged := captured.String()
	if !strings.Contains(logged, "failed generating the email verification challenge") {
		t.Errorf("logs = %q, want the swallowed challenge failure reported", logged)
	}
	if !strings.Contains(logged, "auth_challenges") {
		t.Errorf("logs = %q, want the underlying cause reported", logged)
	}
}

func TestUserErrorsUnwrapToTheirCause(t *testing.T) {
	cause := errors.New("connection refused")

	if !errors.Is(usererrors.ErrEmailLookupFailed{Cause: cause}, cause) {
		t.Error("errors.Is(ErrEmailLookupFailed{...}, cause) = false, want true")
	}
	if !errors.Is(usererrors.ErrPasswordHashingFailed{Cause: cause}, cause) {
		t.Error("errors.Is(ErrPasswordHashingFailed{...}, cause) = false, want true")
	}
	if !errors.Is(usererrors.ErrSaveUserFailed{Cause: cause}, cause) {
		t.Error("errors.Is(ErrSaveUserFailed{...}, cause) = false, want true")
	}
	if !errors.Is(usererrors.ErrChangeEmailFailed{Cause: cause}, cause) {
		t.Error("errors.Is(ErrChangeEmailFailed{...}, cause) = false, want true")
	}
}
