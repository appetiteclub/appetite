package events

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/appetiteclub/appetite/pkg/enums/kitchenstatus"
	"github.com/appetiteclub/appetite/pkg/event"
	"github.com/appetiteclub/appetite/services/kitchen/internal/kitchen"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
	"github.com/google/uuid"
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

// MockTicketRepo implements kitchen.TicketRepository for testing
type MockTicketRepo struct {
	tickets               map[uuid.UUID]*kitchen.Ticket
	byOrderItemID         map[uuid.UUID]*kitchen.Ticket
	CreateFunc            func(ctx context.Context, t *kitchen.Ticket) error
	UpdateFunc            func(ctx context.Context, t *kitchen.Ticket) error
	FindByIDFunc          func(ctx context.Context, id kitchen.TicketID) (*kitchen.Ticket, error)
	FindByOrderItemIDFunc func(ctx context.Context, id kitchen.OrderItemID) (*kitchen.Ticket, error)
	ListFunc              func(ctx context.Context, filter kitchen.TicketFilter) ([]kitchen.Ticket, error)
}

func NewMockTicketRepo() *MockTicketRepo {
	return &MockTicketRepo{
		tickets:       make(map[uuid.UUID]*kitchen.Ticket),
		byOrderItemID: make(map[uuid.UUID]*kitchen.Ticket),
	}
}

func (m *MockTicketRepo) Create(ctx context.Context, t *kitchen.Ticket) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, t)
	}
	m.tickets[t.ID] = t
	m.byOrderItemID[t.OrderItemID] = t
	return nil
}

func (m *MockTicketRepo) Update(ctx context.Context, t *kitchen.Ticket) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, t)
	}
	if _, exists := m.tickets[t.ID]; !exists {
		return errors.New("ticket not found")
	}
	m.tickets[t.ID] = t
	return nil
}

func (m *MockTicketRepo) FindByID(ctx context.Context, id kitchen.TicketID) (*kitchen.Ticket, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	t, exists := m.tickets[id]
	if !exists {
		return nil, errors.New("ticket not found")
	}
	return t, nil
}

func (m *MockTicketRepo) FindByOrderItemID(ctx context.Context, id kitchen.OrderItemID) (*kitchen.Ticket, error) {
	if m.FindByOrderItemIDFunc != nil {
		return m.FindByOrderItemIDFunc(ctx, id)
	}
	return m.byOrderItemID[id], nil
}

func (m *MockTicketRepo) List(ctx context.Context, filter kitchen.TicketFilter) ([]kitchen.Ticket, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, filter)
	}
	result := make([]kitchen.Ticket, 0, len(m.tickets))
	for _, t := range m.tickets {
		result = append(result, *t)
	}
	return result, nil
}

func (m *MockTicketRepo) AddTicket(t *kitchen.Ticket) {
	m.tickets[t.ID] = t
	m.byOrderItemID[t.OrderItemID] = t
}

// MockPublisher implements events.Publisher for testing
type MockPublisher struct {
	PublishedEvents []struct {
		Topic string
		Data  []byte
	}
	PublishFunc func(ctx context.Context, topic string, data []byte) error
}

func NewMockPublisher() *MockPublisher {
	return &MockPublisher{}
}

func (m *MockPublisher) Publish(ctx context.Context, topic string, data []byte) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, topic, data)
	}
	m.PublishedEvents = append(m.PublishedEvents, struct {
		Topic string
		Data  []byte
	}{topic, data})
	return nil
}

func TestNewOrderItemSubscriber(t *testing.T) {
	subscriber := &MockSubscriber{}
	repo := NewMockTicketRepo()
	cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	publisher := NewMockPublisher()
	logger := aqm.NewNoopLogger()

	s := NewOrderItemSubscriber(subscriber, repo, cache, publisher, logger)
	if s == nil {
		t.Error("NewOrderItemSubscriber() returned nil")
	}
}

func TestOrderItemSubscriberStart(t *testing.T) {
	tests := []struct {
		name          string
		subscribeFunc func(ctx context.Context, topic string, handler events.HandlerFunc) error
		wantErr       bool
	}{
		{
			name: "success",
			subscribeFunc: func(ctx context.Context, topic string, handler events.HandlerFunc) error {
				if topic != "orders.items" {
					t.Errorf("Subscribe topic = %v, want 'orders.items'", topic)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "subscribeError",
			subscribeFunc: func(ctx context.Context, topic string, handler events.HandlerFunc) error {
				return errors.New("subscription failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscriber := &MockSubscriber{SubscribeFunc: tt.subscribeFunc}
			repo := NewMockTicketRepo()
			cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
			publisher := NewMockPublisher()

			s := NewOrderItemSubscriber(subscriber, repo, cache, publisher, aqm.NewNoopLogger())
			err := s.Start(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderItemSubscriberHandleCreated(t *testing.T) {
	tests := []struct {
		name           string
		evt            event.OrderItemEvent
		setupRepo      func(*MockTicketRepo)
		wantTicket     bool
		wantErr        bool
		wantPublish    bool
	}{
		{
			name: "successCreateTicket",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemCreated,
				OrderItemID:        uuid.New().String(),
				OrderID:            uuid.New().String(),
				MenuItemID:         uuid.New().String(),
				RequiresProduction: true,
				ProductionStation:  "kitchen",
				Quantity:           2,
				Notes:              "No onions",
				MenuItemName:       "Burger",
				StationName:        "Kitchen",
				TableNumber:        "T1",
			},
			wantTicket:  true,
			wantPublish: true,
		},
		{
			name: "skipNonProductionItem",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemCreated,
				OrderItemID:        uuid.New().String(),
				RequiresProduction: false,
			},
			wantTicket:  false,
			wantPublish: false,
		},
		{
			name: "skipExistingTicket",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemCreated,
				OrderItemID:        "11111111-1111-1111-1111-111111111111",
				OrderID:            uuid.New().String(),
				MenuItemID:         uuid.New().String(),
				RequiresProduction: true,
			},
			setupRepo: func(r *MockTicketRepo) {
				orderItemID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
				r.AddTicket(&kitchen.Ticket{
					ID:          uuid.New(),
					OrderItemID: orderItemID,
				})
			},
			wantTicket:  false,
			wantPublish: false,
		},
		{
			name: "invalidOrderItemID",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemCreated,
				OrderItemID:        "invalid-uuid",
				RequiresProduction: true,
			},
			wantTicket: false,
			wantErr:    false,
		},
		{
			name: "invalidOrderID",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemCreated,
				OrderItemID:        uuid.New().String(),
				OrderID:            "invalid-uuid",
				RequiresProduction: true,
			},
			wantTicket: false,
		},
		{
			name: "invalidMenuItemID",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemCreated,
				OrderItemID:        uuid.New().String(),
				OrderID:            uuid.New().String(),
				MenuItemID:         "invalid-uuid",
				RequiresProduction: true,
			},
			wantTicket: false,
		},
		{
			name: "repoCreateError",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemCreated,
				OrderItemID:        uuid.New().String(),
				OrderID:            uuid.New().String(),
				MenuItemID:         uuid.New().String(),
				RequiresProduction: true,
			},
			setupRepo: func(r *MockTicketRepo) {
				r.CreateFunc = func(ctx context.Context, t *kitchen.Ticket) error {
					return errors.New("create error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepo()
			cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
			publisher := NewMockPublisher()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			s := NewOrderItemSubscriber(&MockSubscriber{}, repo, cache, publisher, aqm.NewNoopLogger())

			eventBytes, _ := json.Marshal(tt.evt)
			err := s.handleEvent(context.Background(), eventBytes)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleEvent() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantPublish && len(publisher.PublishedEvents) == 0 {
				t.Error("Expected event to be published")
			}
		})
	}
}

func TestOrderItemSubscriberHandleUpdated(t *testing.T) {
	ticketID := uuid.New()
	orderItemID := uuid.New()

	tests := []struct {
		name      string
		evt       event.OrderItemEvent
		setupRepo func(*MockTicketRepo)
		wantErr   bool
	}{
		{
			name: "successUpdateTicket",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemUpdated,
				OrderItemID:        orderItemID.String(),
				RequiresProduction: true,
				Quantity:           5,
				Notes:              "Updated notes",
			},
			setupRepo: func(r *MockTicketRepo) {
				r.AddTicket(&kitchen.Ticket{
					ID:          ticketID,
					OrderItemID: orderItemID,
					Quantity:    2,
					Notes:       "Original",
				})
			},
		},
		{
			name: "skipNonProductionItem",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemUpdated,
				OrderItemID:        uuid.New().String(),
				RequiresProduction: false,
			},
		},
		{
			name: "invalidOrderItemID",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemUpdated,
				OrderItemID:        "invalid",
				RequiresProduction: true,
			},
		},
		{
			name: "ticketNotFound",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemUpdated,
				OrderItemID:        uuid.New().String(),
				RequiresProduction: true,
			},
		},
		{
			name: "repoUpdateError",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemUpdated,
				OrderItemID:        orderItemID.String(),
				RequiresProduction: true,
			},
			setupRepo: func(r *MockTicketRepo) {
				r.AddTicket(&kitchen.Ticket{
					ID:          ticketID,
					OrderItemID: orderItemID,
				})
				r.UpdateFunc = func(ctx context.Context, t *kitchen.Ticket) error {
					return errors.New("update error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepo()
			cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
			publisher := NewMockPublisher()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			s := NewOrderItemSubscriber(&MockSubscriber{}, repo, cache, publisher, aqm.NewNoopLogger())

			eventBytes, _ := json.Marshal(tt.evt)
			err := s.handleEvent(context.Background(), eventBytes)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderItemSubscriberHandleCancelled(t *testing.T) {
	ticketID := uuid.New()
	orderItemID := uuid.New()

	tests := []struct {
		name        string
		evt         event.OrderItemEvent
		setupRepo   func(*MockTicketRepo)
		wantErr     bool
		wantPublish bool
	}{
		{
			name: "successCancelTicket",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemCancelled,
				OrderItemID:        orderItemID.String(),
				RequiresProduction: true,
			},
			setupRepo: func(r *MockTicketRepo) {
				r.AddTicket(&kitchen.Ticket{
					ID:          ticketID,
					OrderItemID: orderItemID,
					OrderID:     uuid.New(),
					MenuItemID:  uuid.New(),
					Status:      "started",
				})
			},
			wantPublish: true,
		},
		{
			name: "skipNonProductionItem",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemCancelled,
				OrderItemID:        uuid.New().String(),
				RequiresProduction: false,
			},
		},
		{
			name: "ticketNotFound",
			evt: event.OrderItemEvent{
				EventType:          event.EventOrderItemCancelled,
				OrderItemID:        uuid.New().String(),
				RequiresProduction: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepo()
			cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
			publisher := NewMockPublisher()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			s := NewOrderItemSubscriber(&MockSubscriber{}, repo, cache, publisher, aqm.NewNoopLogger())

			eventBytes, _ := json.Marshal(tt.evt)
			err := s.handleEvent(context.Background(), eventBytes)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleEvent() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantPublish && len(publisher.PublishedEvents) == 0 {
				t.Error("Expected status change event to be published")
			}
		})
	}
}

func TestOrderItemSubscriberHandleStatusChanged(t *testing.T) {
	ticketID := uuid.New()
	orderItemID := uuid.New()

	tests := []struct {
		name        string
		evt         event.OrderItemEvent
		setupRepo   func(*MockTicketRepo)
		wantStatus  string
		wantPublish bool
	}{
		{
			name: "statusChangedToDelivered",
			evt: event.OrderItemEvent{
				EventType:          "order.item.status_changed",
				OrderItemID:        orderItemID.String(),
				RequiresProduction: true,
				Status:             "delivered",
			},
			setupRepo: func(r *MockTicketRepo) {
				r.AddTicket(&kitchen.Ticket{
					ID:          ticketID,
					OrderItemID: orderItemID,
					OrderID:     uuid.New(),
					MenuItemID:  uuid.New(),
					Status:      "ready",
				})
			},
			wantStatus:  kitchenstatus.Statuses.Delivered.Code(),
			wantPublish: true,
		},
		{
			name: "statusChangedToCancelled",
			evt: event.OrderItemEvent{
				EventType:          "order.item.status_changed",
				OrderItemID:        orderItemID.String(),
				RequiresProduction: true,
				Status:             "cancelled",
			},
			setupRepo: func(r *MockTicketRepo) {
				r.AddTicket(&kitchen.Ticket{
					ID:          ticketID,
					OrderItemID: orderItemID,
					OrderID:     uuid.New(),
					MenuItemID:  uuid.New(),
					Status:      "started",
				})
			},
			wantStatus:  kitchenstatus.Statuses.Cancelled.Code(),
			wantPublish: true,
		},
		{
			name: "unmappedStatus",
			evt: event.OrderItemEvent{
				EventType:          "order.item.status_changed",
				OrderItemID:        orderItemID.String(),
				RequiresProduction: true,
				Status:             "pending",
			},
			setupRepo: func(r *MockTicketRepo) {
				r.AddTicket(&kitchen.Ticket{
					ID:          ticketID,
					OrderItemID: orderItemID,
					Status:      "created",
				})
			},
			wantPublish: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepo()
			cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
			publisher := NewMockPublisher()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			s := NewOrderItemSubscriber(&MockSubscriber{}, repo, cache, publisher, aqm.NewNoopLogger())

			eventBytes, _ := json.Marshal(tt.evt)
			s.handleEvent(context.Background(), eventBytes)

			if tt.wantStatus != "" {
				ticket := cache.Get(ticketID)
				if ticket != nil && ticket.Status != tt.wantStatus {
					t.Errorf("ticket status = %v, want %v", ticket.Status, tt.wantStatus)
				}
			}

			if tt.wantPublish && len(publisher.PublishedEvents) == 0 {
				t.Error("Expected event to be published")
			}
		})
	}
}

func TestOrderItemSubscriberHandleUnknownEventType(t *testing.T) {
	repo := NewMockTicketRepo()
	cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	publisher := NewMockPublisher()

	s := NewOrderItemSubscriber(&MockSubscriber{}, repo, cache, publisher, aqm.NewNoopLogger())

	evt := event.OrderItemEvent{
		EventType:          "unknown.event.type",
		OrderItemID:        uuid.New().String(),
		RequiresProduction: true,
	}
	eventBytes, _ := json.Marshal(evt)

	// Should not error on unknown event type
	err := s.handleEvent(context.Background(), eventBytes)
	if err != nil {
		t.Errorf("handleEvent() should not error on unknown event type: %v", err)
	}
}

func TestOrderItemSubscriberHandleInvalidJSON(t *testing.T) {
	repo := NewMockTicketRepo()
	cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	publisher := NewMockPublisher()

	s := NewOrderItemSubscriber(&MockSubscriber{}, repo, cache, publisher, aqm.NewNoopLogger())

	// Should not return error - just logs and continues
	err := s.handleEvent(context.Background(), []byte("invalid json"))
	if err != nil {
		t.Errorf("handleEvent() should not return error for invalid JSON: %v", err)
	}
}

func TestOrderItemSubscriberCacheUpdate(t *testing.T) {
	orderItemID := uuid.New()

	repo := NewMockTicketRepo()
	cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	publisher := NewMockPublisher()

	s := NewOrderItemSubscriber(&MockSubscriber{}, repo, cache, publisher, aqm.NewNoopLogger())

	evt := event.OrderItemEvent{
		EventType:          event.EventOrderItemCreated,
		OrderItemID:        orderItemID.String(),
		OrderID:            uuid.New().String(),
		MenuItemID:         uuid.New().String(),
		RequiresProduction: true,
		ProductionStation:  "kitchen",
		Quantity:           3,
	}
	eventBytes, _ := json.Marshal(evt)

	s.handleEvent(context.Background(), eventBytes)

	// Verify ticket was added to cache
	// The ticket ID is generated inside handleCreated, so we can't directly check by ID
	// Instead, check that a ticket with the correct orderItemID exists
	allTickets := cache.GetAll()
	found := false
	for _, ticket := range allTickets {
		if ticket.OrderItemID == orderItemID {
			found = true
			if ticket.Quantity != 3 {
				t.Errorf("ticket quantity = %d, want 3", ticket.Quantity)
			}
			if ticket.Station != "kitchen" {
				t.Errorf("ticket station = %v, want 'kitchen'", ticket.Station)
			}
		}
	}
	if !found {
		t.Error("Ticket not found in cache after creation")
	}
}

func TestOrderItemSubscriberPublishError(t *testing.T) {
	orderItemID := uuid.New()

	repo := NewMockTicketRepo()
	cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	publisher := NewMockPublisher()
	publisher.PublishFunc = func(ctx context.Context, topic string, data []byte) error {
		return errors.New("publish error")
	}

	s := NewOrderItemSubscriber(&MockSubscriber{}, repo, cache, publisher, aqm.NewNoopLogger())

	evt := event.OrderItemEvent{
		EventType:          event.EventOrderItemCreated,
		OrderItemID:        orderItemID.String(),
		OrderID:            uuid.New().String(),
		MenuItemID:         uuid.New().String(),
		RequiresProduction: true,
	}
	eventBytes, _ := json.Marshal(evt)

	// Should not return error - publish error is logged but not propagated
	err := s.handleEvent(context.Background(), eventBytes)
	if err != nil {
		t.Errorf("handleEvent() should not return error for publish failure: %v", err)
	}
}

func TestOrderItemSubscriberFindByOrderItemIDError(t *testing.T) {
	repo := NewMockTicketRepo()
	repo.FindByOrderItemIDFunc = func(ctx context.Context, id kitchen.OrderItemID) (*kitchen.Ticket, error) {
		return nil, errors.New("database error")
	}

	cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	publisher := NewMockPublisher()

	s := NewOrderItemSubscriber(&MockSubscriber{}, repo, cache, publisher, aqm.NewNoopLogger())

	evt := event.OrderItemEvent{
		EventType:          event.EventOrderItemCreated,
		OrderItemID:        uuid.New().String(),
		OrderID:            uuid.New().String(),
		MenuItemID:         uuid.New().String(),
		RequiresProduction: true,
	}
	eventBytes, _ := json.Marshal(evt)

	err := s.handleEvent(context.Background(), eventBytes)
	if err == nil {
		t.Error("handleEvent() should return error when FindByOrderItemID fails")
	}
}

func TestOrderItemSubscriberNilCache(t *testing.T) {
	orderItemID := uuid.New()

	repo := NewMockTicketRepo()
	publisher := NewMockPublisher()

	// Create subscriber with nil cache
	s := NewOrderItemSubscriber(&MockSubscriber{}, repo, nil, publisher, aqm.NewNoopLogger())

	evt := event.OrderItemEvent{
		EventType:          event.EventOrderItemCreated,
		OrderItemID:        orderItemID.String(),
		OrderID:            uuid.New().String(),
		MenuItemID:         uuid.New().String(),
		RequiresProduction: true,
	}
	eventBytes, _ := json.Marshal(evt)

	// Should not panic with nil cache
	err := s.handleEvent(context.Background(), eventBytes)
	if err != nil {
		t.Errorf("handleEvent() with nil cache should not error: %v", err)
	}
}

func TestOrderItemSubscriberCreatedPublishesEvent(t *testing.T) {
	repo := NewMockTicketRepo()
	cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	publisher := NewMockPublisher()

	s := NewOrderItemSubscriber(&MockSubscriber{}, repo, cache, publisher, aqm.NewNoopLogger())

	evt := event.OrderItemEvent{
		EventType:          event.EventOrderItemCreated,
		OrderItemID:        uuid.New().String(),
		OrderID:            uuid.New().String(),
		MenuItemID:         uuid.New().String(),
		RequiresProduction: true,
		ProductionStation:  "kitchen",
		MenuItemName:       "Pizza",
		StationName:        "Kitchen",
		TableNumber:        "T5",
		Quantity:           2,
		Notes:              "Extra cheese",
	}
	eventBytes, _ := json.Marshal(evt)

	s.handleEvent(context.Background(), eventBytes)

	if len(publisher.PublishedEvents) != 1 {
		t.Fatalf("Expected 1 published event, got %d", len(publisher.PublishedEvents))
	}

	// Verify published event content
	var publishedEvt event.KitchenTicketCreatedEvent
	json.Unmarshal(publisher.PublishedEvents[0].Data, &publishedEvt)

	if publishedEvt.EventType != event.EventKitchenTicketCreated {
		t.Errorf("published event type = %v, want %v", publishedEvt.EventType, event.EventKitchenTicketCreated)
	}
	if publishedEvt.MenuItemName != "Pizza" {
		t.Errorf("published event MenuItemName = %v, want 'Pizza'", publishedEvt.MenuItemName)
	}
	if publishedEvt.Quantity != 2 {
		t.Errorf("published event Quantity = %d, want 2", publishedEvt.Quantity)
	}
}

func TestOrderItemSubscriberOccurredAtTimestamp(t *testing.T) {
	before := time.Now()

	repo := NewMockTicketRepo()
	cache := kitchen.NewTicketStateCache(nil, nil, aqm.NewNoopLogger())
	publisher := NewMockPublisher()

	s := NewOrderItemSubscriber(&MockSubscriber{}, repo, cache, publisher, aqm.NewNoopLogger())

	evt := event.OrderItemEvent{
		EventType:          event.EventOrderItemCreated,
		OrderItemID:        uuid.New().String(),
		OrderID:            uuid.New().String(),
		MenuItemID:         uuid.New().String(),
		RequiresProduction: true,
	}
	eventBytes, _ := json.Marshal(evt)

	s.handleEvent(context.Background(), eventBytes)

	after := time.Now()

	var publishedEvt event.KitchenTicketCreatedEvent
	json.Unmarshal(publisher.PublishedEvents[0].Data, &publishedEvt)

	if publishedEvt.OccurredAt.Before(before) || publishedEvt.OccurredAt.After(after) {
		t.Error("OccurredAt timestamp is outside expected range")
	}
}
