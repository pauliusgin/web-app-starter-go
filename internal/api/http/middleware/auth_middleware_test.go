package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	models "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
)

const accessTokenForTest = "header.payload.signature"

type recordingAccessTokenParser struct {
	claims        jwt.AccessTokenClaims
	failure       error
	receivedCalls []string
}

func (p *recordingAccessTokenParser) parseAccessToken(accessToken string) (jwt.AccessTokenClaims, error) {
	p.receivedCalls = append(p.receivedCalls, accessToken)
	if p.failure != nil {
		return jwt.AccessTokenClaims{}, p.failure
	}

	return p.claims, nil
}

type recordingUserFinder struct {
	found         models.User
	failure       error
	receivedCalls []uuid.UUID
}

func (f *recordingUserFinder) findUserByID(_ context.Context, id uuid.UUID) (models.User, error) {
	f.receivedCalls = append(f.receivedCalls, id)
	if f.failure != nil {
		return models.User{}, f.failure
	}

	return f.found, nil
}

type recordingNextHandler struct {
	ran        bool
	seenUserID uuid.UUID
	seenUserOK bool
}

func (h *recordingNextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.ran = true
	h.seenUserID, h.seenUserOK = UserIDFromContext(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

type capturedLogEntry struct {
	level     slog.Level
	loggedErr error
}

// capturingLogHandler keeps log records in memory. The middleware's typed errors reach only
// the log, never the response, so the log is where a test observes which failure occurred.
type capturingLogHandler struct {
	entries []capturedLogEntry
}

func (h *capturingLogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *capturingLogHandler) Handle(_ context.Context, record slog.Record) error {
	entry := capturedLogEntry{level: record.Level}
	record.Attrs(func(attr slog.Attr) bool {
		if loggedErr, isError := attr.Value.Any().(error); isError {
			entry.loggedErr = loggedErr
		}

		return true
	})
	h.entries = append(h.entries, entry)

	return nil
}

func (h *capturingLogHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *capturingLogHandler) WithGroup(string) slog.Handler {
	return h
}

func (h *capturingLogHandler) singleEntry(t *testing.T) capturedLogEntry {
	t.Helper()

	if len(h.entries) != 1 {
		t.Fatalf("logged %d entries, want exactly 1", len(h.entries))
	}

	return h.entries[0]
}

func captureHandlerLogs(t *testing.T) *capturingLogHandler {
	t.Helper()

	logs := &capturingLogHandler{}
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(logs))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	return logs
}

func loggedErrorAs[T error](t *testing.T, entry capturedLogEntry) T {
	t.Helper()

	var target T
	if !errors.As(entry.loggedErr, &target) {
		t.Fatalf("logged error = %v (%T), want a %T in its chain", entry.loggedErr, entry.loggedErr, target)
	}

	return target
}

func authenticatedUserFixture() models.User {
	createdAt := time.Date(2026, 3, 14, 9, 26, 53, 0, time.UTC)

	return models.User{
		ID:        uuid.MustParse("6f1c9f2e-4a1b-4c3d-9e8f-0a1b2c3d4e5f"),
		Email:     "accountant@example.com",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func claimsFor(authenticated models.User) jwt.AccessTokenClaims {
	return jwt.AccessTokenClaims{
		UserID:    authenticated.ID,
		Email:     authenticated.Email,
		IssuedAt:  authenticated.CreatedAt,
		ExpiresAt: authenticated.CreatedAt.Add(jwt.AccessTokenLifetime),
	}
}

type sendGuardedRequestParams struct {
	parser        *recordingAccessTokenParser
	users         *recordingUserFinder
	authorization string
	omitHeader    bool
}

func sendGuardedRequest(t *testing.T, params sendGuardedRequestParams) (*httptest.ResponseRecorder, *recordingNextHandler) {
	t.Helper()

	next := &recordingNextHandler{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	if !params.omitHeader {
		request.Header.Set("Authorization", params.authorization)
	}

	dependencies := RequireAuthenticatedUserDependencies{
		ParseAccessToken: params.parser.parseAccessToken,
		FindUserByID:     params.users.findUserByID,
	}
	RequireAuthenticatedUser(dependencies)(next).ServeHTTP(recorder, request)

	return recorder, next
}

func decodeErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder) handlers.ErrorResponse {
	t.Helper()

	body := handlers.ErrorResponse{}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the response body: %v", err)
	}

	return body
}

func assertRejection(t *testing.T, recorder *httptest.ResponseRecorder, wantChallenge string, wantCode handlers.ErrorCode) {
	t.Helper()

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if challenge := recorder.Header().Get("WWW-Authenticate"); challenge != wantChallenge {
		t.Errorf("WWW-Authenticate = %q, want %q", challenge, wantChallenge)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}
	body := decodeErrorResponse(t, recorder)
	if body.Code != wantCode {
		t.Errorf("code = %q, want %q", body.Code, wantCode)
	}
	if body.Message == "" {
		t.Error("message is empty, want a message")
	}
}

func TestRequireAuthenticatedUserAdmitsATokenNamingALiveUser(t *testing.T) {
	cases := []struct {
		name   string
		scheme string
	}{
		{name: "canonical scheme", scheme: "Bearer"},
		{name: "lowercase scheme", scheme: "bearer"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			logs := captureHandlerLogs(t)

			authenticated := authenticatedUserFixture()
			parser := &recordingAccessTokenParser{claims: claimsFor(authenticated)}
			users := &recordingUserFinder{found: authenticated}

			recorder, next := sendGuardedRequest(t, sendGuardedRequestParams{
				parser:        parser,
				users:         users,
				authorization: testCase.scheme + " " + accessTokenForTest,
			})

			if !next.ran {
				t.Fatalf("the next handler did not run, want it to run for a valid token; status = %d", recorder.Code)
			}
			if !next.seenUserOK || next.seenUserID != authenticated.ID {
				t.Errorf("UserIDFromContext = (%s, %t), want (%s, true)", next.seenUserID, next.seenUserOK, authenticated.ID)
			}
			if recorder.Code != http.StatusNoContent {
				t.Errorf("status = %d, want %d: the middleware must write no status of its own",
					recorder.Code, http.StatusNoContent)
			}
			if challenge := recorder.Header().Get("WWW-Authenticate"); challenge != "" {
				t.Errorf("WWW-Authenticate = %q, want it absent on an admitted request", challenge)
			}
			if len(parser.receivedCalls) != 1 || parser.receivedCalls[0] != accessTokenForTest {
				t.Errorf("ParseAccessToken received %q, want exactly [%q]", parser.receivedCalls, accessTokenForTest)
			}
			if len(users.receivedCalls) != 1 || users.receivedCalls[0] != authenticated.ID {
				t.Errorf("FindUserByID received %v, want exactly [%s]", users.receivedCalls, authenticated.ID)
			}
			if len(logs.entries) != 0 {
				t.Errorf("logged %d entries, want none for an admitted request", len(logs.entries))
			}
		})
	}
}

func TestRequireAuthenticatedUserRejectsRequestsWithoutAUsableBearerToken(t *testing.T) {
	cases := []struct {
		name            string
		authorization   string
		omitHeader      bool
		wantChallenge   string
		wantCode        handlers.ErrorCode
		wantLoggedError error
	}{
		{
			name:            "no header",
			omitHeader:      true,
			wantChallenge:   tokenRequiredChallenge,
			wantCode:        handlers.ErrorCodeAccessTokenMissing,
			wantLoggedError: ErrAuthorizationHeaderAbsent{},
		},
		{
			name:            "wrong scheme",
			authorization:   "Basic abc",
			wantChallenge:   invalidTokenChallenge,
			wantCode:        handlers.ErrorCodeAccessTokenInvalid,
			wantLoggedError: ErrUnsupportedAuthorizationScheme{Scheme: "Basic"},
		},
		{
			name:            "wrong scheme of exactly six characters is logged whole",
			authorization:   "Digest abc",
			wantChallenge:   invalidTokenChallenge,
			wantCode:        handlers.ErrorCodeAccessTokenInvalid,
			wantLoggedError: ErrUnsupportedAuthorizationScheme{Scheme: "Digest"},
		},
		{
			name:            "wrong scheme longer than six characters is truncated",
			authorization:   "Negotiate abc",
			wantChallenge:   invalidTokenChallenge,
			wantCode:        handlers.ErrorCodeAccessTokenInvalid,
			wantLoggedError: ErrUnsupportedAuthorizationScheme{Scheme: "Negoti..."},
		},
		{
			name:            "token without a scheme is truncated",
			authorization:   accessTokenForTest,
			wantChallenge:   invalidTokenChallenge,
			wantCode:        handlers.ErrorCodeAccessTokenInvalid,
			wantLoggedError: ErrUnsupportedAuthorizationScheme{Scheme: "header..."},
		},
		{
			name:            "malformed: scheme only",
			authorization:   "Bearer",
			wantChallenge:   invalidTokenChallenge,
			wantCode:        handlers.ErrorCodeAccessTokenInvalid,
			wantLoggedError: ErrBearerTokenMalformed{},
		},
		{
			name:            "malformed: double space before the token",
			authorization:   "Bearer  " + accessTokenForTest,
			wantChallenge:   invalidTokenChallenge,
			wantCode:        handlers.ErrorCodeAccessTokenInvalid,
			wantLoggedError: ErrBearerTokenMalformed{},
		},
		{
			name:            "malformed: tab inside the token",
			authorization:   "Bearer header.payload\t.signature",
			wantChallenge:   invalidTokenChallenge,
			wantCode:        handlers.ErrorCodeAccessTokenInvalid,
			wantLoggedError: ErrBearerTokenMalformed{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			logs := captureHandlerLogs(t)

			authenticated := authenticatedUserFixture()
			parser := &recordingAccessTokenParser{claims: claimsFor(authenticated)}
			users := &recordingUserFinder{found: authenticated}

			recorder, next := sendGuardedRequest(t, sendGuardedRequestParams{
				parser:        parser,
				users:         users,
				authorization: testCase.authorization,
				omitHeader:    testCase.omitHeader,
			})

			assertRejection(t, recorder, testCase.wantChallenge, testCase.wantCode)
			if next.ran {
				t.Error("the next handler ran, want it skipped")
			}
			if len(parser.receivedCalls) != 0 {
				t.Errorf("ParseAccessToken received %q, want nothing: the header never yielded a token",
					parser.receivedCalls)
			}
			if len(users.receivedCalls) != 0 {
				t.Errorf("FindUserByID was called %d times, want 0", len(users.receivedCalls))
			}

			entry := logs.singleEntry(t)
			if entry.level != slog.LevelWarn {
				t.Errorf("log level = %s, want %s", entry.level, slog.LevelWarn)
			}
			if !errors.Is(entry.loggedErr, testCase.wantLoggedError) {
				t.Errorf("logged error = %#v, want %#v", entry.loggedErr, testCase.wantLoggedError)
			}
			if strings.Contains(entry.loggedErr.Error(), accessTokenForTest) {
				t.Errorf("logged error = %q, want no trace of the access token", entry.loggedErr)
			}
		})
	}
}

func TestRequireAuthenticatedUserRejectsATokenTheParserRefuses(t *testing.T) {
	leakedDetail := "signature is invalid"

	cases := []struct {
		name          string
		failure       error
		wantChallenge string
		wantCode      handlers.ErrorCode
	}{
		{
			name:          "expired",
			failure:       jwt.ErrTokenExpired{ExpiresAt: time.Date(2026, 3, 14, 9, 26, 53, 0, time.UTC)},
			wantChallenge: expiredTokenChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenExpired,
		},
		{
			name:          "unparsable",
			failure:       jwt.ErrTokenUnparsable{Cause: errors.New(leakedDetail)},
			wantChallenge: invalidTokenChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenInvalid,
		},
		{
			name:          "unusable claim",
			failure:       jwt.ErrClaimUnusable{Claim: "sub", Cause: errors.New(leakedDetail)},
			wantChallenge: invalidTokenChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenInvalid,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			logs := captureHandlerLogs(t)

			parser := &recordingAccessTokenParser{failure: testCase.failure}
			users := &recordingUserFinder{found: authenticatedUserFixture()}

			recorder, next := sendGuardedRequest(t, sendGuardedRequestParams{
				parser:        parser,
				users:         users,
				authorization: "Bearer " + accessTokenForTest,
			})

			assertRejection(t, recorder, testCase.wantChallenge, testCase.wantCode)
			if next.ran {
				t.Error("the next handler ran, want it skipped")
			}
			if len(users.receivedCalls) != 0 {
				t.Errorf("FindUserByID was called %d times, want 0 for a token that never parsed",
					len(users.receivedCalls))
			}

			entry := logs.singleEntry(t)
			if entry.level != slog.LevelWarn {
				t.Errorf("log level = %s, want %s", entry.level, slog.LevelWarn)
			}
			loggedErrorAs[ErrAccessTokenRejected](t, entry)
			if !errors.Is(entry.loggedErr, testCase.failure) {
				t.Errorf("logged error = %v, want it to wrap the parser's %v", entry.loggedErr, testCase.failure)
			}
		})
	}
}

// A deleted user and a forged token must be indistinguishable on the wire: same status,
// same challenge, same body as the "unparsable" case above. Otherwise anyone holding an old
// token can ask the API whether that account still exists.
func TestRequireAuthenticatedUserRejectsATokenNamingADeletedUser(t *testing.T) {
	logs := captureHandlerLogs(t)

	authenticated := authenticatedUserFixture()
	lookupFailure := usererrors.ErrUserNotFound{}
	users := &recordingUserFinder{failure: lookupFailure}

	recorder, next := sendGuardedRequest(t, sendGuardedRequestParams{
		parser:        &recordingAccessTokenParser{claims: claimsFor(authenticated)},
		users:         users,
		authorization: "Bearer " + accessTokenForTest,
	})

	assertRejection(t, recorder, invalidTokenChallenge, handlers.ErrorCodeAccessTokenInvalid)
	if next.ran {
		t.Error("the next handler ran, want it skipped for a user that no longer exists")
	}

	entry := logs.singleEntry(t)
	if entry.level != slog.LevelWarn {
		t.Errorf("log level = %s, want %s", entry.level, slog.LevelWarn)
	}
	userNotFound := loggedErrorAs[ErrAuthenticatedUserNotFound](t, entry)
	if userNotFound.UserID != authenticated.ID {
		t.Errorf("ErrAuthenticatedUserNotFound.UserID = %s, want %s", userNotFound.UserID, authenticated.ID)
	}
	if !errors.Is(entry.loggedErr, lookupFailure) {
		t.Errorf("logged error = %v, want it to wrap the lookup's %v", entry.loggedErr, lookupFailure)
	}
}

func TestRequireAuthenticatedUserAnswersInternalServerErrorWhenTheLookupFails(t *testing.T) {
	logs := captureHandlerLogs(t)

	authenticated := authenticatedUserFixture()
	leakedDetail := "dial tcp 127.0.0.1:5432: connect: connection refused"
	lookupFailure := usererrors.ErrIDLookupFailed{Cause: errors.New(leakedDetail)}

	recorder, next := sendGuardedRequest(t, sendGuardedRequestParams{
		parser:        &recordingAccessTokenParser{claims: claimsFor(authenticated)},
		users:         &recordingUserFinder{failure: lookupFailure},
		authorization: "Bearer " + accessTokenForTest,
	})

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d: a database outage is not a verdict on the token",
			recorder.Code, http.StatusInternalServerError)
	}
	if challenge := recorder.Header().Get("WWW-Authenticate"); challenge != "" {
		t.Errorf("WWW-Authenticate = %q, want it absent: a 500 issues no challenge", challenge)
	}
	if next.ran {
		t.Error("the next handler ran, want it skipped")
	}

	body := decodeErrorResponse(t, recorder)
	if body.Code != handlers.ErrorCodeInternal {
		t.Errorf("code = %q, want %q", body.Code, handlers.ErrorCodeInternal)
	}
	if body.Message == "" || strings.Contains(body.Message, "5432") {
		t.Errorf("message = %q, want a generic server-failure message with no trace of the underlying failure",
			body.Message)
	}

	entry := logs.singleEntry(t)
	if entry.level != slog.LevelError {
		t.Errorf("log level = %s, want %s", entry.level, slog.LevelError)
	}
	if !errors.Is(entry.loggedErr, lookupFailure) {
		t.Errorf("logged error = %v, want the lookup's %v", entry.loggedErr, lookupFailure)
	}
	if errors.As(entry.loggedErr, &ErrAuthenticatedUserNotFound{}) {
		t.Error("logged error is ErrAuthenticatedUserNotFound, want an outage kept apart from a deleted user")
	}
}

func TestUserIDFromContextReportsNoUserOnAPlainContext(t *testing.T) {
	userID, ok := UserIDFromContext(context.Background())

	if ok {
		t.Error("ok = true for a context the middleware never touched, want false")
	}
	if userID != uuid.Nil {
		t.Errorf("userID = %s, want the nil uuid", userID)
	}
}
