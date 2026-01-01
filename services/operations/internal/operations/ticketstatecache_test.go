package operations

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/appetiteclub/appetite/pkg/event"
	"github.com/appetiteclub/apt/events"
)

func TestNewTicketStateCache(t *testing.T) {
	tests := []struct {
		name   string
		stream events.StreamConsumer
		logger bool
	}{
		{
			name:   "withNilStreamAndLogger",
			stream: nil,
			logger: false,
		},
		{
			name:   "withStream",
			stream: &MockStreamConsumer{},
			logger: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewTicketStateCache(tt.stream, nil, nil)
			if cache == nil {
				t.Fatal("NewTicketStateCache() returned nil")
			}
			if cache.tickets == nil {
				t.Error("tickets map is nil")
			}
			if cache.byStation == nil {
				t.Error("byStation index is nil")
			}
			if cache.byStatus == nil {
				t.Error("byStatus index is nil")
			}
			if cache.logger == nil {
				t.Error("logger should default to noop logger")
			}
		})
	}
}

func TestTicketStateCacheSet(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	ticket := &kitchenTicketResource{
		ID:      "ticket-1",
		Station: "grill",
		Status:  StatusCreated,
	}

	cache.Set(ticket)

	got := cache.Get("ticket-1")
	if got == nil {
		t.Fatal("Get() returned nil after Set()")
	}
	if got.ID != "ticket-1" {
		t.Errorf("ID = %q, want %q", got.ID, "ticket-1")
	}
	if got.Station != "grill" {
		t.Errorf("Station = %q, want %q", got.Station, "grill")
	}
}

func TestTicketStateCacheSetNil(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	// Should not panic
	cache.Set(nil)

	if len(cache.tickets) != 0 {
		t.Errorf("tickets count = %d, want 0", len(cache.tickets))
	}
}

func TestTicketStateCacheGet(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	tests := []struct {
		name     string
		ticketID string
		setup    func()
		wantNil  bool
	}{
		{
			name:     "existingTicket",
			ticketID: "ticket-1",
			setup: func() {
				cache.Set(&kitchenTicketResource{ID: "ticket-1"})
			},
			wantNil: false,
		},
		{
			name:     "nonExistentTicket",
			ticketID: "non-existent",
			setup:    func() {},
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got := cache.Get(tt.ticketID)
			if (got == nil) != tt.wantNil {
				t.Errorf("Get(%q) nil = %v, want %v", tt.ticketID, got == nil, tt.wantNil)
			}
		})
	}
}

func TestTicketStateCacheGetByStation(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	cache.Set(&kitchenTicketResource{ID: "ticket-1", Station: "grill", Status: StatusCreated})
	cache.Set(&kitchenTicketResource{ID: "ticket-2", Station: "grill", Status: StatusStarted})
	cache.Set(&kitchenTicketResource{ID: "ticket-3", Station: "prep", Status: StatusCreated})

	tests := []struct {
		name      string
		stationID string
		wantCount int
	}{
		{
			name:      "grillStation",
			stationID: "grill",
			wantCount: 2,
		},
		{
			name:      "prepStation",
			stationID: "prep",
			wantCount: 1,
		},
		{
			name:      "unknownStation",
			stationID: "unknown",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cache.GetByStation(tt.stationID)
			if len(got) != tt.wantCount {
				t.Errorf("GetByStation(%q) count = %d, want %d", tt.stationID, len(got), tt.wantCount)
			}
		})
	}
}

func TestTicketStateCacheGetByStatus(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	cache.Set(&kitchenTicketResource{ID: "ticket-1", Station: "grill", Status: StatusCreated})
	cache.Set(&kitchenTicketResource{ID: "ticket-2", Station: "grill", Status: StatusCreated})
	cache.Set(&kitchenTicketResource{ID: "ticket-3", Station: "prep", Status: StatusStarted})

	tests := []struct {
		name      string
		statusID  string
		wantCount int
	}{
		{
			name:      "pendingStatus",
			statusID:  StatusCreated,
			wantCount: 2,
		},
		{
			name:      "startedStatus",
			statusID:  StatusStarted,
			wantCount: 1,
		},
		{
			name:      "readyStatus",
			statusID:  StatusReady,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cache.GetByStatus(tt.statusID)
			if len(got) != tt.wantCount {
				t.Errorf("GetByStatus(%q) count = %d, want %d", tt.statusID, len(got), tt.wantCount)
			}
		})
	}
}

func TestTicketStateCacheGetByStationAndStatus(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	cache.Set(&kitchenTicketResource{ID: "ticket-1", Station: "grill", Status: StatusCreated})
	cache.Set(&kitchenTicketResource{ID: "ticket-2", Station: "grill", Status: StatusStarted})
	cache.Set(&kitchenTicketResource{ID: "ticket-3", Station: "prep", Status: StatusCreated})

	tests := []struct {
		name      string
		stationID string
		statusID  string
		wantCount int
	}{
		{
			name:      "grillPending",
			stationID: "grill",
			statusID:  StatusCreated,
			wantCount: 1,
		},
		{
			name:      "grillStarted",
			stationID: "grill",
			statusID:  StatusStarted,
			wantCount: 1,
		},
		{
			name:      "prepPending",
			stationID: "prep",
			statusID:  StatusCreated,
			wantCount: 1,
		},
		{
			name:      "prepStarted",
			stationID: "prep",
			statusID:  StatusStarted,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cache.GetByStationAndStatus(tt.stationID, tt.statusID)
			if len(got) != tt.wantCount {
				t.Errorf("GetByStationAndStatus(%q, %q) count = %d, want %d",
					tt.stationID, tt.statusID, len(got), tt.wantCount)
			}
		})
	}
}

func TestTicketStateCacheGetAll(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	if got := cache.GetAll(); len(got) != 0 {
		t.Errorf("GetAll() on empty cache = %d, want 0", len(got))
	}

	cache.Set(&kitchenTicketResource{ID: "ticket-1"})
	cache.Set(&kitchenTicketResource{ID: "ticket-2"})
	cache.Set(&kitchenTicketResource{ID: "ticket-3"})

	got := cache.GetAll()
	if len(got) != 3 {
		t.Errorf("GetAll() = %d, want 3", len(got))
	}
}

func TestTicketStateCacheRemove(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	cache.Set(&kitchenTicketResource{ID: "ticket-1", Station: "grill", Status: StatusCreated})

	if cache.Get("ticket-1") == nil {
		t.Fatal("ticket not found after Set()")
	}

	cache.Remove("ticket-1")

	if cache.Get("ticket-1") != nil {
		t.Error("ticket still found after Remove()")
	}

	// Verify indexes are cleaned up
	if len(cache.GetByStation("grill")) != 0 {
		t.Error("ticket still in station index after Remove()")
	}
	if len(cache.GetByStatus(StatusCreated)) != 0 {
		t.Error("ticket still in status index after Remove()")
	}
}

func TestTicketStateCacheRemoveNonExistent(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	// Should not panic
	cache.Remove("non-existent")
}

func TestTicketStateCacheUpdateExisting(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	// Add initial ticket
	cache.Set(&kitchenTicketResource{ID: "ticket-1", Station: "grill", Status: StatusCreated})

	// Update with new status (create new ticket object to avoid pointer issues)
	cache.Set(&kitchenTicketResource{ID: "ticket-1", Station: "grill", Status: StatusStarted})

	// Verify update
	got := cache.Get("ticket-1")
	if got.Status != StatusStarted {
		t.Errorf("Status = %q, want %q", got.Status, StatusStarted)
	}

	// Verify old status index is updated
	if len(cache.GetByStatus(StatusCreated)) != 0 {
		t.Error("old status still in index")
	}
	if len(cache.GetByStatus(StatusStarted)) != 1 {
		t.Error("new status not in index")
	}
}

func TestTicketStateCacheUpdateStationChange(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	// Add initial ticket
	cache.Set(&kitchenTicketResource{ID: "ticket-1", Station: "grill", Status: StatusCreated})

	// Move to different station (create new ticket object)
	cache.Set(&kitchenTicketResource{ID: "ticket-1", Station: "prep", Status: StatusCreated})

	// Verify old station index is cleaned
	if len(cache.GetByStation("grill")) != 0 {
		t.Error("ticket still in old station index")
	}
	if len(cache.GetByStation("prep")) != 1 {
		t.Error("ticket not in new station index")
	}
}

func TestTicketStateCacheConcurrency(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)
	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(id int) {
			defer wg.Done()
			ticketID := string(rune('a'+id%26)) + "-ticket"
			cache.Set(&kitchenTicketResource{
				ID:      ticketID,
				Station: "grill",
				Status:  StatusCreated,
			})
		}(i)
	}
	wg.Wait()

	// Concurrent reads and writes
	wg.Add(iterations * 3)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			cache.GetAll()
		}()
		go func() {
			defer wg.Done()
			cache.GetByStation("grill")
		}()
		go func() {
			defer wg.Done()
			cache.GetByStatus(StatusCreated)
		}()
	}
	wg.Wait()

	// No panics means success
}

func TestTicketStateCacheWarmFromStream(t *testing.T) {
	now := time.Now()
	createdEvent := event.KitchenTicketCreatedEvent{
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
		Status:   StatusCreated,
		Quantity: 2,
		Notes:    "no onions",
	}
	createdData, _ := json.Marshal(createdEvent)

	mockStream := &MockStreamConsumer{
		FetchFunc: func(ctx context.Context, maxMessages int) ([]events.StreamMessage, error) {
			return []events.StreamMessage{
				{Data: createdData},
			}, nil
		},
	}

	cache := NewTicketStateCache(mockStream, nil, nil)

	err := cache.Warm(context.Background())
	if err != nil {
		t.Fatalf("Warm() error = %v", err)
	}

	got := cache.Get("ticket-1")
	if got == nil {
		t.Fatal("ticket not found after Warm()")
	}
	if got.MenuItemName != "Burger" {
		t.Errorf("MenuItemName = %q, want %q", got.MenuItemName, "Burger")
	}
	if got.Station != "grill" {
		t.Errorf("Station = %q, want %q", got.Station, "grill")
	}
}

func TestTicketStateCacheWarmFromStreamWithStatusChange(t *testing.T) {
	now := time.Now()

	// Create event
	createdEvent := event.KitchenTicketCreatedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:  event.EventKitchenTicketCreated,
			OccurredAt: now,
			TicketID:   "ticket-1",
			Station:    "grill",
		},
		Status: StatusCreated,
	}
	createdData, _ := json.Marshal(createdEvent)

	// Status change event
	statusEvent := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:  event.EventKitchenTicketStatusChange,
			OccurredAt: now.Add(time.Minute),
			TicketID:   "ticket-1",
			Station:    "grill",
		},
		NewStatus: StatusStarted,
	}
	statusData, _ := json.Marshal(statusEvent)

	mockStream := &MockStreamConsumer{
		FetchFunc: func(ctx context.Context, maxMessages int) ([]events.StreamMessage, error) {
			return []events.StreamMessage{
				{Data: createdData},
				{Data: statusData},
			}, nil
		},
	}

	cache := NewTicketStateCache(mockStream, nil, nil)

	err := cache.Warm(context.Background())
	if err != nil {
		t.Fatalf("Warm() error = %v", err)
	}

	got := cache.Get("ticket-1")
	if got == nil {
		t.Fatal("ticket not found")
	}
	if got.Status != StatusStarted {
		t.Errorf("Status = %q, want %q", got.Status, StatusStarted)
	}
}

func TestTicketStateCacheWarmRemovesCompletedTickets(t *testing.T) {
	now := time.Now()

	// Create active and delivered tickets
	activeEvent := event.KitchenTicketCreatedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:  event.EventKitchenTicketCreated,
			OccurredAt: now,
			TicketID:   "active-1",
			Station:    "grill",
		},
		Status: StatusCreated,
	}
	activeData, _ := json.Marshal(activeEvent)

	deliveredEvent := event.KitchenTicketCreatedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:  event.EventKitchenTicketCreated,
			OccurredAt: now,
			TicketID:   "delivered-1",
			Station:    "grill",
		},
		Status: StatusDelivered,
	}
	deliveredData, _ := json.Marshal(deliveredEvent)

	cancelledEvent := event.KitchenTicketCreatedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:  event.EventKitchenTicketCreated,
			OccurredAt: now,
			TicketID:   "cancelled-1",
			Station:    "grill",
		},
		Status: StatusCancelled,
	}
	cancelledData, _ := json.Marshal(cancelledEvent)

	mockStream := &MockStreamConsumer{
		FetchFunc: func(ctx context.Context, maxMessages int) ([]events.StreamMessage, error) {
			return []events.StreamMessage{
				{Data: activeData},
				{Data: deliveredData},
				{Data: cancelledData},
			}, nil
		},
	}

	cache := NewTicketStateCache(mockStream, nil, nil)

	err := cache.Warm(context.Background())
	if err != nil {
		t.Fatalf("Warm() error = %v", err)
	}

	// Active ticket should exist
	if cache.Get("active-1") == nil {
		t.Error("active ticket was removed")
	}

	// Completed tickets should be removed
	if cache.Get("delivered-1") != nil {
		t.Error("delivered ticket should be removed")
	}
	if cache.Get("cancelled-1") != nil {
		t.Error("cancelled ticket should be removed")
	}
}

func TestTicketStateCacheWarmWithNoStream(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	err := cache.Warm(context.Background())
	if err != nil {
		t.Fatalf("Warm() with no stream should not error, got: %v", err)
	}

	if len(cache.GetAll()) != 0 {
		t.Error("cache should be empty when no stream available")
	}
}

func TestTicketStateCacheApplyEventUnknownType(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	unknownEvent := struct {
		EventType string `json:"event_type"`
		Data      string `json:"data"`
	}{
		EventType: "unknown.event",
		Data:      "test",
	}
	data, _ := json.Marshal(unknownEvent)

	cache.mu.Lock()
	cache.applyEventLocked(context.Background(), data)
	cache.mu.Unlock()

	// Should not add anything to cache
	if len(cache.GetAll()) != 0 {
		t.Error("unknown event should not add to cache")
	}
}

func TestTicketStateCacheApplyEventInvalidJSON(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	cache.mu.Lock()
	cache.applyEventLocked(context.Background(), []byte("invalid json"))
	cache.mu.Unlock()

	// Should not panic and should not add anything
	if len(cache.GetAll()) != 0 {
		t.Error("invalid JSON should not add to cache")
	}
}

func TestTicketStateCacheStatusChangeForNonExistentTicket(t *testing.T) {
	now := time.Now()

	statusEvent := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:    event.EventKitchenTicketStatusChange,
			OccurredAt:   now,
			TicketID:     "new-ticket",
			OrderID:      "order-1",
			OrderItemID:  "item-1",
			MenuItemID:   "menu-1",
			Station:      "grill",
			MenuItemName: "Burger",
			StationName:  "Grill Station",
			TableNumber:  "5",
		},
		NewStatus: StatusStarted,
	}
	statusData, _ := json.Marshal(statusEvent)

	mockStream := &MockStreamConsumer{
		FetchFunc: func(ctx context.Context, maxMessages int) ([]events.StreamMessage, error) {
			return []events.StreamMessage{{Data: statusData}}, nil
		},
	}

	cache := NewTicketStateCache(mockStream, nil, nil)
	err := cache.Warm(context.Background())
	if err != nil {
		t.Fatalf("Warm() error = %v", err)
	}

	// Should create minimal entry for non-existent ticket
	got := cache.Get("new-ticket")
	if got == nil {
		t.Fatal("ticket should be created from status change event")
	}
	if got.Status != StatusStarted {
		t.Errorf("Status = %q, want %q", got.Status, StatusStarted)
	}
	if got.MenuItemName != "Burger" {
		t.Errorf("MenuItemName = %q, want %q", got.MenuItemName, "Burger")
	}
}

func TestTicketStateCacheStatusChangeWithReasonCode(t *testing.T) {
	now := time.Now()

	// First create the ticket
	createdEvent := event.KitchenTicketCreatedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:  event.EventKitchenTicketCreated,
			OccurredAt: now,
			TicketID:   "ticket-1",
			Station:    "grill",
		},
		Status: StatusCreated,
	}
	createdData, _ := json.Marshal(createdEvent)

	// Status change with reason code
	statusEvent := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:  event.EventKitchenTicketStatusChange,
			OccurredAt: now.Add(time.Minute),
			TicketID:   "ticket-1",
			Station:    "grill",
		},
		NewStatus:    StatusCancelled,
		ReasonCodeID: "out-of-stock",
		Notes:        "Item unavailable",
	}
	statusData, _ := json.Marshal(statusEvent)

	mockStream := &MockStreamConsumer{
		FetchFunc: func(ctx context.Context, maxMessages int) ([]events.StreamMessage, error) {
			return []events.StreamMessage{
				{Data: createdData},
				{Data: statusData},
			}, nil
		},
	}

	cache := NewTicketStateCache(mockStream, nil, nil)
	err := cache.Warm(context.Background())
	if err != nil {
		t.Fatalf("Warm() error = %v", err)
	}

	// Ticket should be removed because it's cancelled
	if cache.Get("ticket-1") != nil {
		t.Error("cancelled ticket should be removed after warming")
	}
}

func TestTicketStateCacheIndexManagement(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, nil)

	// Add ticket
	cache.Set(&kitchenTicketResource{ID: "ticket-1", Station: "grill", Status: StatusCreated})

	// Verify in both indexes
	if len(cache.byStation["grill"]) != 1 {
		t.Errorf("byStation[grill] = %d, want 1", len(cache.byStation["grill"]))
	}
	if len(cache.byStatus[StatusCreated]) != 1 {
		t.Errorf("byStatus[pending] = %d, want 1", len(cache.byStatus[StatusCreated]))
	}

	// Add another to same station/status
	cache.Set(&kitchenTicketResource{ID: "ticket-2", Station: "grill", Status: StatusCreated})

	if len(cache.byStation["grill"]) != 2 {
		t.Errorf("byStation[grill] = %d, want 2", len(cache.byStation["grill"]))
	}
	if len(cache.byStatus[StatusCreated]) != 2 {
		t.Errorf("byStatus[pending] = %d, want 2", len(cache.byStatus[StatusCreated]))
	}

	// Remove one
	cache.Remove("ticket-1")

	if len(cache.byStation["grill"]) != 1 {
		t.Errorf("byStation[grill] after remove = %d, want 1", len(cache.byStation["grill"]))
	}
	if len(cache.byStatus[StatusCreated]) != 1 {
		t.Errorf("byStatus[pending] after remove = %d, want 1", len(cache.byStatus[StatusCreated]))
	}
}
