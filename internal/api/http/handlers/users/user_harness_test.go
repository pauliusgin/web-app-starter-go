package users

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/tokens"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/queries"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

const testEmail = "accountant@example.com"

const testPassword = "correct-horse"

const issuedRefreshTokenForTest = "PSaJ0Vd0kX3xw0kQeCr9nTgqcQ8bJ9YbFq0Q3fF8Zpo"

var errDatabaseDown = errors.New("database unreachable")

var seededAt = time.Date(2026, time.March, 14, 9, 26, 53, 0, time.UTC)

// userStores is the state a user request reads and changes, with each use case wired over it.
type userStores struct {
	users        *repositories.InMemoryUserRepository
	challenges   challengeLinks
	rateLimited  map[string]bool
	profileNames map[uuid.UUID][2]string
	jwt          jwt.AccessTokenIssuer
	google       googleidtoken.InMemoryVerifier
}

func newUserStores(t *testing.T) *userStores {
	t.Helper()

	issuer, err := jwt.NewAccessTokenIssuer(jwt.NewAccessTokenIssuerParams{Secret: "users-handler-test-secret"})
	if err != nil {
		t.Fatalf("building the access token issuer: %v", err)
	}

	return &userStores{
		users:        repositories.NewInMemoryUserRepository(),
		challenges:   challengeLinks{challenge.NewInMemoryChallengeRepository()},
		rateLimited:  map[string]bool{},
		profileNames: map[uuid.UUID][2]string{},
		jwt:          issuer,
		google:       newGoogleVerifierForTest(),
	}
}

type seedUserParams struct {
	email    string
	verified bool
}

// seedUser stores a user whose password is testPassword.
func (s *userStores) seedUser(t *testing.T, params seedUserParams) uuid.UUID {
	t.Helper()

	passwordHash, err := password.Hash(testPassword)
	if err != nil {
		t.Fatalf("hashing the seeded password: %v", err)
	}

	seeded := models.User{ID: uuid.New(), Email: params.email, CreatedAt: seededAt, UpdatedAt: seededAt, SigninAt: seededAt}
	if params.verified {
		seeded.EmailVerifiedAt = seededAt
	}
	s.users.Users[seeded.ID] = repositories.InMemoryUser{User: seeded, PasswordHash: passwordHash}

	return seeded.ID
}

// seedGoogleUser stores a verified, password-less user signed up with the google account googleSubject.
func (s *userStores) seedGoogleUser(googleSubject string) uuid.UUID {
	seeded := models.User{
		ID:              uuid.New(),
		Email:           testEmail,
		CreatedAt:       seededAt,
		UpdatedAt:       seededAt,
		EmailVerifiedAt: seededAt,
		SigninAt:        seededAt,
	}
	s.users.Users[seeded.ID] = repositories.InMemoryUser{User: seeded, GoogleSubject: googleSubject}

	return seeded.ID
}

// assertSignInMethods checks the response lists exactly want under signInMethods, in order.
func assertSignInMethods(t *testing.T, body map[string]any, want ...readmodels.SignInMethod) {
	t.Helper()

	got, isList := body["signInMethods"].([]any)
	if !isList || len(got) != len(want) {
		t.Fatalf("signInMethods = %v, want %v", body["signInMethods"], want)
	}
	for index, method := range want {
		if got[index] != string(method) {
			t.Errorf("signInMethods[%d] = %v, want %q", index, got[index], method)
		}
	}
}

func (s *userStores) storedUser(t *testing.T, userID uuid.UUID) models.User {
	t.Helper()

	stored, found := s.users.Users[userID]
	if !found {
		t.Fatalf("user %s is not stored", userID)
	}

	return stored.User
}

func ignoringDispatcher() events.Dispatcher {
	ignoreEvent := func(context.Context, events.NewEvent) error { return nil }

	return events.Dispatcher{Recorder: ignoreEvent, Publisher: ignoreEvent}
}

func (s *userStores) ensureIssuanceAllowed(_ context.Context, params challenge.EnsureIssuanceAllowedParams) error {
	if s.rateLimited[params.Target] {
		return challenge.ErrIssuanceRateLimited{RetryAfter: 20 * time.Second}
	}

	return nil
}

// getUserReadModelByID projects the stored user as the read query does, joining profileNames
// and deriving the sign-in methods from the stored credentials.
func (s *userStores) getUserReadModelByID(_ context.Context, id uuid.UUID) (readmodels.UserReadModel, error) {
	if s.users.LookupFailure != nil {
		return readmodels.UserReadModel{}, s.users.LookupFailure
	}

	stored, found := s.users.Users[id]
	if !found {
		return readmodels.UserReadModel{}, usererrors.ErrUserNotFound{ID: id}
	}

	readModel := readmodels.UserReadModel{
		ID:              stored.User.ID,
		Email:           stored.User.Email,
		CreatedAt:       stored.User.CreatedAt,
		EmailVerifiedAt: handlers.NullableTime(stored.User.EmailVerifiedAt),
		SigninAt:        handlers.NullableTime(stored.User.SigninAt),
		SignInMethods: readmodels.NewSignInMethods(readmodels.SignInMethodsParams{
			HasPassword:      stored.PasswordHash != "",
			HasGoogleSubject: stored.GoogleSubject != "",
		}),
	}
	if names, named := s.profileNames[id]; named {
		readModel.FirstName = &names[0]
		readModel.LastName = &names[1]
	}

	return readModel, nil
}

// issuedRefreshTokenExpiresAt sits far past the wall clock the cookie is read against.
var issuedRefreshTokenExpiresAt = time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

func issueRefreshTokenForTest(context.Context, refreshtoken.IssueParams) (refreshtoken.IssueResult, error) {
	return refreshtoken.IssueResult{Plaintext: issuedRefreshTokenForTest, ExpiresAt: issuedRefreshTokenExpiresAt}, nil
}

func (s *userStores) signUpHandler() http.Handler {
	return SignUpUser(SignUpUserDependencies{
		SignUpUser: func(ctx context.Context, params usecases.SignUpUserParams) (usecases.SignUpUserResult, error) {
			return usecases.SignUpUser(ctx, usecases.SignUpUserDependencies{
				UserWriteRepository:   s.users,
				EventDispatcher:       ignoringDispatcher(),
				GenerateAuthChallenge: s.challenges.generate,
			}, params)
		},
		GetUserReadModelByID: s.getUserReadModelByID,
		IssueRefreshToken:    issueRefreshTokenForTest,
		Jwt:                  s.jwt,
	})
}

func (s *userStores) signInHandler() http.Handler {
	return SignInUser(SignInUserDependencies{
		SignInUser: func(ctx context.Context, params usecases.SignInUserParams) (usecases.SignInUserResult, error) {
			return usecases.SignInUser(ctx, usecases.SignInUserDependencies{
				UserWriteRepository: s.users,
				EventDispatcher:     ignoringDispatcher(),
			}, params)
		},
		GetUserReadModelByID: s.getUserReadModelByID,
		IssueRefreshToken:    issueRefreshTokenForTest,
		Jwt:                  s.jwt,
	})
}

func (s *userStores) signUpGoogleHandler() http.Handler {
	return SignUpUserGoogle(SignUpUserGoogleDependencies{
		SignUpUserGoogle: func(
			ctx context.Context,
			params usecases.SignUpUserGoogleParams,
		) (usecases.SignUpUserGoogleResult, error) {
			return usecases.SignUpUserGoogle(ctx, usecases.SignUpUserGoogleDependencies{
				UserWriteRepository: s.users,
				EventDispatcher:     ignoringDispatcher(),
				VerifyGoogleIDToken: s.google.VerifyIDToken,
			}, params)
		},
		GetUserReadModelByID: s.getUserReadModelByID,
		IssueRefreshToken:    issueRefreshTokenForTest,
		Jwt:                  s.jwt,
	})
}

func (s *userStores) signInGoogleHandler() http.Handler {
	return SignInUserGoogle(SignInUserGoogleDependencies{
		SignInUserGoogle: func(
			ctx context.Context,
			params usecases.SignInUserGoogleParams,
		) (usecases.SignInUserGoogleResult, error) {
			return usecases.SignInUserGoogle(ctx, usecases.SignInUserGoogleDependencies{
				UserWriteRepository: s.users,
				EventDispatcher:     ignoringDispatcher(),
				VerifyGoogleIDToken: s.google.VerifyIDToken,
			}, params)
		},
		GetUserReadModelByID: s.getUserReadModelByID,
		IssueRefreshToken:    issueRefreshTokenForTest,
		Jwt:                  s.jwt,
	})
}

func (s *userStores) verifyEmailHandler() http.Handler {
	return VerifyEmail(VerifyEmailDependencies{
		VerifyEmail: func(ctx context.Context, params usecases.VerifyEmailParams) error {
			return usecases.VerifyEmail(ctx, usecases.VerifyEmailDependencies{
				UserWriteRepository:          s.users,
				EventDispatcher:              ignoringDispatcher(),
				FindAuthChallengeByTokenHash: s.challenges.FindByTokenHash,
				ConsumeAuthChallenge:         s.challenges.MarkUsed,
			}, params)
		},
	})
}

func (s *userStores) getUserByIDHandler() http.Handler {
	return GetUserByID(GetUserByIDDependencies{
		GetUserByID: func(ctx context.Context, params queries.GetUserByIDParams) (queries.GetUserByIDResult, error) {
			return queries.GetUserByID(ctx, queries.GetUserByIDDependencies{
				UserReadRepository: userReadProjection{stores: s},
			}, params)
		},
	})
}

func (s *userStores) deleteUserHandler() http.Handler {
	return DeleteUser(DeleteUserDependencies{
		DeleteUser: func(ctx context.Context, params usecases.DeleteUserParams) error {
			return usecases.DeleteUser(ctx, usecases.DeleteUserDependencies{
				UserWriteRepository: s.users,
				EventDispatcher:     ignoringDispatcher(),
			}, params)
		},
	})
}

func (s *userStores) requestEmailChangeHandler() http.Handler {
	return RequestEmailChange(RequestEmailChangeDependencies{
		RequestEmailChange: func(ctx context.Context, params usecases.RequestEmailChangeParams) error {
			return usecases.RequestEmailChange(ctx, usecases.RequestEmailChangeDependencies{
				UserWriteRepository:            s.users,
				EnsureChallengeIssuanceAllowed: s.ensureIssuanceAllowed,
				GenerateAuthChallenge:          s.challenges.generate,
			}, params)
		},
	})
}

func (s *userStores) changePasswordHandler() http.Handler {
	return ChangePassword(ChangePasswordDependencies{
		ChangePassword: func(ctx context.Context, params usecases.ChangePasswordParams) error {
			return usecases.ChangePassword(ctx, usecases.ChangePasswordDependencies{
				UserWriteRepository: s.users,
				EventDispatcher:     ignoringDispatcher(),
			}, params)
		},
	})
}

func (s *userStores) addPasswordHandler() http.Handler {
	return AddPassword(AddPasswordDependencies{
		AddPassword: func(ctx context.Context, params usecases.AddPasswordParams) error {
			return usecases.AddPassword(ctx, usecases.AddPasswordDependencies{
				UserWriteRepository: s.users,
				EventDispatcher:     ignoringDispatcher(),
			}, params)
		},
	})
}

func (s *userStores) addGoogleAccountHandler() http.Handler {
	return AddGoogleAccount(AddGoogleAccountDependencies{
		AddGoogleAccount: func(ctx context.Context, params usecases.AddGoogleAccountParams) error {
			return usecases.AddGoogleAccount(ctx, usecases.AddGoogleAccountDependencies{
				UserWriteRepository: s.users,
				EventDispatcher:     ignoringDispatcher(),
				VerifyGoogleIDToken: s.google.VerifyIDToken,
			}, params)
		},
	})
}

type userReadProjection struct {
	stores *userStores
}

func (p userReadProjection) GetUserReadModelByID(ctx context.Context, id uuid.UUID) (readmodels.UserReadModel, error) {
	return p.stores.getUserReadModelByID(ctx, id)
}

// challengeLinks runs the real challenge.Generate over an in-memory auth_challenges store.
type challengeLinks struct {
	*challenge.InMemoryChallengeRepository
}

// liveLinkExpiresAt sits far past the wall clock the repository checks seeded links against.
var liveLinkExpiresAt = time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

func (l challengeLinks) generate(ctx context.Context, params challenge.GenerateParams) (challenge.GenerateResult, error) {
	return challenge.Generate(ctx, challenge.GenerateDependencies{
		WriteRepository: l.InMemoryChallengeRepository,
		EventDispatcher: (&eventstest.CapturedEvents{}).Dispatcher(),
	}, params)
}

// seedLink stores a live link proving target for the user, opened by presenting plaintextToken.
func (l challengeLinks) seedLink(userID uuid.UUID, target string, plaintextToken string) {
	l.Challenges = append(l.Challenges, challenge.Challenge{
		ID:         uuid.New(),
		UserID:     userID,
		TargetType: challenge.TargetTypeEmail,
		Target:     target,
		Purpose:    challenge.PurposeEmailVerification,
		TokenHash:  token.Hash(plaintextToken),
		ExpiresAt:  liveLinkExpiresAt,
	})
}

func (l challengeLinks) linksSentTo(target string) int {
	sent := 0
	for _, stored := range l.Challenges {
		if stored.Target == target {
			sent++
		}
	}

	return sent
}

// authenticateForTest stands in for the access token check, admitting every request as userID.
func authenticateForTest(userID uuid.UUID) func(http.Handler) http.Handler {
	return middleware.RequireAuthenticatedUser(middleware.RequireAuthenticatedUserDependencies{
		ParseAccessToken: func(string) (jwt.AccessTokenClaims, error) {
			return jwt.AccessTokenClaims{UserID: userID}, nil
		},
		FindUserByID: func(_ context.Context, id uuid.UUID) (models.User, error) {
			return models.User{ID: id, Email: testEmail}, nil
		},
	})
}

type sendUserRequestParams struct {
	handler http.Handler
	method  string
	path    string
	body    string
	// asUserID authenticates the request; uuid.Nil sends it past no guard, as an unprotected route would.
	asUserID uuid.UUID
}

func sendUserRequest(t *testing.T, params sendUserRequestParams) *http.Response {
	t.Helper()

	served := params.handler
	if params.asUserID != uuid.Nil {
		served = authenticateForTest(params.asUserID)(served)
	}

	var body io.Reader = http.NoBody
	if params.body != "" {
		body = bytes.NewBufferString(params.body)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(params.method, params.path, body)
	request.Header.Set("Authorization", "Bearer header.payload.signature")
	request.Header.Set("Content-Type", "application/json")

	served.ServeHTTP(recorder, request)

	response := recorder.Result()
	t.Cleanup(func() { response.Body.Close() })

	return response
}

func decodeBody(t *testing.T, response *http.Response) map[string]any {
	t.Helper()

	body := map[string]any{}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the response body: %v", err)
	}

	return body
}

func assertErrorCode(t *testing.T, response *http.Response, wantStatus int, wantCode handlers.ErrorCode) {
	t.Helper()

	if response.StatusCode != wantStatus {
		t.Fatalf("status = %d, want %d", response.StatusCode, wantStatus)
	}
	body := decodeBody(t, response)
	if body["code"] != string(wantCode) {
		t.Errorf(`body["code"] = %v, want %q`, body["code"], wantCode)
	}
	if message, _ := body["message"].(string); message == "" {
		t.Error(`body["message"] is empty, want a message`)
	}
}

func assertRefreshTokenCookieSet(t *testing.T, response *http.Response) {
	t.Helper()

	for _, cookie := range response.Cookies() {
		if cookie.Name != tokens.RefreshTokenCookieName {
			continue
		}
		if cookie.Value != issuedRefreshTokenForTest {
			t.Errorf("refresh token cookie = %q, want %q", cookie.Value, issuedRefreshTokenForTest)
		}
		if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
			t.Errorf("refresh token cookie flags = %+v, want HttpOnly, Secure, SameSite=Strict", cookie)
		}
		if cookie.Path != tokens.RefreshTokenCookiePath {
			t.Errorf("refresh token cookie path = %q, want %q", cookie.Path, tokens.RefreshTokenCookiePath)
		}
		return
	}
	t.Error("response sets no refresh token cookie")
}

func assertNoRefreshTokenCookie(t *testing.T, response *http.Response) {
	t.Helper()

	for _, cookie := range response.Cookies() {
		if cookie.Name == tokens.RefreshTokenCookieName {
			t.Errorf("refresh token cookie = %q, want none on a rejected request", cookie.Value)
		}
	}
}

func silenceHandlerLogger(t *testing.T) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}
