package kitchen

import (
	"testing"
	"time"

	"github.com/appetiteclub/appetite/pkg/event"
	proto "github.com/appetiteclub/appetite/services/kitchen/internal/kitchen/proto"
	"github.com/aquamarinepk/aqm"
	"github.com/google/uuid"
)

func TestNewEventStreamServer(t *testing.T) {
	tests := []struct {
		name   string
		cache  *TicketStateCache
		logger aqm.Logger
	}{
		{
			name:   "withAllDependencies",
			cache:  NewTicketStateCache(nil, nil, nil),
			logger: aqm.NewNoopLogger(),
		},
		{
			name:   "withNilCache",
			cache:  nil,
			logger: aqm.NewNoopLogger(),
		},
		{
			name:   "withNilLogger",
			cache:  NewTicketStateCache(nil, nil, nil),
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewEventStreamServer(tt.cache, tt.logger)
			if server == nil {
				t.Error("NewEventStreamServer() returned nil")
			}
			if server.subscribers == nil {
				t.Error("subscribers map is nil")
			}
		})
	}
}

func TestEventStreamServerBroadcastTicketEvent(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	server := NewEventStreamServer(cache, aqm.NewNoopLogger())

	// Add a subscriber
	testChan := make(chan *proto.KitchenTicketEvent, 10)
	server.mu.Lock()
	server.subscribers["test-subscriber"] = testChan
	server.mu.Unlock()

	// Broadcast an event
	now := time.Now()
	evt := &event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:    "kitchen.ticket.status_changed",
			OccurredAt:   now,
			TicketID:     uuid.New().String(),
			OrderID:      uuid.New().String(),
			OrderItemID:  uuid.New().String(),
			MenuItemID:   uuid.New().String(),
			Station:      "kitchen",
			MenuItemName: "Burger",
			StationName:  "Kitchen",
			TableNumber:  "T1",
		},
		NewStatus:      "started",
		PreviousStatus: "created",
		Notes:          "Test notes",
	}

	server.BroadcastTicketEvent(evt)

	// Verify event was received
	select {
	case received := <-testChan:
		if received.NewStatusId != "started" {
			t.Errorf("NewStatusId = %v, want 'started'", received.NewStatusId)
		}
		if received.PreviousStatusId != "created" {
			t.Errorf("PreviousStatusId = %v, want 'created'", received.PreviousStatusId)
		}
		if received.StationId != "kitchen" {
			t.Errorf("StationId = %v, want 'kitchen'", received.StationId)
		}
		if received.MenuItemName != "Burger" {
			t.Errorf("MenuItemName = %v, want 'Burger'", received.MenuItemName)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected event was not received")
	}
}

func TestEventStreamServerBroadcastWithTimestamps(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	server := NewEventStreamServer(cache, aqm.NewNoopLogger())

	testChan := make(chan *proto.KitchenTicketEvent, 10)
	server.mu.Lock()
	server.subscribers["test"] = testChan
	server.mu.Unlock()

	now := time.Now()
	startedAt := now.Add(-10 * time.Minute)
	finishedAt := now.Add(-5 * time.Minute)
	deliveredAt := now

	evt := &event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:  "kitchen.ticket.status_changed",
			OccurredAt: now,
			TicketID:   uuid.New().String(),
		},
		NewStatus:   "delivered",
		StartedAt:   &startedAt,
		FinishedAt:  &finishedAt,
		DeliveredAt: &deliveredAt,
	}

	server.BroadcastTicketEvent(evt)

	select {
	case received := <-testChan:
		if received.StartedAt == nil {
			t.Error("StartedAt should not be nil")
		}
		if received.FinishedAt == nil {
			t.Error("FinishedAt should not be nil")
		}
		if received.DeliveredAt == nil {
			t.Error("DeliveredAt should not be nil")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected event was not received")
	}
}

func TestEventStreamServerBroadcastChannelFull(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	server := NewEventStreamServer(cache, aqm.NewNoopLogger())

	// Create a channel with 0 buffer that's already "full"
	fullChan := make(chan *proto.KitchenTicketEvent)
	server.mu.Lock()
	server.subscribers["full-subscriber"] = fullChan
	server.mu.Unlock()

	evt := &event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType: "kitchen.ticket.status_changed",
			TicketID:  uuid.New().String(),
		},
		NewStatus: "started",
	}

	// Should not block - event is dropped when channel is full
	done := make(chan bool)
	go func() {
		server.BroadcastTicketEvent(evt)
		done <- true
	}()

	select {
	case <-done:
		// Success - broadcast completed without blocking
	case <-time.After(100 * time.Millisecond):
		t.Error("BroadcastTicketEvent blocked on full channel")
	}
}

func TestEventStreamServerBroadcastToMultipleSubscribers(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	server := NewEventStreamServer(cache, aqm.NewNoopLogger())

	chan1 := make(chan *proto.KitchenTicketEvent, 10)
	chan2 := make(chan *proto.KitchenTicketEvent, 10)
	chan3 := make(chan *proto.KitchenTicketEvent, 10)

	server.mu.Lock()
	server.subscribers["sub1"] = chan1
	server.subscribers["sub2"] = chan2
	server.subscribers["sub3"] = chan3
	server.mu.Unlock()

	evt := &event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType: "kitchen.ticket.status_changed",
			TicketID:  uuid.New().String(),
		},
		NewStatus: "ready",
	}

	server.BroadcastTicketEvent(evt)

	// All subscribers should receive the event
	for i, ch := range []chan *proto.KitchenTicketEvent{chan1, chan2, chan3} {
		select {
		case received := <-ch:
			if received.NewStatusId != "ready" {
				t.Errorf("Subscriber %d: NewStatusId = %v, want 'ready'", i+1, received.NewStatusId)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Subscriber %d did not receive event", i+1)
		}
	}
}

func TestEventStreamServerBroadcastNoSubscribers(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	server := NewEventStreamServer(cache, aqm.NewNoopLogger())

	evt := &event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType: "kitchen.ticket.status_changed",
			TicketID:  uuid.New().String(),
		},
		NewStatus: "started",
	}

	// Should not panic with no subscribers
	server.BroadcastTicketEvent(evt)
}

func TestGenerateSubscriberID(t *testing.T) {
	id1 := generateSubscriberID()
	if id1 == "" {
		t.Error("generateSubscriberID() returned empty string")
	}

	// IDs should be unique (though this is probabilistic)
	time.Sleep(time.Millisecond) // Ensure different timestamp
	id2 := generateSubscriberID()
	if id1 == id2 {
		t.Error("generateSubscriberID() returned same ID twice")
	}
}

func TestEventStreamServerSubscriberManagement(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	server := NewEventStreamServer(cache, aqm.NewNoopLogger())

	// Add subscribers
	server.mu.Lock()
	server.subscribers["sub1"] = make(chan *proto.KitchenTicketEvent, 10)
	server.subscribers["sub2"] = make(chan *proto.KitchenTicketEvent, 10)
	server.mu.Unlock()

	server.mu.RLock()
	count := len(server.subscribers)
	server.mu.RUnlock()

	if count != 2 {
		t.Errorf("subscriber count = %d, want 2", count)
	}

	// Remove a subscriber
	server.mu.Lock()
	delete(server.subscribers, "sub1")
	server.mu.Unlock()

	server.mu.RLock()
	count = len(server.subscribers)
	server.mu.RUnlock()

	if count != 1 {
		t.Errorf("subscriber count after delete = %d, want 1", count)
	}
}
