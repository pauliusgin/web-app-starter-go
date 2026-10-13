package events

import "fmt"

const recordEventFailedText = "failed recording the event"

const recorderMissingText = "no event recorder is wired, so the event went unrecorded"

const publishEventFailedText = "failed publishing the event"

const publisherMissingText = "no event publisher is wired, so the event went unpublished"

const buildEventFailedText = "failed building the event, so it was neither recorded nor published"

const eventDataUnencodableText = "failed encoding the event payload as JSON"

// ErrEventDataUnencodable reports an event payload json.Marshal refused.
type ErrEventDataUnencodable struct {
	Label Label
	Cause error
}

func (e ErrEventDataUnencodable) Error() string {
	return fmt.Sprintf("%s for %q: %s", eventDataUnencodableText, e.Label, e.Cause)
}

func (e ErrEventDataUnencodable) Unwrap() error {
	return e.Cause
}

// ErrRecordEventFailed reports an event the database refused to record in the event log.
type ErrRecordEventFailed struct {
	Label Label
	Cause error
}

func (e ErrRecordEventFailed) Error() string {
	return fmt.Sprintf("%s for %q: %s", recordEventFailedText, e.Label, e.Cause)
}

func (e ErrRecordEventFailed) Unwrap() error {
	return e.Cause
}
