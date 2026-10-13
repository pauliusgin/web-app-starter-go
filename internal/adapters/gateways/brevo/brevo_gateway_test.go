package brevo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	email "github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
)

const testApiKey = "test-api-key"

const testSenderEmail = "no-reply@example.com"

const testRecipientEmail = "accountant@example.com"

const testVerificationLink = "http://localhost:3001/verify-email?token=abc123"

const testVerificationLinkParam = "verificationUrl"

const testTemplateID = 7

const apiVersionPrefix = "/v3"

const sendTransacEmailPath = apiVersionPrefix + "/smtp/email"

type recordedRequest struct {
	path              string
	apiKeyHeaders     []string
	partnerKeyHeaders []string
	body              map[string]any
}

type fakeBrevoAPI struct {
	baseURL  string
	requests chan recordedRequest
}

func newFakeBrevoAPI(t *testing.T, statusCode int, responseBody string) *fakeBrevoAPI {
	t.Helper()

	api := &fakeBrevoAPI{requests: make(chan recordedRequest, 1)}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawBody, _ := io.ReadAll(r.Body)

		recorded := recordedRequest{
			path:              r.URL.Path,
			apiKeyHeaders:     r.Header.Values(apiKeyHeader),
			partnerKeyHeaders: r.Header.Values("partner-key"),
		}
		_ = json.Unmarshal(rawBody, &recorded.body)

		api.requests <- recorded

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = io.WriteString(w, responseBody)
	}))
	t.Cleanup(server.Close)

	api.baseURL = server.URL + apiVersionPrefix

	return api
}

func (a *fakeBrevoAPI) onlyRequest(t *testing.T) recordedRequest {
	t.Helper()

	select {
	case recorded := <-a.requests:
		return recorded
	default:
		t.Fatal("the gateway sent no request")
		return recordedRequest{}
	}
}

func newTestEmailGateway(t *testing.T, baseURL string) EmailGateway {
	t.Helper()

	gateway, err := NewEmailGateway(EmailGatewayParams{
		ApiKey:                 testApiKey,
		SenderEmail:            testSenderEmail,
		VerificationTemplateID: testTemplateID,
	})
	if err != nil {
		t.Fatalf("NewEmailGateway: %v", err)
	}

	gateway.client.ChangeBasePath(baseURL)

	return gateway
}

func testVerificationMessage() email.VerificationEmailPayload {
	return email.VerificationEmailPayload{
		RecipientEmail:        testRecipientEmail,
		VerificationLinkParam: testVerificationLinkParam,
		VerificationLink:      testVerificationLink,
	}
}

func TestSendVerificationEmailPostsTheTemplateIDTheRecipientAndTheLink(t *testing.T) {
	api := newFakeBrevoAPI(t, http.StatusCreated, `{"messageId":"sent"}`)
	gateway := newTestEmailGateway(t, api.baseURL)

	_, err := gateway.SendVerificationEmail(context.Background(), testVerificationMessage())
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	recorded := api.onlyRequest(t)

	if recorded.path != sendTransacEmailPath {
		t.Errorf("path = %q, want %q", recorded.path, sendTransacEmailPath)
	}

	if recorded.body["templateId"] != float64(testTemplateID) {
		t.Errorf("templateId = %#v, want the number %d", recorded.body["templateId"], testTemplateID)
	}

	params, ok := recorded.body["params"].(map[string]any)
	if !ok {
		t.Fatalf("params = %#v, want a json object", recorded.body["params"])
	}
	if params[testVerificationLinkParam] != testVerificationLink {
		t.Errorf("params[%s] = %#v, want %q",
			testVerificationLinkParam, params[testVerificationLinkParam], testVerificationLink)
	}
	if len(params) != 1 {
		t.Errorf("params = %#v, want only %s", params, testVerificationLinkParam)
	}

	recipients, ok := recorded.body["to"].([]any)
	if !ok || len(recipients) != 1 {
		t.Fatalf("to = %#v, want exactly one recipient", recorded.body["to"])
	}
	recipient, ok := recipients[0].(map[string]any)
	if !ok || recipient["email"] != testRecipientEmail {
		t.Errorf("to[0] = %#v, want email %q", recipients[0], testRecipientEmail)
	}

	sender, ok := recorded.body["sender"].(map[string]any)
	if !ok || sender["email"] != testSenderEmail {
		t.Errorf("sender = %#v, want email %q", recorded.body["sender"], testSenderEmail)
	}
}

func TestSendVerificationEmailAnswersWithTheMessageIDBrevoAssigned(t *testing.T) {
	const brevoMessageID = "<202409.1@smtp-relay.brevo.com>"

	api := newFakeBrevoAPI(t, http.StatusCreated, `{"messageId":"`+brevoMessageID+`"}`)
	gateway := newTestEmailGateway(t, api.baseURL)

	sentEmail, err := gateway.SendVerificationEmail(context.Background(), testVerificationMessage())
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	if sentEmail.ProviderMessageID != brevoMessageID {
		t.Errorf("ProviderMessageID = %q, want %q", sentEmail.ProviderMessageID, brevoMessageID)
	}
}

func TestSendVerificationEmailOmitsTheFieldsTheTemplateOwns(t *testing.T) {
	api := newFakeBrevoAPI(t, http.StatusCreated, `{"messageId":"sent"}`)
	gateway := newTestEmailGateway(t, api.baseURL)

	_, err := gateway.SendVerificationEmail(context.Background(), testVerificationMessage())
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	recorded := api.onlyRequest(t)

	for _, field := range []string{"subject", "htmlContent", "textContent"} {
		if _, present := recorded.body[field]; present {
			t.Errorf("body carries %q = %#v, want it absent when a template id is sent",
				field, recorded.body[field])
		}
	}
}

func TestSendVerificationEmailSendsTheApiKeyOnceAndNeverAsAPartnerKey(t *testing.T) {
	api := newFakeBrevoAPI(t, http.StatusCreated, `{"messageId":"sent"}`)
	gateway := newTestEmailGateway(t, api.baseURL)

	_, err := gateway.SendVerificationEmail(context.Background(), testVerificationMessage())
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	recorded := api.onlyRequest(t)

	if len(recorded.apiKeyHeaders) != 1 || recorded.apiKeyHeaders[0] != testApiKey {
		t.Errorf("%s headers = %#v, want exactly one carrying %q",
			apiKeyHeader, recorded.apiKeyHeaders, testApiKey)
	}
	if len(recorded.partnerKeyHeaders) != 0 {
		t.Errorf("partner-key headers = %#v, want none", recorded.partnerKeyHeaders)
	}
}

func TestSendVerificationEmailAcceptsASuccessBodyItCannotDecode(t *testing.T) {
	api := newFakeBrevoAPI(t, http.StatusCreated, `not json at all`)
	gateway := newTestEmailGateway(t, api.baseURL)

	sentEmail, err := gateway.SendVerificationEmail(context.Background(), testVerificationMessage())
	if err != nil {
		t.Fatalf("SendVerificationEmail = %v, want the send treated as accepted", err)
	}
	if sentEmail.ProviderMessageID != "" {
		t.Errorf("ProviderMessageID = %q, want none read from an undecodable body", sentEmail.ProviderMessageID)
	}

	if recorded := api.onlyRequest(t); recorded.path != sendTransacEmailPath {
		t.Errorf("path = %q, want %q", recorded.path, sendTransacEmailPath)
	}
}

func TestSendVerificationEmailReportsTheStatusAndBodyBrevoRefusedWith(t *testing.T) {
	cases := []struct {
		name            string
		statusCode      int
		responseBody    string
		wantBodyMention string
	}{
		{
			name:            "bad request",
			statusCode:      http.StatusBadRequest,
			responseBody:    `{"code":"invalid_parameter","message":"templateId is invalid"}`,
			wantBodyMention: "templateId is invalid",
		},
		{
			name:            "server error",
			statusCode:      http.StatusInternalServerError,
			responseBody:    `upstream exploded`,
			wantBodyMention: "upstream exploded",
		},
		{
			name:            "rate limited",
			statusCode:      http.StatusTooManyRequests,
			responseBody:    `{"code":"too_many_requests","message":"slow down"}`,
			wantBodyMention: "slow down",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			api := newFakeBrevoAPI(t, testCase.statusCode, testCase.responseBody)
			gateway := newTestEmailGateway(t, api.baseURL)

			_, err := gateway.SendVerificationEmail(context.Background(), testVerificationMessage())

			rejected := ErrSendRejected{}
			if !errors.As(err, &rejected) {
				t.Fatalf("error = %v (%T), want %T", err, err, rejected)
			}
			if rejected.StatusCode != testCase.statusCode {
				t.Errorf("StatusCode = %d, want %d", rejected.StatusCode, testCase.statusCode)
			}
			if !strings.Contains(rejected.BodySnippet, testCase.wantBodyMention) {
				t.Errorf("BodySnippet = %q, want it to mention %q",
					rejected.BodySnippet, testCase.wantBodyMention)
			}
			if rejected.RecipientEmail != testRecipientEmail {
				t.Errorf("RecipientEmail = %q, want %q", rejected.RecipientEmail, testRecipientEmail)
			}
		})
	}
}

func TestSendVerificationEmailTruncatesABodyTooLongToLog(t *testing.T) {
	oversizedBody := strings.Repeat("a", maxBodySnippetLength*4)

	api := newFakeBrevoAPI(t, http.StatusBadRequest, oversizedBody)
	gateway := newTestEmailGateway(t, api.baseURL)

	_, err := gateway.SendVerificationEmail(context.Background(), testVerificationMessage())

	rejected := ErrSendRejected{}
	if !errors.As(err, &rejected) {
		t.Fatalf("error = %v (%T), want %T", err, err, rejected)
	}

	wantLength := maxBodySnippetLength + len(bodySnippetTruncatedMarker)
	if len(rejected.BodySnippet) != wantLength {
		t.Errorf("len(BodySnippet) = %d, want %d", len(rejected.BodySnippet), wantLength)
	}
	if !strings.HasSuffix(rejected.BodySnippet, bodySnippetTruncatedMarker) {
		t.Errorf("BodySnippet = %q, want it to end with %q",
			rejected.BodySnippet, bodySnippetTruncatedMarker)
	}
}

func TestSendVerificationEmailReportsNoStatusWhenBrevoNeverAnswers(t *testing.T) {
	unreachable := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachableURL := unreachable.URL
	unreachable.Close()

	gateway := newTestEmailGateway(t, unreachableURL)

	_, err := gateway.SendVerificationEmail(context.Background(), testVerificationMessage())

	unanswered := ErrSendUnanswered{}
	if !errors.As(err, &unanswered) {
		t.Fatalf("error = %v (%T), want %T", err, err, unanswered)
	}
	if unanswered.RecipientEmail != testRecipientEmail {
		t.Errorf("RecipientEmail = %q, want %q", unanswered.RecipientEmail, testRecipientEmail)
	}

	rejected := ErrSendRejected{}
	if errors.As(err, &rejected) {
		t.Errorf("error = %v, want no status-carrying error for a request that never landed", err)
	}
}

func TestSendVerificationEmailSurfacesCancellationRatherThanBlocking(t *testing.T) {
	releaseHandler := make(chan struct{})

	blocking := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-releaseHandler:
		case <-r.Context().Done():
		}
	}))

	defer blocking.Close()
	defer close(releaseHandler)

	gateway := newTestEmailGateway(t, blocking.URL)

	cancelledCtx, cancelSend := context.WithCancel(context.Background())
	cancelSend()

	_, err := gateway.SendVerificationEmail(cancelledCtx, testVerificationMessage())

	unanswered := ErrSendUnanswered{}
	if !errors.As(err, &unanswered) {
		t.Fatalf("error = %v (%T), want %T", err, err, unanswered)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want it to unwrap to context.Canceled", err)
	}
}

func TestSendVerificationEmailRefusesToSendThroughTheZeroValue(t *testing.T) {
	_, err := EmailGateway{}.SendVerificationEmail(context.Background(), testVerificationMessage())

	notWired := ErrGatewayNotWired{}
	if !errors.As(err, &notWired) {
		t.Fatalf("error = %v (%T), want %T", err, err, notWired)
	}
	if notWired.RecipientEmail != testRecipientEmail {
		t.Errorf("RecipientEmail = %q, want %q", notWired.RecipientEmail, testRecipientEmail)
	}
}

func TestNewEmailGatewayRejectsAnAccountItCouldNotSendWith(t *testing.T) {
	cases := []struct {
		name        string
		params      EmailGatewayParams
		wantErrType any
	}{
		{
			name: "empty api key",
			params: EmailGatewayParams{
				SenderEmail:            testSenderEmail,
				VerificationTemplateID: testTemplateID,
			},
			wantErrType: &ErrApiKeyMissing{},
		},
		{
			name: "empty sender email",
			params: EmailGatewayParams{
				ApiKey:                 testApiKey,
				VerificationTemplateID: testTemplateID,
			},
			wantErrType: &ErrSenderEmailMissing{},
		},
		{
			name: "zero template id",
			params: EmailGatewayParams{
				ApiKey:      testApiKey,
				SenderEmail: testSenderEmail,
			},
			wantErrType: &ErrVerificationTemplateIDInvalid{},
		},
		{
			name: "negative template id",
			params: EmailGatewayParams{
				ApiKey:                 testApiKey,
				SenderEmail:            testSenderEmail,
				VerificationTemplateID: -1,
			},
			wantErrType: &ErrVerificationTemplateIDInvalid{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			gateway, err := NewEmailGateway(testCase.params)

			if !errors.As(err, testCase.wantErrType) {
				t.Fatalf("error = %v (%T), want %T", err, err, testCase.wantErrType)
			}
			if gateway != (EmailGateway{}) {
				t.Errorf("gateway = %#v, want the zero value for a rejected account", gateway)
			}
		})
	}
}

func TestNewEmailGatewayAcceptsACompleteAccount(t *testing.T) {
	gateway, err := NewEmailGateway(EmailGatewayParams{
		ApiKey:                 testApiKey,
		SenderEmail:            testSenderEmail,
		VerificationTemplateID: testTemplateID,
	})
	if err != nil {
		t.Fatalf("NewEmailGateway: %v", err)
	}

	if gateway.senderEmail != testSenderEmail {
		t.Errorf("senderEmail = %q, want %q", gateway.senderEmail, testSenderEmail)
	}
	if gateway.verificationTemplateID != int64(testTemplateID) {
		t.Errorf("verificationTemplateID = %d, want %d",
			gateway.verificationTemplateID, testTemplateID)
	}
}
