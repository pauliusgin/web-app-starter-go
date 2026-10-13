package events

import "context"

// Publisher hands an event to the message broker for other services to consume.
type Publisher func(ctx context.Context, event NewEvent) error
