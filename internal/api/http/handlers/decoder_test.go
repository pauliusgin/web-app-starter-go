package handlers

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type decoderTestRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func decodeTestRequest(t *testing.T, contentType string, body io.Reader) (decoderTestRequest, error) {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/users/signup", body)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	return DecodeRequest[decoderTestRequest](httptest.NewRecorder(), request)
}

func decodeTestRequestBody(t *testing.T, body string) (decoderTestRequest, error) {
	t.Helper()

	return decodeTestRequest(t, jsonMediaType, strings.NewReader(body))
}

func requireUndecodableBody(t *testing.T, decoded decoderTestRequest, err error) {
	t.Helper()

	undecodable := ErrJSONRequestBodyUndecodable{}
	if !errors.As(err, &undecodable) {
		t.Fatalf("DecodeRequest returned %v, want ErrJSONRequestBodyUndecodable", err)
	}
	if decoded != (decoderTestRequest{}) {
		t.Errorf("decoded = %+v, want the zero value on failure", decoded)
	}
}

func TestDecodeRequestReadsAWellFormedBody(t *testing.T) {
	decoded, err := decodeTestRequestBody(t, `{"email":"accountant@example.com","password":"correct-horse"}`)

	if err != nil {
		t.Fatalf("DecodeRequest returned %v, want no error", err)
	}
	if decoded.Email != "accountant@example.com" {
		t.Errorf("email = %q, want %q", decoded.Email, "accountant@example.com")
	}
	if decoded.Password != "correct-horse" {
		t.Errorf("password = %q, want %q", decoded.Password, "correct-horse")
	}
}

func TestDecodeRequestRejectsNonJSONContentTypes(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
	}{
		{name: "missing header", contentType: ""},
		{name: "plain text", contentType: "text/plain"},
		{name: "form encoded", contentType: "application/x-www-form-urlencoded"},
		{name: "unparseable header", contentType: "application/json; charset"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			body := strings.NewReader(`{"email":"accountant@example.com","password":"correct-horse"}`)

			decoded, err := decodeTestRequest(t, testCase.contentType, body)

			unsupported := ErrUnsupportedContentType{}
			if !errors.As(err, &unsupported) {
				t.Fatalf("DecodeRequest returned %v, want ErrUnsupportedContentType", err)
			}
			if unsupported.ContentType != testCase.contentType {
				t.Errorf("ContentType = %q, want the raw header %q", unsupported.ContentType, testCase.contentType)
			}
			if decoded != (decoderTestRequest{}) {
				t.Errorf("decoded = %+v, want the zero value on failure", decoded)
			}
		})
	}
}

func TestDecodeRequestRejectsAnEmptyBody(t *testing.T) {
	cases := []struct {
		name         string
		buildRequest func() *http.Request
	}{
		{
			name: "zero length body",
			buildRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/api/v1/users/signup", strings.NewReader(``))
			},
		},
		{
			name: "nil body",
			buildRequest: func() *http.Request {
				request := httptest.NewRequest(http.MethodPost, "/api/v1/users/signup", nil)
				request.Body = nil
				return request
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := testCase.buildRequest()
			request.Header.Set("Content-Type", jsonMediaType)

			decoded, err := DecodeRequest[decoderTestRequest](httptest.NewRecorder(), request)

			requireUndecodableBody(t, decoded, err)
			if !errors.Is(err, errEmptyRequestBody) {
				t.Errorf("DecodeRequest returned %v, want it to wrap errEmptyRequestBody", err)
			}
		})
	}
}

func jsonBodyOfSize(size int64) (body string, email string) {
	frame := `{"email":""}`
	email = strings.Repeat("a", int(size)-len(frame))

	return `{"email":"` + email + `"}`, email
}

func TestDecodeRequestLimitsTheBodyToMaxRequestBodyBytes(t *testing.T) {
	acceptedSizes := []struct {
		name string
		size int64
	}{
		{name: "body one byte under the limit", size: maxRequestBodyBytes - 1},
		{name: "body exactly at the limit", size: maxRequestBodyBytes},
	}

	for _, testCase := range acceptedSizes {
		t.Run(testCase.name, func(t *testing.T) {
			body, email := jsonBodyOfSize(testCase.size)

			decoded, err := decodeTestRequestBody(t, body)

			if err != nil {
				t.Fatalf("DecodeRequest returned %v, want no error for a %d byte body", err, len(body))
			}
			if decoded.Email != email {
				t.Errorf("email has %d bytes, want %d", len(decoded.Email), len(email))
			}
		})
	}

	t.Run("body one byte over the limit", func(t *testing.T) {
		body, _ := jsonBodyOfSize(maxRequestBodyBytes + 1)

		decoded, err := decodeTestRequestBody(t, body)

		tooLarge := ErrRequestBodyTooLarge{}
		if !errors.As(err, &tooLarge) {
			t.Fatalf("DecodeRequest returned %v, want ErrRequestBodyTooLarge", err)
		}
		if tooLarge.Limit != maxRequestBodyBytes {
			t.Errorf("Limit = %d, want %d", tooLarge.Limit, maxRequestBodyBytes)
		}
		if decoded != (decoderTestRequest{}) {
			t.Errorf("decoded = %+v, want the zero value on failure", decoded)
		}
	})
}

func TestDecodeRequestRejectsUnknownFields(t *testing.T) {
	decoded, err := decodeTestRequestBody(t, `{"email":"accountant@example.com","passwrod":"correct-horse"}`)

	requireUndecodableBody(t, decoded, err)
}

func TestDecodeRequestRejectsMoreThanOneJSONValue(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "second object", body: `{"email":"a@example.com"}{"email":"b@example.com"}`},
		{name: "second object after whitespace", body: "{\"email\":\"a@example.com\"}\n{\"email\":\"b@example.com\"}"},
		{name: "trailing scalar", body: `{"email":"a@example.com"} 42`},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			decoded, err := decodeTestRequestBody(t, testCase.body)

			requireUndecodableBody(t, decoded, err)
			if !errors.Is(err, errTrailingRequestBody) {
				t.Errorf("DecodeRequest returned %v, want it to wrap errTrailingRequestBody", err)
			}
		})
	}
}

func TestRequestDecodingRulesAnswerEveryDecoderFailure(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   ErrorCode
	}{
		{name: "undecodable", err: ErrJSONRequestBodyUndecodable{Cause: errEmptyRequestBody},
			wantStatus: http.StatusBadRequest, wantCode: ErrorCodeRequestBodyInvalid},
		{name: "too large", err: ErrRequestBodyTooLarge{Limit: 1024},
			wantStatus: http.StatusRequestEntityTooLarge, wantCode: ErrorCodeRequestBodyTooLarge},
		{name: "unsupported content type", err: ErrUnsupportedContentType{},
			wantStatus: http.StatusUnsupportedMediaType, wantCode: ErrorCodeUnsupportedContentType},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rejection, matched := SelectRejection(testCase.err, RequestDecodingRules)

			if !matched {
				t.Fatal("matched = false, want a decoding rule to match")
			}
			if rejection.Status != testCase.wantStatus || rejection.Code != testCase.wantCode {
				t.Errorf("rejection = %+v, want %d %s", rejection, testCase.wantStatus, testCase.wantCode)
			}
		})
	}
}
