package users

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

const signInBodyForTest = `{"email":"accountant@example.com","password":"correct-horse"}`

var signedInAtForTest = time.Date(2026, 3, 15, 8, 30, 0, 0, time.UTC)

var signedInUserForTest = models.User{
	ID:        uuid.MustParse("6f1c9f2e-4a1b-4c3d-9e8f-0a1b2c3d4e5f"),
	Email:     "accountant@example.com",
	CreatedAt: time.Date(2026, 3, 14, 9, 26, 53, 0, time.UTC),
	UpdatedAt: signedInAtForTest,
	SigninAt:  signedInAtForTest,
}

func signInAsSignedInUser(context.Context, usecases.SignInUserParams) (usecases.SignInUserResult, error) {
	return usecases.SignInUserResult{User: signedInUserForTest}, nil
}

func findSignedInUserReadModel(_ context.Context, id uuid.UUID) (readmodels.UserReadModel, error) {
	return readmodels.UserReadModel{
		ID:        id,
		Email:     signedInUserForTest.Email,
		CreatedAt: signedInUserForTest.CreatedAt,
		SigninAt:  &signedInAtForTest,
	}, nil
}

func newWorkingSignInDependencies(t *testing.T) SignInUserDependencies {
	t.Helper()

	return SignInUserDependencies{
		SignInUser:            signInAsSignedInUser,
		FindUserReadModelByID: findSignedInUserReadModel,
		IssueRefreshToken:     issueRefreshTokenForTest,
		Jwt:                   newAccessTokenIssuerForTest(t),
	}
}

type signInRequestParams struct {
	Body         string
	Dependencies SignInUserDependencies
}

func signInWithDependencies(t *testing.T, params signInRequestParams) *http.Response {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users/signin", strings.NewReader(params.Body))
	request.Header.Set("Content-Type", "application/json")

	SignInUser(params.Dependencies)(recorder, request)

	response := recorder.Result()
	t.Cleanup(func() { response.Body.Close() })

	return response
}

type assertSignInRefusedParams struct {
	response   *http.Response
	wantStatus int
	wantCode   handlers.ErrorCode
}

func assertSignInRefused(t *testing.T, params assertSignInRefusedParams) {
	t.Helper()

	if params.response.StatusCode != params.wantStatus {
		t.Fatalf("status = %d, want %d", params.response.StatusCode, params.wantStatus)
	}

	body := decodeBody(t, params.response)
	if code, _ := body["code"].(string); code != string(params.wantCode) {
		t.Errorf(`body["code"] = %q, want %q`, code, params.wantCode)
	}
	if message, _ := body["message"].(string); message == "" {
		t.Error(`body["message"] is empty, want a message`)
	}
	if body["accessToken"] != nil || body["refreshToken"] != nil {
		t.Errorf("response carries tokens %v, want none", body)
	}
}

func TestSignInUserAnswersWithTheReadModelAndAFreshTokenPair(t *testing.T) {
	dependencies := newWorkingSignInDependencies(t)

	response := signInWithDependencies(t, signInRequestParams{
		Body:         signInBodyForTest,
		Dependencies: dependencies,
	})

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	body := decodeBody(t, response)
	if body["id"] != signedInUserForTest.ID.String() {
		t.Errorf("id = %v, want %s", body["id"], signedInUserForTest.ID)
	}
	if body["email"] != signedInUserForTest.Email {
		t.Errorf("email = %v, want %q", body["email"], signedInUserForTest.Email)
	}
	if body["signinAt"] != signedInAtForTest.Format(time.RFC3339) {
		t.Errorf("signinAt = %v, want %s", body["signinAt"], signedInAtForTest.Format(time.RFC3339))
	}
	if _, carriesUpdatedAt := body["updatedAt"]; carriesUpdatedAt {
		t.Error("response carries updatedAt, want only the read model's fields")
	}
	if body["refreshToken"] != issuedRefreshTokenForTest {
		t.Errorf("refreshToken = %v, want %q", body["refreshToken"], issuedRefreshTokenForTest)
	}

	accessToken, _ := body["accessToken"].(string)
	claims, err := dependencies.Jwt.ParseAccessToken(accessToken)
	if err != nil {
		t.Fatalf("parsing the issued access token: %v", err)
	}
	if claims.UserID != signedInUserForTest.ID {
		t.Errorf("access token subject = %s, want %s", claims.UserID, signedInUserForTest.ID)
	}
}

func TestSignInUserAnswersNullForWhatTheUserHasNotRecordedYet(t *testing.T) {
	response := signInWithDependencies(t, signInRequestParams{
		Body:         signInBodyForTest,
		Dependencies: newWorkingSignInDependencies(t),
	})

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	body := decodeBody(t, response)
	for _, key := range []string{"emailVerifiedAt", "firstName", "lastName"} {
		value, present := body[key]
		if !present || value != nil {
			t.Errorf("%s = %v (present %t), want present as null", key, value, present)
		}
	}
}

func TestSignInUserRefusesInvalidCredentialsAsUnauthorized(t *testing.T) {
	dependencies := newWorkingSignInDependencies(t)
	dependencies.SignInUser = func(context.Context, usecases.SignInUserParams) (usecases.SignInUserResult, error) {
		return usecases.SignInUserResult{}, usererrors.ErrInvalidCredentials{}
	}

	response := signInWithDependencies(t, signInRequestParams{
		Body:         signInBodyForTest,
		Dependencies: dependencies,
	})

	assertSignInRefused(t, assertSignInRefusedParams{
		response:   response,
		wantStatus: http.StatusUnauthorized,
		wantCode:   handlers.ErrorCodeInvalidCredentials,
	})
}

func TestSignInUserRefusesBadRequestsBeforeSigningIn(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode handlers.ErrorCode
	}{
		{name: "undecodable body", body: `{"email":`, wantCode: handlers.ErrorCodeRequestBodyInvalid},
		{name: "invalid email", body: `{"email":"not-an-email","password":"correct-horse"}`,
			wantCode: handlers.ErrorCodeEmailInvalid},
		{name: "missing password", body: `{"email":"accountant@example.com"}`,
			wantCode: handlers.ErrorCodePasswordMissing},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			signInAttempted := false
			dependencies := newWorkingSignInDependencies(t)
			dependencies.SignInUser = func(ctx context.Context, params usecases.SignInUserParams) (usecases.SignInUserResult, error) {
				signInAttempted = true
				return signInAsSignedInUser(ctx, params)
			}

			response := signInWithDependencies(t, signInRequestParams{
				Body:         testCase.body,
				Dependencies: dependencies,
			})

			assertSignInRefused(t, assertSignInRefusedParams{
				response:   response,
				wantStatus: http.StatusBadRequest,
				wantCode:   testCase.wantCode,
			})
			if signInAttempted {
				t.Error("sign-in attempted with a request the handler should have rejected")
			}
		})
	}
}

func TestSignInUserAnswersUnexpectedFailuresWithAGenericError(t *testing.T) {
	cases := []struct {
		name   string
		breaks func(dependencies *SignInUserDependencies)
	}{
		{name: "sign-in fails", breaks: func(dependencies *SignInUserDependencies) {
			dependencies.SignInUser = func(context.Context, usecases.SignInUserParams) (usecases.SignInUserResult, error) {
				return usecases.SignInUserResult{}, errCollaboratorFailedForTest
			}
		}},
		{name: "read model is missing after the stamp", breaks: func(dependencies *SignInUserDependencies) {
			dependencies.FindUserReadModelByID = func(context.Context, uuid.UUID) (readmodels.UserReadModel, error) {
				return readmodels.UserReadModel{}, usererrors.ErrUserNotFound{}
			}
		}},
		{name: "access token issuing fails", breaks: func(dependencies *SignInUserDependencies) {
			dependencies.Jwt = refusingAccessTokenIssuer{}
		}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			silenceHandlerLogger(t)

			dependencies := newWorkingSignInDependencies(t)
			testCase.breaks(&dependencies)

			response := signInWithDependencies(t, signInRequestParams{
				Body:         signInBodyForTest,
				Dependencies: dependencies,
			})

			assertSignInRefused(t, assertSignInRefusedParams{
				response:   response,
				wantStatus: http.StatusInternalServerError,
				wantCode:   handlers.ErrorCodeInternal,
			})
		})
	}
}

func TestSignInUserNormalizesTheEmailBeforeSigningIn(t *testing.T) {
	var signedInEmails []string
	dependencies := newWorkingSignInDependencies(t)
	dependencies.SignInUser = func(ctx context.Context, params usecases.SignInUserParams) (usecases.SignInUserResult, error) {
		signedInEmails = append(signedInEmails, params.Email)
		return signInAsSignedInUser(ctx, params)
	}

	response := signInWithDependencies(t, signInRequestParams{
		Body:         `{"email":"  Bob@Example.COM ","password":"correct-horse"}`,
		Dependencies: dependencies,
	})

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if len(signedInEmails) != 1 || signedInEmails[0] != "bob@example.com" {
		t.Errorf("signed in emails = %q, want [\"bob@example.com\"]", signedInEmails)
	}
}
