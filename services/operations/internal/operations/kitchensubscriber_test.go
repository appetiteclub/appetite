package operations

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/appetiteclub/appetite/pkg/event"
	"github.com/aquamarinepk/aqm/events"
)

// MockSubscriber implements events.Subscriber for testing
type MockSubscriber struct {
	SubscribeFunc func(ctx context.Context, topic string, handler events.HandlerFunc) error
}

func (m *MockSubscriber) Subscribe(ctx context.Context, topic string, handler events.HandlerFunc) error {
	if m.SubscribeFunc != nil {
		return m.SubscribeFunc(ctx, topic, handler)
	}
	return nil
}

func TestNewKitchenTicketSubscriber(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	tests := []struct {
		name       string
		subscriber events.Subscriber
	}{
		{
			name:       "withNilSubscriber",
			subscriber: nil,
		},
		{
			name:       "withMockSubscriber",
			subscriber: &MockSubscriber{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := NewKitchenTicketSubscriber(tt.subscriber, cache, nil)
			if sub == nil {
				t.Fatal("NewKitchenTicketSubscriber() returned nil")
			}
			if sub.cache != cache {
				t.Error("cache not set correctly")
			}
			if sub.logger == nil {
				t.Error("logger should default to noop logger")
			}
		})
	}
}

func TestKitchenTicketSubscriberStartWithNilSubscriber(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)
	sub := NewKitchenTicketSubscriber(nil, cache, nil)

	err := sub.Start(context.Background())
	if err != nil {
		t.Errorf("Start() with nil subscriber should not error, got: %v", err)
	}
}

func TestKitchenTicketSubscriberStartSuccess(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)
	mockSub := &MockSubscriber{
		SubscribeFunc: func(ctx context.Context, topic string, handler events.HandlerFunc) error {
			if topic != event.KitchenTicketsTopic {
				t.Errorf("topic = %q, want %q", topic, event.KitchenTicketsTopic)
			}
			return nil
		},
	}
	sub := NewKitchenTicketSubscriber(mockSub, cache, nil)

	err := sub.Start(context.Background())
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}
}

func TestKitchenTicketSubscriberStartError(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)
	mockSub := &MockSubscriber{
		SubscribeFunc: func(ctx context.Context, topic string, handler events.HandlerFunc) error {
			return errors.New("subscription failed")
		},
	}
	sub := NewKitchenTicketSubscriber(mockSub, cache, nil)

	err := sub.Start(context.Background())
	if err == nil {
		t.Error("Start() should return error when subscription fails")
	}
}

func TestKitchenTicketSubscriberStop(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)
	sub := NewKitchenTicketSubscriber(nil, cache, nil)

	err := sub.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestKitchenTicketSubscriberHandleTicketCreated(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)
	sub := NewKitchenTicketSubscriber(nil, cache, nil)
	ctx := context.Background()

	now := time.Now()
	evt := event.KitchenTicketCreatedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:    event.EventKitchenTicketCreated,
			OccurredAt:   now,
			TicketID:     "ticket-1",
			OrderID:      "order-1",
			OrderItemID:  "item-1",
			MenuItemID:   "menu-1",
			Station:      "grill",
			MenuItemName: "Burger",
			StationName:  "Grill Station",
			TableNumber:  "5",
		},
		Status:   "created",
		Quantity: 2,
		Notes:    "no onions",
	}
	data, _ := json.Marshal(evt)

	err := sub.handleTicketCreated(ctx, data)
	if err != nil {
		t.Fatalf("handleTicketCreated() error = %v", err)
	}

	// Verify ticket is in cache
	ticket := cache.Get("ticket-1")
	if ticket == nil {
		t.Fatal("ticket not found in cache")
	}
	if ticket.MenuItemName != "Burger" {
		t.Errorf("MenuItemName = %q, want %q", ticket.MenuItemName, "Burger")
	}
}

func TestKitchenTicketSubscriberHandleTicketCreatedInvalidJSON(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)
	sub := NewKitchenTicketSubscriber(nil, cache, nil)

	// Should not panic, should return nil (graceful handling)
	err := sub.handleTicketCreated(context.Background(), []byte("invalid json"))
	if err != nil {
		t.Errorf("handleTicketCreated() with invalid JSON should return nil, got: %v", err)
	}
}

func TestKitchenTicketSubscriberHandleTicketStatusChanged(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)
	sub := NewKitchenTicketSubscriber(nil, cache, nil)
	ctx := context.Background()

	// First create a ticket in cache
	cache.Set(&kitchenTicketResource{
		ID:      "ticket-1",
		Station: "grill",
		Status:  "created",
	})

	now := time.Now()
	evt := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:  event.EventKitchenTicketStatusChange,
			OccurredAt: now,
			TicketID:   "ticket-1",
			Station:    "grill",
		},
		NewStatus: "started",
	}
	data, _ := json.Marshal(evt)

	err := sub.handleTicketStatusChanged(ctx, data)
	if err != nil {
		t.Fatalf("handleTicketStatusChanged() error = %v", err)
	}

	// Verify status was updated
	ticket := cache.Get("ticket-1")
	if ticket == nil {
		t.Fatal("ticket not found in cache")
	}
	if ticket.Status != "started" {
		t.Errorf("Status = %q, want %q", ticket.Status, "started")
	}
}

func TestKitchenTicketSubscriberHandleTicketStatusChangedNewTicket(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)
	sub := NewKitchenTicketSubscriber(nil, cache, nil)
	ctx := context.Background()

	now := time.Now()
	evt := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:    event.EventKitchenTicketStatusChange,
			OccurredAt:   now,
			TicketID:     "new-ticket",
			OrderID:      "order-1",
			Station:      "grill",
			MenuItemName: "Pizza",
		},
		NewStatus: "started",
	}
	data, _ := json.Marshal(evt)

	err := sub.handleTicketStatusChanged(ctx, data)
	if err != nil {
		t.Fatalf("handleTicketStatusChanged() error = %v", err)
	}

	// Verify new ticket was created
	ticket := cache.Get("new-ticket")
	if ticket == nil {
		t.Fatal("new ticket should be created")
	}
	if ticket.Status != "started" {
		t.Errorf("Status = %q, want %q", ticket.Status, "started")
	}
}

func TestKitchenTicketSubscriberHandleTicketStatusChangedWithReasonCode(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)
	sub := NewKitchenTicketSubscriber(nil, cache, nil)
	ctx := context.Background()

	cache.Set(&kitchenTicketResource{
		ID:      "ticket-1",
		Station: "grill",
		Status:  "created",
	})

	now := time.Now()
	evt := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:  event.EventKitchenTicketStatusChange,
			OccurredAt: now,
			TicketID:   "ticket-1",
			Station:    "grill",
		},
		NewStatus:    "cancelled",
		ReasonCodeID: "out-of-stock",
		Notes:        "Item unavailable",
	}
	data, _ := json.Marshal(evt)

	err := sub.handleTicketStatusChanged(ctx, data)
	if err != nil {
		t.Fatalf("handleTicketStatusChanged() error = %v", err)
	}

	ticket := cache.Get("ticket-1")
	if ticket == nil {
		t.Fatal("ticket not found")
	}
	if ticket.ReasonCodeID == nil || *ticket.ReasonCodeID != "out-of-stock" {
		t.Errorf("ReasonCodeID not set correctly")
	}
}

func TestKitchenTicketSubscriberHandleEvent(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)
	sub := NewKitchenTicketSubscriber(nil, cache, nil)
	ctx := context.Background()

	tests := []struct {
		name      string
		eventType string
		wantErr   bool
	}{
		{
			name:      "unknownEventType",
			eventType: "unknown.event",
			wantErr:   false,
		},
		{
			name:      "invalidJSON",
			eventType: "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data []byte
			if tt.eventType == "" {
				data = []byte("invalid json")
			} else {
				data, _ = json.Marshal(map[string]string{"event_type": tt.eventType})
			}

			err := sub.handleEvent(ctx, data)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
