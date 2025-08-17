package eventbus

import "context"

// Topic is the type for event topics.
type Topic string

// Event represents a message passed on the bus.
type Event struct {
	Topic   Topic
	Payload any // The data associated with the event.

	// For the Request/Reply pattern
	ReplyChannel chan Event
	ErrorChannel chan error
}

// Handler is a function that processes an event.
type Handler func(ctx context.Context, event Event) error
