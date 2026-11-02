package users

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
)

func sendRequestEmailChange(t *testing.T, stores *userStores, userID uuid.UUID, body string) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler:  stores.requestEmailChangeHandler(),
		method:   http.MethodPost,
		path:     "/api/v1/users/change-email",
		body:     body,
		asUserID: userID,
	})
}

func TestRequestEmailChangeSendsALinkToTheNormalizedNewAddress(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail, verified: true})

	response := sendRequestEmailChange(t, stores, userID, `{"newEmail":"  Bookkeeper@Example.com "}`)

	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusAccepted)
	}
	if sent := stores.challenges.linksSentTo("bookkeeper@example.com"); sent != 1 {
		t.Errorf("links sent to the new address = %d, want 1", sent)
	}
	if email := stores.storedUser(t, userID).Email; email != testEmail {
		t.Errorf("stored email = %q, want %q kept until the link is clicked", email, testEmail)
	}
}

func TestRequestEmailChangeRejectsWithoutSendingALink(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   handlers.ErrorCode
	}{
		{name: "malformed json", body: `{"newEmail":`, wantStatus: http.StatusBadRequest, wantCode: handlers.ErrorCodeRequestBodyInvalid},
		{name: "invalid email", body: `{"newEmail":"bob"}`, wantStatus: http.StatusBadRequest, wantCode: handlers.ErrorCodeEmailInvalid},
		{
			name:       "current email",
			body:       `{"newEmail":"accountant@example.com"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodeEmailUnchanged,
		},
		{
			name:       "email held by another account",
			body:       `{"newEmail":"taken@example.com"}`,
			wantStatus: http.StatusConflict,
			wantCode:   handlers.ErrorCodeEmailTaken,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stores := newUserStores(t)
			userID := stores.seedUser(t, seedUserParams{email: testEmail, verified: true})
			stores.seedUser(t, seedUserParams{email: "taken@example.com"})

			response := sendRequestEmailChange(t, stores, userID, testCase.body)

			assertErrorCode(t, response, testCase.wantStatus, testCase.wantCode)
			if len(stores.challenges.Challenges) != 0 {
				t.Errorf("links sent = %d, want none", len(stores.challenges.Challenges))
			}
		})
	}
}

func TestRequestEmailChangeAnswersARateLimitWithTheWait(t *testing.T) {
	silenceHandlerLogger(t)
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail, verified: true})
	stores.rateLimited["bookkeeper@example.com"] = true

	response := sendRequestEmailChange(t, stores, userID, `{"newEmail":"bookkeeper@example.com"}`)

	if retryAfter := response.Header.Get("Retry-After"); retryAfter != "20" {
		t.Errorf("Retry-After = %q, want %q", retryAfter, "20")
	}
	assertErrorCode(t, response, http.StatusTooManyRequests, handlers.ErrorCodeVerificationEmailRateLimited)
}
