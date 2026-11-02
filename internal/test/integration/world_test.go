package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	authrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/auth/repositories"
	eventsrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/events/repositories"
	profilesrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/profiles/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	uploadsrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/uploads/repositories"
	usersrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/users/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	profileusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/imaging"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	uploadmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
	uploadusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/usecases"
	userqueries "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/queries"
	userusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	authevents "github.com/pauliusgin/web_app_starter-be/internal/events/auth"
)

const testPassword = "correct-horse-battery"

var jpegContent = []byte("\xFF\xD8\xFF\xE0 jpeg bytes")

// integrationWorld is cmd/server's wiring cut down to Postgres: brokers, email and storage stay in memory.
type integrationWorld struct {
	database  *sql.DB
	queries   *sqlc.Queries
	published *publishedEvents
	files     *storage.InMemoryFileStorage
	ctx       context.Context

	userWriteRepository   usersrepo.UserWriteRepository
	userReadRepository    usersrepo.UserReadRepository
	uploadRepository      uploadsrepo.UploadWriteRepository
	challengeReadRepo     authrepo.AuthChallengeReadRepository
	refreshTokenReadRepo  authrepo.RefreshTokenReadRepository
	refreshTokenWriteRepo authrepo.RefreshTokenWriteRepository
	eventDispatcher       events.Dispatcher
	ensureIssuanceAllowed func(ctx context.Context, params challenge.EnsureIssuanceAllowedParams) error
	generateAuthChallenge func(ctx context.Context, params challenge.GenerateParams) (challenge.GenerateResult, error)
	profileRepository     profilesrepo.ProfileWriteRepository
	profileReadRepository profilesrepo.ProfileReadRepository
}

func newIntegrationWorld(t *testing.T) *integrationWorld {
	t.Helper()

	database := testPostgres.NewTestDatabase(t)
	queries := sqlc.New(database)
	published := &publishedEvents{}

	recordEventDependencies := events.RecordDependencies{
		EventWriteRepository: eventsrepo.NewEventWriteRepository(queries),
	}
	eventDispatcher := events.Dispatcher{
		Recorder: func(ctx context.Context, event events.NewEvent) error {
			_, err := events.Record(ctx, recordEventDependencies, event)
			return err
		},
		Publisher: published.publish,
	}

	generateDependencies := challenge.GenerateDependencies{
		WriteRepository: authrepo.NewAuthChallengeWriteRepository(queries),
		EventDispatcher: eventDispatcher,
	}
	issuanceDependencies := challenge.EnsureIssuanceAllowedDependencies{
		ReadRepository: authrepo.NewAuthChallengeReadRepository(queries),
		Now:            time.Now,
	}

	return &integrationWorld{
		database:              database,
		queries:               queries,
		published:             published,
		files:                 storage.NewInMemoryFileStorage(),
		ctx:                   context.Background(),
		userWriteRepository:   usersrepo.NewUserWriteRepository(queries),
		userReadRepository:    usersrepo.NewUserReadRepository(queries),
		uploadRepository:      uploadsrepo.NewUploadWriteRepository(queries),
		challengeReadRepo:     authrepo.NewAuthChallengeReadRepository(queries),
		refreshTokenReadRepo:  authrepo.NewRefreshTokenReadRepository(queries),
		refreshTokenWriteRepo: authrepo.NewRefreshTokenWriteRepository(queries),
		profileRepository:     profilesrepo.NewProfileWriteRepository(queries),
		profileReadRepository: profilesrepo.NewProfileReadRepository(queries),
		eventDispatcher:       eventDispatcher,
		ensureIssuanceAllowed: func(ctx context.Context, params challenge.EnsureIssuanceAllowedParams) error {
			return challenge.EnsureIssuanceAllowed(ctx, issuanceDependencies, params)
		},
		generateAuthChallenge: func(ctx context.Context, params challenge.GenerateParams) (challenge.GenerateResult, error) {
			return challenge.Generate(ctx, generateDependencies, params)
		},
	}
}

// publishedEvents stands in for RabbitMQ; tests read challenge tokens out of it as subscribers would.
type publishedEvents struct {
	lock   sync.Mutex
	events []events.NewEvent
}

func (p *publishedEvents) publish(_ context.Context, event events.NewEvent) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.events = append(p.events, event)

	return nil
}

func (p *publishedEvents) labelled(label events.Label) []events.NewEvent {
	p.lock.Lock()
	defer p.lock.Unlock()

	var matching []events.NewEvent
	for _, event := range p.events {
		if event.Label == label {
			matching = append(matching, event)
		}
	}

	return matching
}

// latestChallengeToken is the plaintext token the verification email would carry to target.
func (w *integrationWorld) latestChallengeToken(t *testing.T, target string) string {
	t.Helper()

	generated := w.published.labelled(events.AuthChallengeGeneratedLabel)
	for index := len(generated) - 1; index >= 0; index-- {
		payload := authevents.AuthChallengeGeneratedData{}
		if err := json.Unmarshal(generated[index].Data, &payload); err != nil {
			t.Fatalf("decoding the challenge event: %v", err)
		}
		if payload.Target == target {
			return payload.Token
		}
	}

	t.Fatalf("no challenge was published for %q", target)

	return ""
}

func (w *integrationWorld) signUp(t *testing.T, email string) userusecases.SignUpUserResult {
	t.Helper()

	signedUp, err := userusecases.SignUpUser(w.ctx, userusecases.SignUpUserDependencies{
		UserWriteRepository:   w.userWriteRepository,
		EventDispatcher:       w.eventDispatcher,
		GenerateAuthChallenge: w.generateAuthChallenge,
	}, userusecases.SignUpUserParams{Email: email, Password: testPassword})
	if err != nil {
		t.Fatalf("SignUpUser(%q): %v", email, err)
	}

	return signedUp
}

const integrationGoogleIDToken = "integration-google-id-token"

const integrationGoogleSubject = "109876543210987654321"

// googleVerifier vouches for integrationGoogleIDToken as a verified google account holding email.
func googleVerifier(email string) googleidtoken.InMemoryVerifier {
	return googleidtoken.InMemoryVerifier{Identities: map[string]googleidtoken.Identity{
		integrationGoogleIDToken: {Subject: integrationGoogleSubject, Email: email, EmailVerified: true},
	}}
}

func (w *integrationWorld) signUpGoogle(email string) (userusecases.SignUpUserGoogleResult, error) {
	return userusecases.SignUpUserGoogle(w.ctx, userusecases.SignUpUserGoogleDependencies{
		UserWriteRepository: w.userWriteRepository,
		EventDispatcher:     w.eventDispatcher,
		VerifyGoogleIDToken: googleVerifier(email).VerifyIDToken,
	}, userusecases.SignUpUserGoogleParams{IDToken: integrationGoogleIDToken})
}

func (w *integrationWorld) signInGoogle(email string) (userusecases.SignInUserGoogleResult, error) {
	return userusecases.SignInUserGoogle(w.ctx, userusecases.SignInUserGoogleDependencies{
		UserWriteRepository: w.userWriteRepository,
		EventDispatcher:     w.eventDispatcher,
		VerifyGoogleIDToken: googleVerifier(email).VerifyIDToken,
	}, userusecases.SignInUserGoogleParams{IDToken: integrationGoogleIDToken})
}

func (w *integrationWorld) verifyEmail(params userusecases.VerifyEmailParams) error {
	return userusecases.VerifyEmail(w.ctx, userusecases.VerifyEmailDependencies{
		UserWriteRepository:          w.userWriteRepository,
		EventDispatcher:              w.eventDispatcher,
		FindAuthChallengeByTokenHash: w.challengeReadRepo.FindByTokenHash,
		ConsumeAuthChallenge:         authrepo.NewAuthChallengeWriteRepository(w.queries).MarkUsed,
	}, params)
}

// verifiedUserID signs a user up and clicks the link from their verification email.
func (w *integrationWorld) verifiedUserID(t *testing.T) uuid.UUID {
	t.Helper()

	email := fmt.Sprintf("member-%s@example.com", uuid.NewString()[:8])
	signedUp := w.signUp(t, email)

	err := w.verifyEmail(userusecases.VerifyEmailParams{
		VerificationToken: w.latestChallengeToken(t, email),
		UserID:            signedUp.User.ID,
	})
	if err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	return signedUp.User.ID
}

func (w *integrationWorld) signIn(params userusecases.SignInUserParams) (userusecases.SignInUserResult, error) {
	return userusecases.SignInUser(w.ctx, userusecases.SignInUserDependencies{
		UserWriteRepository: w.userWriteRepository,
		EventDispatcher:     w.eventDispatcher,
	}, params)
}

func (w *integrationWorld) addPassword(params userusecases.AddPasswordParams) error {
	return userusecases.AddPassword(w.ctx, userusecases.AddPasswordDependencies{
		UserWriteRepository: w.userWriteRepository,
		EventDispatcher:     w.eventDispatcher,
	}, params)
}

func (w *integrationWorld) requestEmailChange(params userusecases.RequestEmailChangeParams) error {
	return userusecases.RequestEmailChange(w.ctx, userusecases.RequestEmailChangeDependencies{
		UserWriteRepository:            w.userWriteRepository,
		EnsureChallengeIssuanceAllowed: w.ensureIssuanceAllowed,
		GenerateAuthChallenge:          w.generateAuthChallenge,
	}, params)
}

func (w *integrationWorld) resendEmailVerification(params userusecases.ResendEmailVerificationParams) error {
	return userusecases.ResendEmailVerification(w.ctx, userusecases.ResendEmailVerificationDependencies{
		UserWriteRepository:            w.userWriteRepository,
		EnsureChallengeIssuanceAllowed: w.ensureIssuanceAllowed,
		GenerateAuthChallenge:          w.generateAuthChallenge,
	}, params)
}

func (w *integrationWorld) deleteUser(params userusecases.DeleteUserParams) error {
	return userusecases.DeleteUser(w.ctx, userusecases.DeleteUserDependencies{
		UserWriteRepository: w.userWriteRepository,
		EventDispatcher:     w.eventDispatcher,
	}, params)
}

func (w *integrationWorld) getUserByID(t *testing.T, userID uuid.UUID) userqueries.GetUserByIDResult {
	t.Helper()

	found, err := userqueries.GetUserByID(w.ctx, userqueries.GetUserByIDDependencies{
		UserReadRepository: w.userReadRepository,
	}, userqueries.GetUserByIDParams{UserID: userID})
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}

	return found
}

func (w *integrationWorld) createProfile(
	params profileusecases.CreateProfileParams,
) (profileusecases.CreateProfileResult, error) {
	return profileusecases.CreateProfile(w.ctx, profileusecases.CreateProfileDependencies{
		ProfileWriteRepository: w.profileRepository,
		EventDispatcher:        w.eventDispatcher,
		AttachUpload:           w.uploadRepository.AttachUpload,
	}, params)
}

func (w *integrationWorld) updateProfile(
	params profileusecases.UpdateProfileParams,
) (profileusecases.UpdateProfileResult, error) {
	return profileusecases.UpdateProfile(w.ctx, profileusecases.UpdateProfileDependencies{
		ProfileWriteRepository: w.profileRepository,
		EventDispatcher:        w.eventDispatcher,
		AttachUpload:           w.uploadRepository.AttachUpload,
	}, params)
}

func (w *integrationWorld) uploadImage(t *testing.T, userID uuid.UUID, purpose uploadmodels.Purpose) uploadmodels.Upload {
	t.Helper()

	uploaded, err := uploadusecases.UploadFiles(w.ctx, uploadusecases.UploadFilesDependencies{
		UploadWriteRepository: w.uploadRepository,
		FileStorage:           w.files,
		ImageProcessor:        imaging.NewInMemoryImageProcessor(imaging.ImageInfo{Width: 400, Height: 300}),
		EventDispatcher:       w.eventDispatcher,
		AppEnv:                "test",
	}, uploadusecases.UploadFilesParams{
		UserID:  userID,
		Purpose: purpose,
		Files:   []uploadusecases.IncomingFile{{Name: "picture.jpg", Content: jpegContent}},
	})
	if err != nil {
		t.Fatalf("UploadFiles: %v", err)
	}

	return uploaded.Uploads[0]
}

func (w *integrationWorld) issueRefreshToken(t *testing.T, userID uuid.UUID) refreshtoken.IssueResult {
	t.Helper()

	issued, err := refreshtoken.Issue(w.ctx, refreshtoken.IssueDependencies{
		WriteRepository: w.refreshTokenWriteRepo,
	}, refreshtoken.IssueParams{UserID: userID})
	if err != nil {
		t.Fatalf("Issue refresh token: %v", err)
	}

	return issued
}
