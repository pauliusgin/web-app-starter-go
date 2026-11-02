package profiles

import (
	"bytes"
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
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/queries"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/usecases"
	uploadmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
	uploadrepositories "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories"
	usermodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const profilesPath = "/api/v1/profiles"

var testUserID = uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")

var storedAt = time.Date(2026, time.October, 12, 9, 0, 0, 0, time.UTC)

var errDatabaseDown = errors.New("database unreachable")

func storedProfile() models.Profile {
	return models.Profile{
		ID:        testUserID,
		FirstName: "Ada",
		LastName:  "Lovelace",
		CreatedAt: storedAt,
		UpdatedAt: storedAt,
	}
}

func ignoringDispatcher() events.Dispatcher {
	ignoreEvent := func(context.Context, events.NewEvent) error { return nil }

	return events.Dispatcher{Recorder: ignoreEvent, Publisher: ignoreEvent}
}

var pendingPictureID = uuid.MustParse("7d2e5f3c-6a91-4b22-9c4d-3f8b0e7a5d26")

// pendingPictureUploads holds one pending profile picture the test user uploaded, as an upload request would leave it.
func pendingPictureUploads() *uploadrepositories.InMemoryUploadRepository {
	return uploadrepositories.NewInMemoryUploadRepository(uploadmodels.Upload{
		ID:          pendingPictureID,
		UserID:      testUserID,
		Purpose:     uploadmodels.PurposeProfilePicture,
		Kind:        uploadmodels.KindImage,
		PreviewPath: "picture/preview.webp",
		PreviewURL:  "https://files.example/picture/preview.webp",
		CreatedAt:   storedAt,
		UpdatedAt:   storedAt,
	})
}

func createProfileHandler(repository *repositories.InMemoryProfileRepository) http.Handler {
	dependencies := usecases.CreateProfileDependencies{
		ProfileWriteRepository: repository,
		EventDispatcher:        ignoringDispatcher(),
		AttachUpload:           pendingPictureUploads().AttachUpload,
	}

	return CreateProfile(CreateProfileDependencies{
		CreateProfile: func(ctx context.Context, params usecases.CreateProfileParams) (usecases.CreateProfileResult, error) {
			return usecases.CreateProfile(ctx, dependencies, params)
		},
		GetProfileByID: profileReader(repository),
	})
}

func updateProfileHandler(repository *repositories.InMemoryProfileRepository) http.Handler {
	dependencies := usecases.UpdateProfileDependencies{
		ProfileWriteRepository: repository,
		EventDispatcher:        ignoringDispatcher(),
		AttachUpload:           pendingPictureUploads().AttachUpload,
	}

	return UpdateProfile(UpdateProfileDependencies{
		UpdateProfile: func(ctx context.Context, params usecases.UpdateProfileParams) (usecases.UpdateProfileResult, error) {
			return usecases.UpdateProfile(ctx, dependencies, params)
		},
		GetProfileByID: profileReader(repository),
	})
}

func profileReader(repository *repositories.InMemoryProfileRepository) GetProfileByIDUseCase {
	dependencies := queries.GetProfileByIDDependencies{ProfileReadRepository: repository}

	return func(ctx context.Context, params queries.GetProfileByIDParams) (queries.GetProfileByIDResult, error) {
		return queries.GetProfileByID(ctx, dependencies, params)
	}
}

func authenticateForTest(userID uuid.UUID) func(http.Handler) http.Handler {
	return middleware.RequireAuthenticatedUser(middleware.RequireAuthenticatedUserDependencies{
		ParseAccessToken: func(string) (jwt.AccessTokenClaims, error) {
			return jwt.AccessTokenClaims{UserID: userID}, nil
		},
		FindUserByID: func(_ context.Context, id uuid.UUID) (usermodels.User, error) {
			return usermodels.User{ID: id, Email: "accountant@example.com"}, nil
		},
	})
}

type sendProfileRequestParams struct {
	handler   http.Handler
	method    string
	body      string
	skipGuard bool
}

func sendProfileRequest(t *testing.T, params sendProfileRequestParams) *http.Response {
	t.Helper()

	served := params.handler
	if !params.skipGuard {
		served = authenticateForTest(testUserID)(params.handler)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(params.method, profilesPath, strings.NewReader(params.body))
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
	if code, _ := decodeBody(t, response)["code"].(string); code != string(wantCode) {
		t.Errorf(`body["code"] = %q, want %q`, code, wantCode)
	}
}

func silenceHandlerLogger(t *testing.T) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}

var invalidProfileBodyCases = []struct {
	name     string
	body     string
	wantCode handlers.ErrorCode
}{
	{name: "malformed json", body: `{"firstName":`, wantCode: handlers.ErrorCodeRequestBodyInvalid},
	{name: "unknown field", body: `{"firstName":"Ada","lastName":"Lovelace","age":36}`,
		wantCode: handlers.ErrorCodeRequestBodyInvalid},
	{name: "missing first name", body: `{"lastName":"Lovelace"}`, wantCode: handlers.ErrorCodeFirstNameMissing},
	{name: "missing last name", body: `{"firstName":"Ada"}`, wantCode: handlers.ErrorCodeLastNameMissing},
	{name: "first name with a leading dash", body: `{"firstName":"-Taylor","lastName":"Joy"}`,
		wantCode: handlers.ErrorCodeFirstNameInvalid},
	{name: "last name with a digit", body: `{"firstName":"Ada","lastName":"Lovelace3"}`,
		wantCode: handlers.ErrorCodeLastNameInvalid},
	{name: "malformed picture upload id", body: `{"firstName":"Ada","lastName":"Lovelace","pictureUploadId":"me.jpg"}`,
		wantCode: handlers.ErrorCodeProfilePictureUploadInvalid},
	{name: "unknown picture upload", body: `{"firstName":"Ada","lastName":"Lovelace",` +
		`"pictureUploadId":"0b6f3a52-1c4d-4e8f-9a2b-7c5d3e1f0a94"}`,
		wantCode: handlers.ErrorCodeProfilePictureUploadInvalid},
}

// assertPictureResponse checks the picture object a profile response carries: the upload id when one is attached,
// and null for every other field, as the in-memory profile store holds no upload URLs.
func assertPictureResponse(t *testing.T, body map[string]any, wantUploadID any) {
	t.Helper()

	picture, isObject := body["picture"].(map[string]any)
	if !isObject {
		t.Fatalf(`body["picture"] = %v, want an object`, body["picture"])
	}
	want := map[string]any{"uploadId": wantUploadID, "previewUrl": nil, "originalUrl": nil}
	for key, wantValue := range want {
		gotValue, present := picture[key]
		if !present || gotValue != wantValue {
			t.Errorf("picture[%q] = %v (present %v), want %v", key, gotValue, present, wantValue)
		}
	}
}
