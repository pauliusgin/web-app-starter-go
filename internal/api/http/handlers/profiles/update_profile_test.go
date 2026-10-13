package profiles

import (
	"net/http"
	"testing"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/repositories"
)

func TestUpdateProfileAnswersOKWithTheReplacedNames(t *testing.T) {
	repository := repositories.NewInMemoryProfileRepository(storedProfile())

	response := sendProfileRequest(t, sendProfileRequestParams{
		handler: updateProfileHandler(repository),
		method:  http.MethodPut,
		body:    `{"firstName":" Augusta","lastName":"King "}`,
	})

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	body := decodeBody(t, response)
	if body["id"] != testUserID.String() || body["firstName"] != "Augusta" || body["lastName"] != "King" {
		t.Errorf("body = %v, want Augusta King under %s", body, testUserID)
	}
	if stored := repository.Profiles[testUserID]; stored.FirstName != "Augusta" || stored.LastName != "King" {
		t.Errorf("stored names = %q %q, want Augusta King", stored.FirstName, stored.LastName)
	}
}

func TestUpdateProfileRejectsABadRequestBeforeChangingAnything(t *testing.T) {
	for _, testCase := range invalidProfileBodyCases {
		t.Run(testCase.name, func(t *testing.T) {
			existing := storedProfile()
			repository := repositories.NewInMemoryProfileRepository(existing)

			response := sendProfileRequest(t, sendProfileRequestParams{
				handler: updateProfileHandler(repository),
				method:  http.MethodPut,
				body:    testCase.body,
			})

			assertErrorCode(t, response, http.StatusBadRequest, testCase.wantCode)
			if stored := repository.Profiles[testUserID]; stored != existing {
				t.Errorf("stored = %+v, want the existing %+v untouched", stored, existing)
			}
		})
	}
}

func TestUpdateProfileAnswersNotFoundWithoutAProfile(t *testing.T) {
	repository := repositories.NewInMemoryProfileRepository()

	response := sendProfileRequest(t, sendProfileRequestParams{
		handler: updateProfileHandler(repository),
		method:  http.MethodPut,
		body:    `{"firstName":"Augusta","lastName":"King"}`,
	})

	assertErrorCode(t, response, http.StatusNotFound, "profile_not_found")
	if len(repository.Profiles) != 0 {
		t.Errorf("stored profiles = %d, want none created by an update", len(repository.Profiles))
	}
}

func TestUpdateProfileAnswersInternalErrorWhenUpdatingFails(t *testing.T) {
	silenceHandlerLogger(t)
	repository := repositories.NewInMemoryProfileRepository(storedProfile())
	repository.UpdateFailure = e.ErrUpdateProfileFailed{UserID: testUserID, Cause: errDatabaseDown}

	response := sendProfileRequest(t, sendProfileRequestParams{
		handler: updateProfileHandler(repository),
		method:  http.MethodPut,
		body:    `{"firstName":"Augusta","lastName":"King"}`,
	})

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusInternalServerError)
	}
	body := decodeBody(t, response)
	if body["code"] != "internal_error" || body["message"] != unexpectedProfileUpdateFailureText {
		t.Errorf("body = %v, want internal_error with %q", body, unexpectedProfileUpdateFailureText)
	}
}

func TestUpdateProfileAnswersInternalErrorWhenTheRouteIsUnprotected(t *testing.T) {
	silenceHandlerLogger(t)
	existing := storedProfile()
	repository := repositories.NewInMemoryProfileRepository(existing)

	response := sendProfileRequest(t, sendProfileRequestParams{
		handler:   updateProfileHandler(repository),
		method:    http.MethodPut,
		body:      `{"firstName":"Augusta","lastName":"King"}`,
		skipGuard: true,
	})

	assertErrorCode(t, response, http.StatusInternalServerError, "internal_error")
	if stored := repository.Profiles[testUserID]; stored != existing {
		t.Errorf("stored = %+v, want the existing %+v untouched", stored, existing)
	}
}
