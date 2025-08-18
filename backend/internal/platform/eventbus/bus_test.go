package eventbus_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"backend/internal/platform/eventbus"
)

// mockLogger implements the logger.Logger interface for testing
type mockLogger struct {
	mu     sync.Mutex
	errors []string
}

func (m *mockLogger) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) Info(ctx context.Context, msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Warn(ctx context.Context, msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors = append(m.errors, msg)
}

func (m *mockLogger) getErrors() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.errors))
	copy(result, m.errors)
	return result
}

func TestBusSubscribeAndPublish(t *testing.T) {
	logger := &mockLogger{}
	bus := eventbus.NewBus(logger)

	topic := eventbus.Topic("test.event")

	// Track calls to handlers
	var mu sync.Mutex
	handlerCalls := make([]string, 0)

	// Subscribe first handler
	handler1 := func(ctx context.Context, event eventbus.Event) error {
		mu.Lock()
		defer mu.Unlock()
		handlerCalls = append(handlerCalls, "handler1")
		payload, ok := event.Payload.(string)
		if !ok {
			t.Error("expected string payload")
		}
		if payload != "test message" {
			t.Errorf("expected 'test message', got %v", payload)
		}
		return nil
	}
	bus.Subscribe(topic, handler1)

	// Subscribe second handler
	handler2 := func(ctx context.Context, event eventbus.Event) error {
		mu.Lock()
		defer mu.Unlock()
		handlerCalls = append(handlerCalls, "handler2")
		return nil
	}
	bus.Subscribe(topic, handler2)

	// Publish event
	event := eventbus.Event{
		Topic:   topic,
		Payload: "test message",
	}
	bus.Publish(context.Background(), event)

	// Wait briefly for async handlers to complete
	time.Sleep(50 * time.Millisecond)

	// Verify both handlers were called
	mu.Lock()
	defer mu.Unlock()
	if len(handlerCalls) != 2 {
		t.Fatalf("expected 2 handler calls, got %d", len(handlerCalls))
	}

	// Verify both handlers ran (order may vary due to async)
	foundHandler1 := false
	foundHandler2 := false
	for _, call := range handlerCalls {
		if call == "handler1" {
			foundHandler1 = true
		}
		if call == "handler2" {
			foundHandler2 = true
		}
	}
	if !foundHandler1 {
		t.Error("handler1 was not called")
	}
	if !foundHandler2 {
		t.Error("handler2 was not called")
	}
}

func TestBusPublishWithNoSubscribers(t *testing.T) {
	logger := &mockLogger{}
	bus := eventbus.NewBus(logger)

	// Publish to a topic with no subscribers (should not panic)
	event := eventbus.Event{
		Topic:   eventbus.Topic("no.subscribers"),
		Payload: "test",
	}
	bus.Publish(context.Background(), event)

	// No error should be logged
	errors := logger.getErrors()
	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d", len(errors))
	}
}

func TestBusPublishWithHandlerError(t *testing.T) {
	logger := &mockLogger{}
	bus := eventbus.NewBus(logger)

	topic := eventbus.Topic("error.event")

	// Subscribe handler that returns an error
	handlerErr := errors.New("handler failed")
	handler := func(ctx context.Context, event eventbus.Event) error {
		return handlerErr
	}
	bus.Subscribe(topic, handler)

	// Publish event
	event := eventbus.Event{
		Topic:   topic,
		Payload: "test",
	}
	bus.Publish(context.Background(), event)

	// Wait briefly for async handler to complete
	time.Sleep(50 * time.Millisecond)

	// Verify error was logged
	errors := logger.getErrors()
	if len(errors) != 1 {
		t.Fatalf("expected 1 error log, got %d", len(errors))
	}
	if errors[0] != "event handler failed" {
		t.Errorf("expected 'event handler failed', got %v", errors[0])
	}
}

func TestBusRequest(t *testing.T) {
	logger := &mockLogger{}
	bus := eventbus.NewBus(logger)

	topic := eventbus.Topic("request.event")

	// Subscribe handler that replies
	handler := func(ctx context.Context, event eventbus.Event) error {
		// Check payload
		request, ok := event.Payload.(string)
		if !ok {
			event.ErrorChannel <- errors.New("invalid payload type")
			return errors.New("invalid payload type")
		}

		// Send reply
		reply := eventbus.Event{
			Payload: "reply to: " + request,
		}
		event.ReplyChannel <- reply
		return nil
	}
	bus.Subscribe(topic, handler)

	// Send request
	ctx := context.Background()
	request := eventbus.Event{
		Topic:   topic,
		Payload: "my request",
	}

	reply, err := bus.Request(ctx, request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Verify reply
	replyPayload, ok := reply.Payload.(string)
	if !ok {
		t.Fatal("expected string reply payload")
	}
	if replyPayload != "reply to: my request" {
		t.Errorf("expected 'reply to: my request', got %v", replyPayload)
	}
}

func TestBusRequestWithNoHandler(t *testing.T) {
	logger := &mockLogger{}
	bus := eventbus.NewBus(logger)

	// Send request to topic with no handler
	ctx := context.Background()
	request := eventbus.Event{
		Topic:   eventbus.Topic("no.handler"),
		Payload: "test",
	}

	_, err := bus.Request(ctx, request)
	if err == nil {
		t.Fatal("expected error for no handler")
	}
	if err.Error() != "no handler registered for request topic: no.handler" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestBusRequestWithHandlerError(t *testing.T) {
	logger := &mockLogger{}
	bus := eventbus.NewBus(logger)

	topic := eventbus.Topic("error.request")

	// Subscribe handler that sends error
	handlerErr := errors.New("processing failed")
	handler := func(ctx context.Context, event eventbus.Event) error {
		event.ErrorChannel <- handlerErr
		return handlerErr
	}
	bus.Subscribe(topic, handler)

	// Send request
	ctx := context.Background()
	request := eventbus.Event{
		Topic:   topic,
		Payload: "test",
	}

	_, err := bus.Request(ctx, request)
	if err == nil {
		t.Fatal("expected error from handler")
	}
	if err.Error() != "processing failed" {
		t.Errorf("expected 'processing failed', got %v", err)
	}
}

func TestBusRequestWithTimeout(t *testing.T) {
	logger := &mockLogger{}
	bus := eventbus.NewBus(logger)

	topic := eventbus.Topic("slow.request")

	// Subscribe handler that never replies
	handler := func(ctx context.Context, event eventbus.Event) error {
		// Simulate slow processing - never send reply
		time.Sleep(1 * time.Second)
		return nil
	}
	bus.Subscribe(topic, handler)

	// Send request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	request := eventbus.Event{
		Topic:   topic,
		Payload: "test",
	}

	_, err := bus.Request(ctx, request)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestBusRequestWithMultipleHandlers(t *testing.T) {
	logger := &mockLogger{}
	bus := eventbus.NewBus(logger)

	topic := eventbus.Topic("multi.request")

	// Subscribe first handler
	handler1 := func(ctx context.Context, event eventbus.Event) error {
		reply := eventbus.Event{
			Payload: "reply from handler1",
		}
		event.ReplyChannel <- reply
		return nil
	}
	bus.Subscribe(topic, handler1)

	// Subscribe second handler (should be ignored for request/reply)
	handler2 := func(ctx context.Context, event eventbus.Event) error {
		reply := eventbus.Event{
			Payload: "reply from handler2",
		}
		event.ReplyChannel <- reply
		return nil
	}
	bus.Subscribe(topic, handler2)

	// Send request
	ctx := context.Background()
	request := eventbus.Event{
		Topic:   topic,
		Payload: "test",
	}

	reply, err := bus.Request(ctx, request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Should get reply from first handler only
	replyPayload, ok := reply.Payload.(string)
	if !ok {
		t.Fatal("expected string reply payload")
	}
	if replyPayload != "reply from handler1" {
		t.Errorf("expected 'reply from handler1', got %v", replyPayload)
	}
}

func TestBusConcurrentSubscribe(t *testing.T) {
	logger := &mockLogger{}
	bus := eventbus.NewBus(logger)

	topic := eventbus.Topic("concurrent.subscribe")

	// Concurrently subscribe multiple handlers
	var wg sync.WaitGroup
	handlerCount := 10

	for i := 0; i < handlerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			handler := func(ctx context.Context, event eventbus.Event) error {
				return nil
			}
			bus.Subscribe(topic, handler)
		}(i)
	}

	wg.Wait()

	// Verify we can publish without issues
	event := eventbus.Event{
		Topic:   topic,
		Payload: "test",
	}
	bus.Publish(context.Background(), event)
}

func TestBusConcurrentPublish(t *testing.T) {
	logger := &mockLogger{}
	bus := eventbus.NewBus(logger)

	topic := eventbus.Topic("concurrent.publish")

	// Track handler calls
	var mu sync.Mutex
	callCount := 0

	handler := func(ctx context.Context, event eventbus.Event) error {
		mu.Lock()
		defer mu.Unlock()
		callCount++
		return nil
	}
	bus.Subscribe(topic, handler)

	// Concurrently publish multiple events
	var wg sync.WaitGroup
	publishCount := 10

	for i := 0; i < publishCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := eventbus.Event{
				Topic:   topic,
				Payload: id,
			}
			bus.Publish(context.Background(), event)
		}(i)
	}

	wg.Wait()

	// Wait for async handlers
	time.Sleep(100 * time.Millisecond)

	// Verify all events were handled
	mu.Lock()
	defer mu.Unlock()
	if callCount != publishCount {
		t.Errorf("expected %d handler calls, got %d", publishCount, callCount)
	}
}
