package rabbitmq

import (
	"fmt"
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const brokerUnreachableText = "failed connecting to the message broker"
const publisherUnavailableText = "failed opening the publisher's channel on the broker"
const publisherNotWiredText = "expected a publisher built by NewEventPublisher"
const publishFailedText = "failed handing the event to the broker"
const publishNackedText = "the broker refused the event"
const confirmUnavailableText = "the channel is not in confirm mode, so the broker will never acknowledge the event"
const confirmTimedOutText = "the broker did not acknowledge the event in time, so whether it was stored is unknown"
const consumerUnavailableText = "failed opening the consumer's channel on the broker"
const consumerNotWiredText = "expected a consumer built by NewEventConsumer"
const consumerQueueMissingText = "expected the name of the queue to consume"
const consumerBindingMissingText = "expected at least one routing key to bind the queue with"
const consumerHandlerMissingText = "expected a handler to give each delivered event to"
const consumerStartFailedText = "failed declaring the queue and its binding, so nothing is being consumed"
const deadLetterExchangeUndeclaredText = "failed declaring the dead-letter exchange, so exhausted events would be dropped"
const deadLetterQueueUndeclaredText = "failed declaring the dead-letter queue, so exhausted events would be dropped"

// ErrBrokerUnreachable reports a broker the server could not dial.
type ErrBrokerUnreachable struct {
	Cause error
}

func (e ErrBrokerUnreachable) Error() string {
	return fmt.Sprintf("%s: %s", brokerUnreachableText, e.Cause)
}

func (e ErrBrokerUnreachable) Unwrap() error {
	return e.Cause
}

// ErrPublisherUnavailable reports a publisher that could not open its channel or exchange.
type ErrPublisherUnavailable struct {
	Cause error
}

func (e ErrPublisherUnavailable) Error() string {
	return fmt.Sprintf("%s %q: %s", publisherUnavailableText, EventExchange, e.Cause)
}

func (e ErrPublisherUnavailable) Unwrap() error {
	return e.Cause
}

// ErrPublisherNotWired reports a publish through the zero value of EventPublisher.
type ErrPublisherNotWired struct {
	Label events.Label
}

func (e ErrPublisherNotWired) Error() string {
	return fmt.Sprintf("cannot publish %q: %s", e.Label, publisherNotWiredText)
}

// ErrPublishFailed reports an event that never reached the broker because the write failed.
type ErrPublishFailed struct {
	Label      events.Label
	RoutingKey string
	Cause      error
}

func (e ErrPublishFailed) Error() string {
	return fmt.Sprintf("%s for %q on %q: %s", publishFailedText, e.Label, e.RoutingKey, e.Cause)
}

func (e ErrPublishFailed) Unwrap() error {
	return e.Cause
}

// ErrPublishNacked reports an event the broker answered for and refused.
type ErrPublishNacked struct {
	Label      events.Label
	RoutingKey string
}

func (e ErrPublishNacked) Error() string {
	return fmt.Sprintf("%s %q on %q", publishNackedText, e.Label, e.RoutingKey)
}

// ErrConfirmUnavailable reports a publish on a channel that was not in confirm mode.
type ErrConfirmUnavailable struct {
	Label events.Label
}

func (e ErrConfirmUnavailable) Error() string {
	return fmt.Sprintf("cannot confirm %q: %s", e.Label, confirmUnavailableText)
}

// ErrPublishConfirmTimedOut reports an acknowledgement that did not arrive before the timeout.
type ErrPublishConfirmTimedOut struct {
	Label   events.Label
	Timeout time.Duration
	Cause   error
}

func (e ErrPublishConfirmTimedOut) Error() string {
	return fmt.Sprintf("%s within %s for %q: %s", confirmTimedOutText, e.Timeout, e.Label, e.Cause)
}

func (e ErrPublishConfirmTimedOut) Unwrap() error {
	return e.Cause
}

// ErrConsumerUnavailable reports a consumer that could not open its channel on the broker.
type ErrConsumerUnavailable struct {
	Queue string
	Cause error
}

func (e ErrConsumerUnavailable) Error() string {
	return fmt.Sprintf("%s for %q: %s", consumerUnavailableText, e.Queue, e.Cause)
}

func (e ErrConsumerUnavailable) Unwrap() error {
	return e.Cause
}

// ErrConsumerNotWired reports a run through the zero value of EventConsumer.
type ErrConsumerNotWired struct {
	Queue string
}

func (e ErrConsumerNotWired) Error() string {
	return fmt.Sprintf("cannot consume %q: %s", e.Queue, consumerNotWiredText)
}

// ErrConsumerQueueMissing reports a consumer declared without a queue to consume.
type ErrConsumerQueueMissing struct{}

func (e ErrConsumerQueueMissing) Error() string {
	return fmt.Sprintf("cannot consume: %s", consumerQueueMissingText)
}

// ErrConsumerBindingMissing reports a consumer with no routing keys, whose queue would stay empty.
type ErrConsumerBindingMissing struct {
	Queue string
}

func (e ErrConsumerBindingMissing) Error() string {
	return fmt.Sprintf("cannot consume %q: %s", e.Queue, consumerBindingMissingText)
}

// ErrConsumerHandlerMissing reports a consumer built without anything to hand its events to.
type ErrConsumerHandlerMissing struct {
	Queue string
}

func (e ErrConsumerHandlerMissing) Error() string {
	return fmt.Sprintf("cannot consume %q: %s", e.Queue, consumerHandlerMissingText)
}

// ErrConsumerStartFailed reports a consumer whose topology the broker refused on the way in.
type ErrConsumerStartFailed struct {
	Queue string
	Cause error
}

func (e ErrConsumerStartFailed) Error() string {
	return fmt.Sprintf("%s for %q: %s", consumerStartFailedText, e.Queue, e.Cause)
}

func (e ErrConsumerStartFailed) Unwrap() error {
	return e.Cause
}

// ErrDeadLetterExchangeUndeclared reports a dead-letter exchange the broker would not declare.
type ErrDeadLetterExchangeUndeclared struct {
	Cause error
}

func (e ErrDeadLetterExchangeUndeclared) Error() string {
	return fmt.Sprintf("%s %q: %s", deadLetterExchangeUndeclaredText, DeadLetterExchange, e.Cause)
}

func (e ErrDeadLetterExchangeUndeclared) Unwrap() error {
	return e.Cause
}

// ErrDeadLetterQueueUndeclared reports a dead-letter queue the broker would not declare or bind.
type ErrDeadLetterQueueUndeclared struct {
	Queue string
	Cause error
}

func (e ErrDeadLetterQueueUndeclared) Error() string {
	return fmt.Sprintf("%s %q: %s", deadLetterQueueUndeclaredText, e.Queue, e.Cause)
}

func (e ErrDeadLetterQueueUndeclared) Unwrap() error {
	return e.Cause
}
