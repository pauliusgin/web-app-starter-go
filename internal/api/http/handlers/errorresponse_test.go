package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestErrorResponseAnswersAbsentDetailsAsNull(t *testing.T) {
	recorder := httptest.NewRecorder()

	WriteJSON(recorder, 400, ErrorResponse{Code: ErrorCodeRequestBodyInvalid, Message: fixedMessageForTest})

	body := map[string]any{}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the response body: %v", err)
	}
	if details, present := body["details"]; !present || details != nil {
		t.Errorf(`body["details"] = %v (present %v), want null`, details, present)
	}
	if body["message"] != fixedMessageForTest {
		t.Errorf(`body["message"] = %v, want %q`, body["message"], fixedMessageForTest)
	}
}
