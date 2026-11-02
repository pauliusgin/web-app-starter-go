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
	usermodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
)

const accessTokenForTest = "header.payload.signature"

// stubAccessTokenParser parses only accessTokenForTest, so an admitted request proves the header's
// token is what reached the parser.
type stubAccessTokenParser struct {
	claims  jwt.AccessTokenClaims
	failure error
}

func (p *stubAccessTokenParser) parseAccessToken(accessToken string) (jwt.AccessTokenClaims, error) {
	if p.failure != nil {
		return jwt.AccessTokenClaims{}, p.failure
	}
	if accessToken != accessTokenForTest {
		return jwt.AccessTokenClaims{}, jwt.ErrTokenUnparsable{Cause: errors.New("not the test token")}
	}

	return p.claims, nil
}

// stubUserFinder knows only the found user, so an admitted request proves the claimed id was looked up.
type stubUserFinder struct {
	found   usermodels.User
	failure error
}

func (f *stubUserFinder) findUserByID(_ context.Context, id uuid.UUID) (usermodels.User, error) {
	if f.failure != nil {
		return usermodels.User{}, f.failure
	}
	if id != f.found.ID {
		return usermodels.User{}, usererrors.ErrUserNotFound{}
	}

	return f.found, nil
}

// userReadingNextHandler answers 204 and keeps the user ID the middleware put in the request context.
type userReadingNextHandler struct {
	seenUserID uuid.UUID
	seenUserOK bool
}

func (h *userReadingNextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

// assertNothingLogged pins that a 401 is answered without a log line; client errors are not server events.
func (h *capturingLogHandler) assertNothingLogged(t *testing.T) {
	t.Helper()

	if len(h.entries) != 0 {
		t.Errorf("logged %d entries, want none for a rejected request", len(h.entries))
	}
}

func captureHandlerLogs(t *testing.T) *capturingLogHandler {
	t.Helper()

	logs := &capturingLogHandler{}
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(logs))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	return logs
}

func authenticatedUserFixture() usermodels.User {
	createdAt := time.Date(2026, 3, 14, 9, 26, 53, 0, time.UTC)

	return usermodels.User{
		ID:        uuid.MustParse("6f1c9f2e-4a1b-4c3d-9e8f-0a1b2c3d4e5f"),
		Email:     "accountant@example.com",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func claimsFor(authenticated usermodels.User) jwt.AccessTokenClaims {
	return jwt.AccessTokenClaims{
		UserID:    authenticated.ID,
		Email:     authenticated.Email,
		IssuedAt:  authenticated.CreatedAt,
		ExpiresAt: authenticated.CreatedAt.Add(jwt.AccessTokenLifetime),
	}
}

type sendGuardedRequestParams struct {
	parser        *stubAccessTokenParser
	users         *stubUserFinder
	authorization string
	omitHeader    bool
}

func sendGuardedRequest(t *testing.T, params sendGuardedRequestParams) (*httptest.ResponseRecorder, *userReadingNextHandler) {
	t.Helper()

	next := &userReadingNextHandler{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
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
			parser := &stubAccessTokenParser{claims: claimsFor(authenticated)}
			users := &stubUserFinder{found: authenticated}

			recorder, next := sendGuardedRequest(t, sendGuardedRequestParams{
				parser:        parser,
				users:         users,
				authorization: testCase.scheme + " " + accessTokenForTest,
			})

			if recorder.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want the next handler's %d for a valid token", recorder.Code, http.StatusNoContent)
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
			if len(logs.entries) != 0 {
				t.Errorf("logged %d entries, want none for an admitted request", len(logs.entries))
			}
		})
	}
}

func TestRequireAuthenticatedUserRejectsRequestsWithoutAUsableBearerToken(t *testing.T) {
	cases := []struct {
		name          string
		authorization string
		omitHeader    bool
		wantChallenge string
		wantCode      handlers.ErrorCode
	}{
		{
			name:          "no header",
			omitHeader:    true,
			wantChallenge: tokenRequiredChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenMissing,
		},
		{
			name:          "wrong scheme",
			authorization: "Basic abc",
			wantChallenge: handlers.InvalidTokenChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenInvalid,
		},
		{
			name:          "wrong scheme of exactly six characters is logged whole",
			authorization: "Digest abc",
			wantChallenge: handlers.InvalidTokenChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenInvalid,
		},
		{
			name:          "wrong scheme longer than six characters is truncated",
			authorization: "Negotiate abc",
			wantChallenge: handlers.InvalidTokenChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenInvalid,
		},
		{
			name:          "token without a scheme is truncated",
			authorization: accessTokenForTest,
			wantChallenge: handlers.InvalidTokenChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenInvalid,
		},
		{
			name:          "malformed: scheme only",
			authorization: "Bearer",
			wantChallenge: handlers.InvalidTokenChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenInvalid,
		},
		{
			name:          "malformed: double space before the token",
			authorization: "Bearer  " + accessTokenForTest,
			wantChallenge: handlers.InvalidTokenChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenInvalid,
		},
		{
			name:          "malformed: tab inside the token",
			authorization: "Bearer header.payload\t.signature",
			wantChallenge: handlers.InvalidTokenChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenInvalid,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			logs := captureHandlerLogs(t)

			authenticated := authenticatedUserFixture()
			parser := &stubAccessTokenParser{claims: claimsFor(authenticated)}
			users := &stubUserFinder{found: authenticated}

			recorder, _ := sendGuardedRequest(t, sendGuardedRequestParams{
				parser:        parser,
				users:         users,
				authorization: testCase.authorization,
				omitHeader:    testCase.omitHeader,
			})

			assertRejection(t, recorder, testCase.wantChallenge, testCase.wantCode)

			logs.assertNothingLogged(t)
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
			wantChallenge: handlers.InvalidTokenChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenInvalid,
		},
		{
			name:          "unusable claim",
			failure:       jwt.ErrClaimUnusable{Claim: "sub", Cause: errors.New(leakedDetail)},
			wantChallenge: handlers.InvalidTokenChallenge,
			wantCode:      handlers.ErrorCodeAccessTokenInvalid,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			logs := captureHandlerLogs(t)

			parser := &stubAccessTokenParser{failure: testCase.failure}
			users := &stubUserFinder{found: authenticatedUserFixture()}

			recorder, _ := sendGuardedRequest(t, sendGuardedRequestParams{
				parser:        parser,
				users:         users,
				authorization: "Bearer " + accessTokenForTest,
			})

			assertRejection(t, recorder, testCase.wantChallenge, testCase.wantCode)

			logs.assertNothingLogged(t)
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
	users := &stubUserFinder{failure: lookupFailure}

	recorder, _ := sendGuardedRequest(t, sendGuardedRequestParams{
		parser:        &stubAccessTokenParser{claims: claimsFor(authenticated)},
		users:         users,
		authorization: "Bearer " + accessTokenForTest,
	})

	assertRejection(t, recorder, handlers.InvalidTokenChallenge, handlers.ErrorCodeAccessTokenInvalid)

	logs.assertNothingLogged(t)
}

func TestRequireAuthenticatedUserAnswersInternalServerErrorWhenTheLookupFails(t *testing.T) {
	logs := captureHandlerLogs(t)

	authenticated := authenticatedUserFixture()
	leakedDetail := "dial tcp 127.0.0.1:5432: connect: connection refused"
	lookupFailure := usererrors.ErrFindUserFailed{Cause: errors.New(leakedDetail)}

	recorder, _ := sendGuardedRequest(t, sendGuardedRequestParams{
		parser:        &stubAccessTokenParser{claims: claimsFor(authenticated)},
		users:         &stubUserFinder{failure: lookupFailure},
		authorization: "Bearer " + accessTokenForTest,
	})

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d: a database outage is not a verdict on the token",
			recorder.Code, http.StatusInternalServerError)
	}
	if challenge := recorder.Header().Get("WWW-Authenticate"); challenge != "" {
		t.Errorf("WWW-Authenticate = %q, want it absent: a 500 issues no challenge", challenge)
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

func TestReadBearerTokenReportsWhyAHeaderIsUnusable(t *testing.T) {
	cases := []struct {
		name      string
		rawHeader string
		wantToken string
		wantErr   error
	}{
		{name: "bearer token", rawHeader: "Bearer " + accessTokenForTest, wantToken: accessTokenForTest},
		{name: "scheme case is ignored", rawHeader: "bearer " + accessTokenForTest, wantToken: accessTokenForTest},
		{name: "no header", rawHeader: "", wantErr: ErrAuthorizationHeaderMissing{}},
		{name: "wrong scheme", rawHeader: "Basic abc", wantErr: ErrUnsupportedAuthorizationScheme{Scheme: "Basic"}},
		{name: "wrong scheme of exactly six characters is kept whole", rawHeader: "Digest abc",
			wantErr: ErrUnsupportedAuthorizationScheme{Scheme: "Digest"}},
		{name: "wrong scheme longer than six characters is truncated", rawHeader: "Negotiate abc",
			wantErr: ErrUnsupportedAuthorizationScheme{Scheme: "Negoti..."}},
		{name: "token without a scheme is truncated", rawHeader: accessTokenForTest,
			wantErr: ErrUnsupportedAuthorizationScheme{Scheme: "header..."}},
		{name: "scheme only", rawHeader: "Bearer", wantErr: ErrBearerTokenMalformed{}},
		{name: "double space before the token", rawHeader: "Bearer  " + accessTokenForTest, wantErr: ErrBearerTokenMalformed{}},
		{name: "tab inside the token", rawHeader: "Bearer header.payload\t.signature", wantErr: ErrBearerTokenMalformed{}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			token, err := ReadBearerToken(testCase.rawHeader)

			if !errors.Is(err, testCase.wantErr) || token != testCase.wantToken {
				t.Fatalf("ReadBearerToken = %q, %#v; want %q, %#v", token, err, testCase.wantToken, testCase.wantErr)
			}
			if err != nil && strings.Contains(err.Error(), accessTokenForTest) {
				t.Errorf("error = %q, want no trace of the access token, so it is safe to log", err)
			}
		})
	}
}
