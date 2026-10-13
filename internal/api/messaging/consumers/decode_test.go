package consumers

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const decodeTestLabel = events.Label("test.decodable")

type decodeTestPayload struct {
	ChallengeID string `json:"challengeId"`
	Attempts    int    `json:"attempts"`
}

func TestDecodeEventPayloadReadsAWellFormedBody(t *testing.T) {
	payload, err := decodeEventPayload[decodeTestPayload](decodeEventPayloadParams{
		Body:  []byte(`{"challengeId":"challenge-1","attempts":3}`),
		Label: decodeTestLabel,
	})

	if err != nil {
		t.Fatalf("error = %v, want nil for a well-formed body", err)
	}
	want := decodeTestPayload{ChallengeID: "challenge-1", Attempts: 3}
	if payload != want {
		t.Errorf("payload = %+v, want %+v", payload, want)
	}
}

func TestDecodeEventPayloadReportsAnUndecodableBody(t *testing.T) {
	payload, err := decodeEventPayload[decodeTestPayload](decodeEventPayloadParams{
		Body:  []byte(`{"challengeId": `),
		Label: decodeTestLabel,
	})

	undecodable := ErrEventPayloadUndecodable{}
	if !errors.As(err, &undecodable) {
		t.Fatalf("error = %v (%T), want %T", err, err, undecodable)
	}
	if undecodable.Label != decodeTestLabel {
		t.Errorf("Label = %q, want %q", undecodable.Label, decodeTestLabel)
	}
	syntaxError := &json.SyntaxError{}
	if !errors.As(err, &syntaxError) {
		t.Errorf("Cause = %v (%T), want the json syntax failure unwrapped", undecodable.Cause, undecodable.Cause)
	}
	if payload != (decodeTestPayload{}) {
		t.Errorf("payload = %+v, want the zero value when the body cannot be decoded", payload)
	}
}
