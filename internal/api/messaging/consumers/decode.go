package consumers

import (
	"encoding/json"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type decodeEventPayloadParams struct {
	Body  []byte
	Label events.Label
}

func decodeEventPayload[Payload any](params decodeEventPayloadParams) (Payload, error) {
	payload := *new(Payload)

	if err := json.Unmarshal(params.Body, &payload); err != nil {
		return *new(Payload), ErrEventPayloadUndecodable{Label: params.Label, Cause: err}
	}

	return payload, nil
}
