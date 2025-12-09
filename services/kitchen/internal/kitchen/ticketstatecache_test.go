package kitchen

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/appetiteclub/appetite/pkg/event"
	proto "github.com/appetiteclub/appetite/services/kitchen/internal/kitchen/proto"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
	"github.com/google/uuid"
)

func TestNewTicketStateCache(t *testing.T) {
	tests := []struct {
		name   string
		stream events.StreamConsumer
		repo   TicketRepository
		logger aqm.Logger
	}{
		{
			name:   "withAllDependencies",
			stream: NewMockStreamConsumer(),
			repo:   NewMockTicketRepository(),
			logger: aqm.NewNoopLogger(),
		},
		{
			name:   "withNilStream",
			stream: nil,
			repo:   NewMockTicketRepository(),
			logger: aqm.NewNoopLogger(),
		},
		{
			name:   "withNilRepo",
			stream: NewMockStreamConsumer(),
			repo:   nil,
			logger: aqm.NewNoopLogger(),
		},
		{
			name:   "withNilLogger",
			stream: NewMockStreamConsumer(),
			repo:   NewMockTicketRepository(),
			logger: nil,
		},
		{
			name:   "withAllNil",
			stream: nil,
			repo:   nil,
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewTicketStateCache(tt.stream, tt.repo, tt.logger)
			if cache == nil {
				t.Error("NewTicketStateCache() returned nil")
			}
			if cache.tickets == nil {
				t.Error("tickets map is nil")
			}
			if cache.byStation == nil {
				t.Error("byStation map is nil")
			}
			if cache.byStatus == nil {
				t.Error("byStatus map is nil")
			}
		})
	}
}

func TestTicketStateCacheSetAndGet(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	ticketID := uuid.New()
	orderID := uuid.New()
	orderItemID := uuid.New()

	ticket := &Ticket{
		ID:          ticketID,
		OrderID:     orderID,
		OrderItemID: orderItemID,
		Station:     "kitchen",
		Status:      "created",
		Quantity:    2,
	}

	cache.Set(ticket)

	got := cache.Get(ticketID)
	if got == nil {
		t.Fatal("Get() returned nil after Set()")
	}
	if got.ID != ticketID {
		t.Errorf("Get() ID = %v, want %v", got.ID, ticketID)
	}
	if got.Station != "kitchen" {
		t.Errorf("Get() Station = %v, want %v", got.Station, "kitchen")
	}
}

func TestTicketStateCacheSetNilTicket(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	// Should not panic
	cache.Set(nil)

	if cache.Count() != 0 {
		t.Errorf("Count() = %d, want 0 after setting nil ticket", cache.Count())
	}
}

func TestTicketStateCacheUpdateExisting(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	ticketID := uuid.New()
	ticket := &Ticket{
		ID:      ticketID,
		Station: "kitchen",
		Status:  "created",
	}

	cache.Set(ticket)

	// Create a new ticket object with updated values (simulating what happens in real code)
	updatedTicket := &Ticket{
		ID:      ticketID,
		Station: "bar",
		Status:  "started",
	}
	cache.Set(updatedTicket)

	got := cache.Get(ticketID)
	if got.Status != "started" {
		t.Errorf("Updated ticket Status = %v, want %v", got.Status, "started")
	}
	if got.Station != "bar" {
		t.Errorf("Updated ticket Station = %v, want %v", got.Station, "bar")
	}

	// Verify old index entries are cleaned up
	kitchenTickets := cache.GetByStationCode("kitchen")
	for _, tk := range kitchenTickets {
		if tk.ID == ticketID {
			t.Error("Ticket still indexed under old station 'kitchen'")
		}
	}
}

func TestTicketStateCacheGetByStationCode(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	ticket1 := &Ticket{ID: uuid.New(), Station: "kitchen", Status: "created"}
	ticket2 := &Ticket{ID: uuid.New(), Station: "kitchen", Status: "started"}
	ticket3 := &Ticket{ID: uuid.New(), Station: "bar", Status: "created"}

	cache.Set(ticket1)
	cache.Set(ticket2)
	cache.Set(ticket3)

	kitchenTickets := cache.GetByStationCode("kitchen")
	if len(kitchenTickets) != 2 {
		t.Errorf("GetByStationCode('kitchen') returned %d tickets, want 2", len(kitchenTickets))
	}

	barTickets := cache.GetByStationCode("bar")
	if len(barTickets) != 1 {
		t.Errorf("GetByStationCode('bar') returned %d tickets, want 1", len(barTickets))
	}

	emptyTickets := cache.GetByStationCode("nonexistent")
	if len(emptyTickets) != 0 {
		t.Errorf("GetByStationCode('nonexistent') returned %d tickets, want 0", len(emptyTickets))
	}
}

func TestTicketStateCacheGetByStatusCode(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	ticket1 := &Ticket{ID: uuid.New(), Station: "kitchen", Status: "created"}
	ticket2 := &Ticket{ID: uuid.New(), Station: "bar", Status: "created"}
	ticket3 := &Ticket{ID: uuid.New(), Station: "kitchen", Status: "started"}

	cache.Set(ticket1)
	cache.Set(ticket2)
	cache.Set(ticket3)

	createdTickets := cache.GetByStatusCode("created")
	if len(createdTickets) != 2 {
		t.Errorf("GetByStatusCode('created') returned %d tickets, want 2", len(createdTickets))
	}

	startedTickets := cache.GetByStatusCode("started")
	if len(startedTickets) != 1 {
		t.Errorf("GetByStatusCode('started') returned %d tickets, want 1", len(startedTickets))
	}
}

func TestTicketStateCacheGetByStationAndStatusCode(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	ticket1 := &Ticket{ID: uuid.New(), Station: "kitchen", Status: "created"}
	ticket2 := &Ticket{ID: uuid.New(), Station: "kitchen", Status: "started"}
	ticket3 := &Ticket{ID: uuid.New(), Station: "bar", Status: "created"}

	cache.Set(ticket1)
	cache.Set(ticket2)
	cache.Set(ticket3)

	tickets := cache.GetByStationAndStatusCode("kitchen", "created")
	if len(tickets) != 1 {
		t.Errorf("GetByStationAndStatusCode('kitchen', 'created') returned %d tickets, want 1", len(tickets))
	}

	noTickets := cache.GetByStationAndStatusCode("bar", "started")
	if len(noTickets) != 0 {
		t.Errorf("GetByStationAndStatusCode('bar', 'started') returned %d tickets, want 0", len(noTickets))
	}
}

func TestTicketStateCacheGetAll(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	ticket1 := &Ticket{ID: uuid.New(), Station: "kitchen", Status: "created"}
	ticket2 := &Ticket{ID: uuid.New(), Station: "bar", Status: "started"}

	cache.Set(ticket1)
	cache.Set(ticket2)

	all := cache.GetAll()
	if len(all) != 2 {
		t.Errorf("GetAll() returned %d tickets, want 2", len(all))
	}
}

func TestTicketStateCacheRemove(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	ticketID := uuid.New()
	ticket := &Ticket{ID: ticketID, Station: "kitchen", Status: "created"}

	cache.Set(ticket)
	if cache.Count() != 1 {
		t.Errorf("Count() after Set = %d, want 1", cache.Count())
	}

	cache.Remove(ticketID)
	if cache.Count() != 0 {
		t.Errorf("Count() after Remove = %d, want 0", cache.Count())
	}

	if got := cache.Get(ticketID); got != nil {
		t.Error("Get() returned ticket after Remove()")
	}

	// Verify indexes are cleaned up
	kitchenTickets := cache.GetByStationCode("kitchen")
	if len(kitchenTickets) != 0 {
		t.Errorf("GetByStationCode('kitchen') after Remove = %d, want 0", len(kitchenTickets))
	}
}

func TestTicketStateCacheRemoveNonexistent(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	// Should not panic
	cache.Remove(uuid.New())
}

func TestTicketStateCacheCount(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	if cache.Count() != 0 {
		t.Errorf("Empty cache Count() = %d, want 0", cache.Count())
	}

	cache.Set(&Ticket{ID: uuid.New(), Station: "kitchen", Status: "created"})
	if cache.Count() != 1 {
		t.Errorf("Count() after 1 ticket = %d, want 1", cache.Count())
	}

	cache.Set(&Ticket{ID: uuid.New(), Station: "bar", Status: "created"})
	if cache.Count() != 2 {
		t.Errorf("Count() after 2 tickets = %d, want 2", cache.Count())
	}
}

func TestTicketStateCacheSetStreamServer(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	server := NewEventStreamServer(cache, aqm.NewNoopLogger())
	cache.SetStreamServer(server)

	if cache.streamServer != server {
		t.Error("SetStreamServer() did not set the server")
	}
}

func TestTicketStateCacheWarmFromStream(t *testing.T) {
	mockStream := NewMockStreamConsumer()

	// Add a ticket created event
	ticketID := uuid.New()
	orderID := uuid.New()
	orderItemID := uuid.New()
	menuItemID := uuid.New()

	createdEvent := event.KitchenTicketCreatedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:    event.EventKitchenTicketCreated,
			OccurredAt:   time.Now(),
			TicketID:     ticketID.String(),
			OrderID:      orderID.String(),
			OrderItemID:  orderItemID.String(),
			MenuItemID:   menuItemID.String(),
			Station:      "kitchen",
			MenuItemName: "Burger",
			StationName:  "Kitchen",
			TableNumber:  "T1",
		},
		Status:   "created",
		Quantity: 2,
		Notes:    "No onions",
	}
	eventBytes, _ := json.Marshal(createdEvent)
	mockStream.AddMessage(eventBytes)

	cache := NewTicketStateCache(mockStream, nil, aqm.NewNoopLogger())

	err := cache.Warm(context.Background())
	if err != nil {
		t.Fatalf("Warm() error = %v", err)
	}

	if cache.Count() != 1 {
		t.Errorf("Count() after Warm = %d, want 1", cache.Count())
	}

	ticket := cache.Get(ticketID)
	if ticket == nil {
		t.Fatal("Ticket not found after warming from stream")
	}
	if ticket.Station != "kitchen" {
		t.Errorf("Ticket Station = %v, want 'kitchen'", ticket.Station)
	}
	if ticket.MenuItemName != "Burger" {
		t.Errorf("Ticket MenuItemName = %v, want 'Burger'", ticket.MenuItemName)
	}
}

func TestTicketStateCacheWarmFromStreamWithStatusChange(t *testing.T) {
	mockStream := NewMockStreamConsumer()

	ticketID := uuid.New()
	orderID := uuid.New()
	orderItemID := uuid.New()
	menuItemID := uuid.New()

	// First event: create
	createdEvent := event.KitchenTicketCreatedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:   event.EventKitchenTicketCreated,
			OccurredAt:  time.Now(),
			TicketID:    ticketID.String(),
			OrderID:     orderID.String(),
			OrderItemID: orderItemID.String(),
			MenuItemID:  menuItemID.String(),
			Station:     "kitchen",
		},
		Status: "created",
	}
	createdBytes, _ := json.Marshal(createdEvent)
	mockStream.AddMessage(createdBytes)

	// Second event: status change
	now := time.Now()
	statusEvent := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:   event.EventKitchenTicketStatusChange,
			OccurredAt:  now,
			TicketID:    ticketID.String(),
			OrderID:     orderID.String(),
			OrderItemID: orderItemID.String(),
			MenuItemID:  menuItemID.String(),
			Station:     "kitchen",
		},
		NewStatus:      "started",
		PreviousStatus: "created",
		StartedAt:      &now,
	}
	statusBytes, _ := json.Marshal(statusEvent)
	mockStream.AddMessage(statusBytes)

	cache := NewTicketStateCache(mockStream, nil, aqm.NewNoopLogger())
	cache.Warm(context.Background())

	ticket := cache.Get(ticketID)
	if ticket == nil {
		t.Fatal("Ticket not found after warming")
	}
	if ticket.Status != "started" {
		t.Errorf("Ticket Status = %v, want 'started'", ticket.Status)
	}
	if ticket.StartedAt == nil {
		t.Error("Ticket StartedAt is nil after status change")
	}
}

func TestTicketStateCacheWarmFromStreamRemovesDelivered(t *testing.T) {
	mockStream := NewMockStreamConsumer()

	// Create two tickets
	ticket1ID := uuid.New()
	ticket2ID := uuid.New()

	// Ticket 1: created status (should remain)
	createdEvent1 := event.KitchenTicketCreatedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType: event.EventKitchenTicketCreated,
			TicketID:  ticket1ID.String(),
			Station:   "kitchen",
		},
		Status: "created",
	}
	bytes1, _ := json.Marshal(createdEvent1)
	mockStream.AddMessage(bytes1)

	// Ticket 2: delivered status (should be removed)
	createdEvent2 := event.KitchenTicketCreatedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType: event.EventKitchenTicketCreated,
			TicketID:  ticket2ID.String(),
			Station:   "kitchen",
		},
		Status: "delivered",
	}
	bytes2, _ := json.Marshal(createdEvent2)
	mockStream.AddMessage(bytes2)

	cache := NewTicketStateCache(mockStream, nil, aqm.NewNoopLogger())
	cache.Warm(context.Background())

	if cache.Count() != 1 {
		t.Errorf("Count() = %d, want 1 (delivered should be removed)", cache.Count())
	}

	if cache.Get(ticket1ID) == nil {
		t.Error("Non-delivered ticket was incorrectly removed")
	}
	if cache.Get(ticket2ID) != nil {
		t.Error("Delivered ticket was not removed")
	}
}

func TestTicketStateCacheWarmFromRepo(t *testing.T) {
	mockRepo := NewMockTicketRepository()

	ticket1 := &Ticket{
		ID:      uuid.New(),
		Station: "kitchen",
		Status:  "created",
	}
	ticket2 := &Ticket{
		ID:      uuid.New(),
		Station: "bar",
		Status:  "started",
	}
	mockRepo.AddTicket(ticket1)
	mockRepo.AddTicket(ticket2)

	cache := NewTicketStateCache(nil, mockRepo, aqm.NewNoopLogger())

	err := cache.Warm(context.Background())
	if err != nil {
		t.Fatalf("Warm() error = %v", err)
	}

	if cache.Count() != 2 {
		t.Errorf("Count() after Warm from repo = %d, want 2", cache.Count())
	}
}

func TestTicketStateCacheWarmFallbackToRepo(t *testing.T) {
	// Stream that fails
	mockStream := NewMockStreamConsumer()
	mockStream.FetchFunc = func(ctx context.Context, maxMessages int) ([]events.StreamMessage, error) {
		return nil, errors.New("stream error")
	}

	mockRepo := NewMockTicketRepository()
	ticket := &Ticket{ID: uuid.New(), Station: "kitchen", Status: "created"}
	mockRepo.AddTicket(ticket)

	cache := NewTicketStateCache(mockStream, mockRepo, aqm.NewNoopLogger())

	err := cache.Warm(context.Background())
	if err != nil {
		t.Fatalf("Warm() should not error when falling back to repo: %v", err)
	}

	if cache.Count() != 1 {
		t.Errorf("Count() after fallback = %d, want 1", cache.Count())
	}
}

func TestTicketStateCacheWarmWithNeitherStreamNorRepo(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	err := cache.Warm(context.Background())
	if err != nil {
		t.Fatalf("Warm() should not error with neither stream nor repo: %v", err)
	}

	if cache.Count() != 0 {
		t.Errorf("Count() = %d, want 0", cache.Count())
	}
}

func TestTicketStateCacheWarmFromRepoError(t *testing.T) {
	mockRepo := NewMockTicketRepository()
	mockRepo.ListFunc = func(ctx context.Context, filter TicketFilter) ([]Ticket, error) {
		return nil, errors.New("database error")
	}

	cache := NewTicketStateCache(nil, mockRepo, aqm.NewNoopLogger())

	// Should not return error - just leave cache empty
	err := cache.Warm(context.Background())
	if err != nil {
		t.Fatalf("Warm() returned error: %v", err)
	}

	if cache.Count() != 0 {
		t.Errorf("Count() = %d, want 0 after repo error", cache.Count())
	}
}

func TestTicketStateCacheWarmFromRepoPublic(t *testing.T) {
	mockRepo := NewMockTicketRepository()
	ticket := &Ticket{ID: uuid.New(), Station: "kitchen", Status: "created"}
	mockRepo.AddTicket(ticket)

	cache := NewTicketStateCache(nil, mockRepo, aqm.NewNoopLogger())

	err := cache.WarmFromRepo(context.Background())
	if err != nil {
		t.Fatalf("WarmFromRepo() error = %v", err)
	}

	if cache.Count() != 1 {
		t.Errorf("Count() = %d, want 1", cache.Count())
	}
}

func TestTicketStateCacheConcurrency(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ticket := &Ticket{
				ID:      uuid.New(),
				Station: "kitchen",
				Status:  "created",
			}
			cache.Set(ticket)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.GetAll()
			cache.GetByStationCode("kitchen")
			cache.GetByStatusCode("created")
			cache.Count()
		}()
	}

	wg.Wait()

	if cache.Count() != numGoroutines {
		t.Errorf("Count() = %d, want %d after concurrent operations", cache.Count(), numGoroutines)
	}
}

func TestTicketStateCacheApplyEventUnknownType(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	unknownEvent := map[string]string{
		"event_type": "unknown.event",
	}
	eventBytes, _ := json.Marshal(unknownEvent)

	// Should not panic or error
	cache.mu.Lock()
	cache.applyEventLocked(context.Background(), eventBytes)
	cache.mu.Unlock()

	if cache.Count() != 0 {
		t.Errorf("Count() = %d, want 0 for unknown event type", cache.Count())
	}
}

func TestTicketStateCacheApplyEventInvalidJSON(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	// Should not panic
	cache.mu.Lock()
	cache.applyEventLocked(context.Background(), []byte("invalid json"))
	cache.mu.Unlock()

	if cache.Count() != 0 {
		t.Errorf("Count() = %d, want 0 for invalid JSON", cache.Count())
	}
}

func TestTicketStateCacheStatusChangeForNonexistentTicket(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	// Status change for a ticket that doesn't exist in cache
	ticketID := uuid.New()
	statusEvent := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:    event.EventKitchenTicketStatusChange,
			TicketID:     ticketID.String(),
			OrderID:      uuid.New().String(),
			OrderItemID:  uuid.New().String(),
			MenuItemID:   uuid.New().String(),
			Station:      "kitchen",
			MenuItemName: "Pizza",
			StationName:  "Kitchen",
			TableNumber:  "T5",
		},
		NewStatus: "started",
	}
	eventBytes, _ := json.Marshal(statusEvent)

	cache.mu.Lock()
	cache.applyEventLocked(context.Background(), eventBytes)
	cache.mu.Unlock()

	// Should create a minimal ticket entry
	if cache.Count() != 1 {
		t.Errorf("Count() = %d, want 1 (minimal ticket should be created)", cache.Count())
	}

	ticket := cache.Get(ticketID)
	if ticket == nil {
		t.Fatal("Ticket should be created from status change event")
	}
	if ticket.Status != "started" {
		t.Errorf("Ticket Status = %v, want 'started'", ticket.Status)
	}
	if ticket.MenuItemName != "Pizza" {
		t.Errorf("Ticket MenuItemName = %v, want 'Pizza'", ticket.MenuItemName)
	}
}

func TestTicketStateCacheStatusChangeWithReasonCode(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	ticketID := uuid.New()
	reasonCodeID := uuid.New()

	// First create the ticket
	ticket := &Ticket{ID: ticketID, Station: "kitchen", Status: "created"}
	cache.Set(ticket)

	// Then update with reason code
	statusEvent := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType: event.EventKitchenTicketStatusChange,
			TicketID:  ticketID.String(),
			Station:   "kitchen",
		},
		NewStatus:    "blocked",
		ReasonCodeID: reasonCodeID.String(),
	}
	eventBytes, _ := json.Marshal(statusEvent)

	cache.mu.Lock()
	cache.applyEventLocked(context.Background(), eventBytes)
	cache.mu.Unlock()

	updated := cache.Get(ticketID)
	if updated.ReasonCodeID == nil {
		t.Error("ReasonCodeID should be set after status change")
	}
	if *updated.ReasonCodeID != reasonCodeID {
		t.Errorf("ReasonCodeID = %v, want %v", *updated.ReasonCodeID, reasonCodeID)
	}
}

func TestTicketStateCacheSetBroadcastsToStreamServer(t *testing.T) {
	cache := NewTicketStateCache(nil, nil, aqm.NewNoopLogger())

	// Create a stream server and add a subscriber channel
	server := NewEventStreamServer(cache, aqm.NewNoopLogger())
	cache.SetStreamServer(server)

	testChan := make(chan *proto.KitchenTicketEvent, 10)
	server.mu.Lock()
	server.subscribers["test"] = testChan
	server.mu.Unlock()

	// Set a ticket - should broadcast
	ticket := &Ticket{
		ID:           uuid.New(),
		OrderID:      uuid.New(),
		OrderItemID:  uuid.New(),
		MenuItemID:   uuid.New(),
		Station:      "kitchen",
		Status:       "started",
		MenuItemName: "Burger",
		UpdatedAt:    time.Now(),
	}
	cache.Set(ticket)

	// Check that event was broadcast
	select {
	case evt := <-testChan:
		if evt.NewStatusId != "started" {
			t.Errorf("Broadcast event NewStatusId = %v, want 'started'", evt.NewStatusId)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected broadcast event was not received")
	}
}

