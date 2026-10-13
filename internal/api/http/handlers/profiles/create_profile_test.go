package profiles

import (
	"net/http"
	"testing"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/repositories"
)

func TestCreateProfileAnswersCreatedWithTheTrimmedProfile(t *testing.T) {
	repository := repositories.NewInMemoryProfileRepository()

	response := sendProfileRequest(t, sendProfileRequestParams{
		handler: createProfileHandler(repository),
		method:  http.MethodPost,
		body:    `{"firstName":"  Ada ","lastName":" Lovelace"}`,
	})

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	body := decodeBody(t, response)
	want := map[string]any{"id": testUserID.String(), "firstName": "Ada", "lastName": "Lovelace"}
	if len(body) != len(want) {
		t.Fatalf("body = %v, want exactly %v", body, want)
	}
	for key, wantValue := range want {
		if body[key] != wantValue {
			t.Errorf("body[%q] = %v, want %v", key, body[key], wantValue)
		}
	}
	if stored := repository.Profiles[testUserID]; stored.FirstName != "Ada" || stored.LastName != "Lovelace" {
		t.Errorf("stored names = %q %q, want trimmed Ada Lovelace", stored.FirstName, stored.LastName)
	}
}

func TestCreateProfileRejectsABadRequestBeforeStoringAnything(t *testing.T) {
	for _, testCase := range invalidProfileBodyCases {
		t.Run(testCase.name, func(t *testing.T) {
			repository := repositories.NewInMemoryProfileRepository()

			response := sendProfileRequest(t, sendProfileRequestParams{
				handler: createProfileHandler(repository),
				method:  http.MethodPost,
				body:    testCase.body,
			})

			assertErrorCode(t, response, http.StatusBadRequest, testCase.wantCode)
			if len(repository.Profiles) != 0 {
				t.Errorf("stored profiles = %d, want none", len(repository.Profiles))
			}
		})
	}
}

func TestCreateProfileAnswersConflictWhenTheUserHasAProfile(t *testing.T) {
	existing := storedProfile()
	repository := repositories.NewInMemoryProfileRepository(existing)

	response := sendProfileRequest(t, sendProfileRequestParams{
		handler: createProfileHandler(repository),
		method:  http.MethodPost,
		body:    `{"firstName":"Augusta","lastName":"King"}`,
	})

	assertErrorCode(t, response, http.StatusConflict, "profile_exists")
	if stored := repository.Profiles[testUserID]; stored != existing {
		t.Errorf("stored = %+v, want the existing %+v untouched", stored, existing)
	}
}

func TestCreateProfileAnswersInternalErrorWhenSavingFails(t *testing.T) {
	silenceHandlerLogger(t)
	repository := repositories.NewInMemoryProfileRepository()
	repository.SaveFailure = e.ErrSaveProfileFailed{UserID: testUserID, Cause: errDatabaseDown}

	response := sendProfileRequest(t, sendProfileRequestParams{
		handler: createProfileHandler(repository),
		method:  http.MethodPost,
		body:    `{"firstName":"Ada","lastName":"Lovelace"}`,
	})

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusInternalServerError)
	}
	body := decodeBody(t, response)
	if body["code"] != "internal_error" || body["message"] != unexpectedProfileCreationFailureText {
		t.Errorf("body = %v, want internal_error with %q", body, unexpectedProfileCreationFailureText)
	}
}

func TestCreateProfileAnswersInternalErrorWhenTheRouteIsUnprotected(t *testing.T) {
	silenceHandlerLogger(t)
	repository := repositories.NewInMemoryProfileRepository()

	response := sendProfileRequest(t, sendProfileRequestParams{
		handler:   createProfileHandler(repository),
		method:    http.MethodPost,
		body:      `{"firstName":"Ada","lastName":"Lovelace"}`,
		skipGuard: true,
	})

	assertErrorCode(t, response, http.StatusInternalServerError, "internal_error")
	if len(repository.Profiles) != 0 {
		t.Errorf("stored profiles = %d, want none", len(repository.Profiles))
	}
}
