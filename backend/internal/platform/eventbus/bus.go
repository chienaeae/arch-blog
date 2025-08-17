package eventbus

import (
	"context"
	"errors"
	"sync"

	"github.com/philly/arch-blog/backend/internal/platform/logger"
)

// Bus manages subscriptions and event dispatching.
type Bus struct {
	subscriptions map[Topic][]Handler
	mu            sync.RWMutex // Protects the subscriptions map
	logger        logger.Logger
}

// NewBus creates a new event bus.
func NewBus(logger logger.Logger) *Bus {
	return &Bus{
		subscriptions: make(map[Topic][]Handler),
		logger:        logger,
	}
}

// Subscribe adds a handler for a specific topic.
func (b *Bus) Subscribe(topic Topic, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscriptions[topic] = append(b.subscriptions[topic], handler)
}

// Publish sends an event to all subscribers of a topic (Fire-and-Forget).
func (b *Bus) Publish(ctx context.Context, event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if handlers, found := b.subscriptions[event.Topic]; found {
		for _, handler := range handlers {
			// Run each handler in its own goroutine for true asynchronicity.
			go func(h Handler) {
				if err := h(ctx, event); err != nil {
					b.logger.Error(ctx, "event handler failed", "topic", event.Topic, "error", err)
				}
			}(handler)
		}
	}
}

// Request sends an event and waits for a single reply.
func (b *Bus) Request(ctx context.Context, event Event) (Event, error) {
	b.mu.RLock()
	handlers, found := b.subscriptions[event.Topic]
	b.mu.RUnlock()

	if !found || len(handlers) == 0 {
		return Event{}, errors.New("no handler registered for request topic: " + string(event.Topic))
	}

	// For request/reply, we typically expect only one handler. Use the first one.
	handler := handlers[0]

	// Set up channels for the reply.
	event.ReplyChannel = make(chan Event, 1)
	event.ErrorChannel = make(chan error, 1)

	// Run the handler in a goroutine so we can respect the context timeout.
	go func() {
		// The handler is responsible for sending a reply or an error.
		// We can ignore the returned error here because it's sent via the channel.
		_ = handler(ctx, event)
	}()

	// Wait for a reply, an error, or a timeout from the context.
	select {
	case reply := <-event.ReplyChannel:
		return reply, nil
	case err := <-event.ErrorChannel:
		return Event{}, err
	case <-ctx.Done():
		return Event{}, ctx.Err()
	}
}