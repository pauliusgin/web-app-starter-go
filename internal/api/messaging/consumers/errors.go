package consumers

import (
	"fmt"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const eventPayloadUndecodableText = "failed decoding the delivered event payload as JSON"

// ErrEventPayloadUndecodable reports a delivered payload json.Unmarshal refused.
type ErrEventPayloadUndecodable struct {
	Label events.Label
	Cause error
}

func (e ErrEventPayloadUndecodable) Error() string {
	return fmt.Sprintf("%s for %q: %s", eventPayloadUndecodableText, e.Label, e.Cause)
}

func (e ErrEventPayloadUndecodable) Unwrap() error {
	return e.Cause
}
