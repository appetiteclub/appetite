package order

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/appetiteclub/appetite/pkg"
	"github.com/appetiteclub/appetite/pkg/event"
	proto "github.com/appetiteclub/appetite/services/order/internal/order/proto"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
	"github.com/google/uuid"
)

func TestNewTableStatusSubscriber(t *testing.T) {
	cache := NewTableStateCache(nil, nil)

	tests := []struct {
		name  string
		cache *TableStateCache
	}{
		{
			name:  "withCache",
			cache: cache,
		},
		{
			name:  "withNilCache",
			cache: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := NewTableStatusSubscriber(nil, tt.cache, nil)

			if sub == nil {
				t.Fatal("NewTableStatusSubscriber() returned nil")
			}

			if sub.logger == nil {
				t.Error("NewTableStatusSubscriber() should set noop logger when nil")
			}

			if tt.cache != nil && sub.cache != tt.cache {
				t.Error("NewTableStatusSubscriber() should set cache")
			}
		})
	}
}

func TestTableStatusSubscriberStartNilSubscriber(t *testing.T) {
	sub := NewTableStatusSubscriber(nil, nil, nil)

	err := sub.Start(context.Background())
	if err == nil {
		t.Error("Start() with nil subscriber should return error")
	}

	expectedMsg := "table status subscriber not configured"
	if err.Error() != expectedMsg {
		t.Errorf("Start() error = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestNewKitchenTicketSubscriber(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "withNilDependencies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := NewKitchenTicketSubscriber(nil, nil, nil)

			if sub == nil {
				t.Fatal("NewKitchenTicketSubscriber() returned nil")
			}

			if sub.logger == nil {
				t.Error("NewKitchenTicketSubscriber() should set noop logger when nil")
			}
		})
	}
}

func TestKitchenTicketSubscriberSetStreamServer(t *testing.T) {
	sub := NewKitchenTicketSubscriber(nil, nil, nil)

	if sub.streamServer != nil {
		t.Error("streamServer should be nil initially")
	}

	// Create a mock stream server (just for testing the setter)
	streamServer := &OrderEventStreamServer{}
	sub.SetStreamServer(streamServer)

	if sub.streamServer != streamServer {
		t.Error("SetStreamServer() should set the stream server")
	}
}

func TestKitchenTicketSubscriberStartNilSubscriber(t *testing.T) {
	sub := NewKitchenTicketSubscriber(nil, nil, nil)

	err := sub.Start(context.Background())
	if err == nil {
		t.Error("Start() with nil subscriber should return error")
	}

	expectedMsg := "kitchen ticket subscriber not configured"
	if err.Error() != expectedMsg {
		t.Errorf("Start() error = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestMapKitchenStatusToOrderStatus(t *testing.T) {
	sub := NewKitchenTicketSubscriber(nil, nil, nil)

	tests := []struct {
		name          string
		kitchenStatus string
		expectedOrder string
	}{
		{
			name:          "created",
			kitchenStatus: "created",
			expectedOrder: "pending",
		},
		{
			name:          "started",
			kitchenStatus: "started",
			expectedOrder: "preparing",
		},
		{
			name:          "ready",
			kitchenStatus: "ready",
			expectedOrder: "ready",
		},
		{
			name:          "delivered",
			kitchenStatus: "delivered",
			expectedOrder: "delivered",
		},
		{
			name:          "cancelled",
			kitchenStatus: "cancelled",
			expectedOrder: "cancelled",
		},
		{
			name:          "unknownStatus",
			kitchenStatus: "unknown",
			expectedOrder: "",
		},
		{
			name:          "emptyStatus",
			kitchenStatus: "",
			expectedOrder: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sub.mapKitchenStatusToOrderStatus(tt.kitchenStatus)
			if result != tt.expectedOrder {
				t.Errorf("mapKitchenStatusToOrderStatus(%q) = %q, want %q",
					tt.kitchenStatus, result, tt.expectedOrder)
			}
		})
	}
}

func TestTableStatusSubscriberHandleEvent(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440090")

	tests := []struct {
		name           string
		event          interface{}
		expectedStatus string
		expectCached   bool
	}{
		{
			name: "validEvent",
			event: pkg.TableStatusEvent{
				TableID:    tableID.String(),
				Status:     "occupied",
				OccurredAt: time.Now(),
			},
			expectedStatus: "occupied",
			expectCached:   true,
		},
		{
			name: "invalidTableID",
			event: pkg.TableStatusEvent{
				TableID:    "not-a-uuid",
				Status:     "available",
				OccurredAt: time.Now(),
			},
			expectedStatus: "",
			expectCached:   false,
		},
		{
			name:           "invalidJSON",
			event:          "not json",
			expectedStatus: "",
			expectCached:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewTableStateCache(nil, nil)
			sub := NewTableStatusSubscriber(nil, cache, nil)

			var msg []byte
			if s, ok := tt.event.(string); ok {
				msg = []byte(s)
			} else {
				msg, _ = json.Marshal(tt.event)
			}

			// Call handleEvent directly
			err := sub.handleEvent(context.Background(), msg)
			if err != nil {
				t.Errorf("handleEvent() unexpected error: %v", err)
			}

			if tt.expectCached {
				status, ok := cache.Get(tableID)
				if !ok {
					t.Error("handleEvent() table status not cached")
				}
				if status != tt.expectedStatus {
					t.Errorf("handleEvent() cached status = %q, want %q", status, tt.expectedStatus)
				}
			}
		})
	}
}

func TestKitchenTicketSubscriberHandleEvent(t *testing.T) {
	orderItemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440091")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440092")

	tests := []struct {
		name           string
		event          interface{}
		setupRepo      func(*MockOrderItemRepo)
		expectedStatus string
		expectUpdate   bool
	}{
		{
			name: "statusChangeToReady",
			event: event.KitchenTicketStatusChangedEvent{
				KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
					EventType:   event.EventKitchenTicketStatusChange,
					TicketID:    uuid.New().String(),
					OrderItemID: orderItemID.String(),
				},
				NewStatus:      "ready",
				PreviousStatus: "started",
			},
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[orderItemID] = &OrderItem{
					ID:      orderItemID,
					OrderID: orderID,
					Status:  "preparing",
				}
			},
			expectedStatus: "ready",
			expectUpdate:   true,
		},
		{
			name: "statusChangeToPreparing",
			event: event.KitchenTicketStatusChangedEvent{
				KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
					EventType:   event.EventKitchenTicketStatusChange,
					TicketID:    uuid.New().String(),
					OrderItemID: orderItemID.String(),
				},
				NewStatus:      "started",
				PreviousStatus: "created",
			},
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[orderItemID] = &OrderItem{
					ID:      orderItemID,
					OrderID: orderID,
					Status:  "pending",
				}
			},
			expectedStatus: "preparing",
			expectUpdate:   true,
		},
		{
			name: "statusChangeToDelivered",
			event: event.KitchenTicketStatusChangedEvent{
				KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
					EventType:   event.EventKitchenTicketStatusChange,
					TicketID:    uuid.New().String(),
					OrderItemID: orderItemID.String(),
				},
				NewStatus:      "delivered",
				PreviousStatus: "ready",
			},
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[orderItemID] = &OrderItem{
					ID:      orderItemID,
					OrderID: orderID,
					Status:  "ready",
				}
			},
			expectedStatus: "delivered",
			expectUpdate:   true,
		},
		{
			name: "statusChangeToCancelled",
			event: event.KitchenTicketStatusChangedEvent{
				KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
					EventType:   event.EventKitchenTicketStatusChange,
					TicketID:    uuid.New().String(),
					OrderItemID: orderItemID.String(),
				},
				NewStatus:      "cancelled",
				PreviousStatus: "pending",
			},
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[orderItemID] = &OrderItem{
					ID:      orderItemID,
					OrderID: orderID,
					Status:  "pending",
				}
			},
			expectedStatus: "cancelled",
			expectUpdate:   true,
		},
		{
			name: "ticketCreatedEventIgnored",
			event: event.KitchenTicketEventMetadata{
				EventType: event.EventKitchenTicketCreated,
			},
			setupRepo:    func(repo *MockOrderItemRepo) {},
			expectUpdate: false,
		},
		{
			name: "unknownEventType",
			event: event.KitchenTicketEventMetadata{
				EventType: "unknown.event",
			},
			setupRepo:    func(repo *MockOrderItemRepo) {},
			expectUpdate: false,
		},
		{
			name:         "invalidJSON",
			event:        "not json",
			setupRepo:    func(repo *MockOrderItemRepo) {},
			expectUpdate: false,
		},
		{
			name: "missingOrderItemID",
			event: event.KitchenTicketStatusChangedEvent{
				KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
					EventType:   event.EventKitchenTicketStatusChange,
					TicketID:    uuid.New().String(),
					OrderItemID: "",
				},
				NewStatus: "ready",
			},
			setupRepo:    func(repo *MockOrderItemRepo) {},
			expectUpdate: false,
		},
		{
			name: "invalidOrderItemID",
			event: event.KitchenTicketStatusChangedEvent{
				KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
					EventType:   event.EventKitchenTicketStatusChange,
					TicketID:    uuid.New().String(),
					OrderItemID: "not-a-uuid",
				},
				NewStatus: "ready",
			},
			setupRepo:    func(repo *MockOrderItemRepo) {},
			expectUpdate: false,
		},
		{
			name: "orderItemNotFound",
			event: event.KitchenTicketStatusChangedEvent{
				KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
					EventType:   event.EventKitchenTicketStatusChange,
					TicketID:    uuid.New().String(),
					OrderItemID: uuid.New().String(),
				},
				NewStatus: "ready",
			},
			setupRepo:    func(repo *MockOrderItemRepo) {},
			expectUpdate: false,
		},
		{
			name: "unknownKitchenStatus",
			event: event.KitchenTicketStatusChangedEvent{
				KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
					EventType:   event.EventKitchenTicketStatusChange,
					TicketID:    uuid.New().String(),
					OrderItemID: orderItemID.String(),
				},
				NewStatus: "unknown_status",
			},
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[orderItemID] = &OrderItem{
					ID:      orderItemID,
					OrderID: orderID,
					Status:  "pending",
				}
			},
			expectUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderItemRepo()
			tt.setupRepo(repo)

			sub := NewKitchenTicketSubscriber(nil, repo, nil)

			var msg []byte
			if s, ok := tt.event.(string); ok {
				msg = []byte(s)
			} else {
				msg, _ = json.Marshal(tt.event)
			}

			// Call handleEvent directly
			err := sub.handleEvent(context.Background(), msg)
			if err != nil {
				t.Errorf("handleEvent() unexpected error: %v", err)
			}

			if tt.expectUpdate {
				item, getErr := repo.Get(context.Background(), orderItemID)
				if getErr != nil {
					t.Errorf("failed to get order item: %v", getErr)
					return
				}
				if item.Status != tt.expectedStatus {
					t.Errorf("handleEvent() item status = %q, want %q", item.Status, tt.expectedStatus)
				}
			}
		})
	}
}

func TestNewOrderEventStreamServer(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "withNilDependencies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewOrderEventStreamServer(nil, nil)

			if server == nil {
				t.Fatal("NewOrderEventStreamServer() returned nil")
			}

			if server.subscribers == nil {
				t.Error("NewOrderEventStreamServer() should initialize subscribers map")
			}
		})
	}
}

func TestOrderEventStreamServerBroadcastOrderItemEvent(t *testing.T) {
	orderItemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440093")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440094")
	menuItemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440095")

	tests := []struct {
		name           string
		item           *OrderItem
		eventType      string
		previousStatus string
	}{
		{
			name: "broadcastStatusChange",
			item: &OrderItem{
				ID:       orderItemID,
				OrderID:  orderID,
				DishName: "Pizza",
				Category: "main",
				Status:   "ready",
				Quantity: 2,
				Price:    15.99,
			},
			eventType:      "order.item.status_changed",
			previousStatus: "preparing",
		},
		{
			name: "broadcastWithMenuItemID",
			item: &OrderItem{
				ID:         orderItemID,
				OrderID:    orderID,
				MenuItemID: &menuItemID,
				DishName:   "Burger",
				Status:     "delivered",
			},
			eventType:      "order.item.status_changed",
			previousStatus: "ready",
		},
		{
			name: "broadcastWithDeliveredAt",
			item: func() *OrderItem {
				item := &OrderItem{
					ID:       orderItemID,
					OrderID:  orderID,
					DishName: "Salad",
					Status:   "delivered",
				}
				item.MarkAsDelivered()
				return item
			}(),
			eventType:      "order.item.status_changed",
			previousStatus: "ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderItemRepo()
			logger := aqm.NewNoopLogger()
			server := NewOrderEventStreamServer(repo, logger)

			// BroadcastOrderItemEvent should not panic even with no subscribers
			server.BroadcastOrderItemEvent(tt.item, tt.eventType, tt.previousStatus)
		})
	}
}

func TestGenerateSubscriberID(t *testing.T) {
	id1 := generateSubscriberID()
	if id1 == "" {
		t.Error("generateSubscriberID() returned empty string")
	}

	// Wait a tiny bit to ensure different timestamps
	time.Sleep(time.Microsecond)

	id2 := generateSubscriberID()
	if id1 == id2 {
		t.Error("generateSubscriberID() should generate unique IDs")
	}
}

func TestTableStatusSubscriberStartWithSubscriber(t *testing.T) {
	cache := NewTableStateCache(nil, nil)
	mockSub := NewMockSubscriber()
	mockSub.SubscribeFunc = func(ctx context.Context, topic string, handler events.HandlerFunc) error {
		// Simulate successful subscription
		return nil
	}

	sub := NewTableStatusSubscriber(mockSub, cache, nil)

	err := sub.Start(context.Background())
	if err != nil {
		t.Errorf("Start() with mock subscriber should not return error, got: %v", err)
	}
}

func TestKitchenTicketSubscriberStartWithSubscriber(t *testing.T) {
	mockSub := NewMockSubscriber()
	mockSub.SubscribeFunc = func(ctx context.Context, topic string, handler events.HandlerFunc) error {
		return nil
	}

	itemRepo := NewMockOrderItemRepo()
	sub := NewKitchenTicketSubscriber(mockSub, itemRepo, nil)

	err := sub.Start(context.Background())
	if err != nil {
		t.Errorf("Start() with mock subscriber should not return error, got: %v", err)
	}
}

func TestTableStatusSubscriberStartSubscribeError(t *testing.T) {
	cache := NewTableStateCache(nil, nil)
	mockSub := NewMockSubscriber()
	mockSub.SubscribeFunc = func(ctx context.Context, topic string, handler events.HandlerFunc) error {
		return fmt.Errorf("subscription error")
	}

	sub := NewTableStatusSubscriber(mockSub, cache, nil)

	err := sub.Start(context.Background())
	if err == nil {
		t.Error("Start() with subscribe error should return error")
	}
}

func TestKitchenTicketSubscriberStartSubscribeError(t *testing.T) {
	mockSub := NewMockSubscriber()
	mockSub.SubscribeFunc = func(ctx context.Context, topic string, handler events.HandlerFunc) error {
		return fmt.Errorf("subscription error")
	}

	itemRepo := NewMockOrderItemRepo()
	sub := NewKitchenTicketSubscriber(mockSub, itemRepo, nil)

	err := sub.Start(context.Background())
	if err == nil {
		t.Error("Start() with subscribe error should return error")
	}
}

func TestKitchenTicketSubscriberHandleStatusChangeInvalidEvent(t *testing.T) {
	mockSub := NewMockSubscriber()
	itemRepo := NewMockOrderItemRepo()
	sub := NewKitchenTicketSubscriber(mockSub, itemRepo, nil)

	// Test with invalid JSON
	err := sub.handleStatusChange(context.Background(), []byte("invalid json"))
	if err != nil {
		t.Errorf("handleStatusChange() with invalid JSON should return nil, got: %v", err)
	}
}

func TestKitchenTicketSubscriberHandleStatusChangeRepoSaveError(t *testing.T) {
	orderItemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440200")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440201")

	mockSub := NewMockSubscriber()
	itemRepo := NewMockOrderItemRepo()
	itemRepo.items[orderItemID] = &OrderItem{
		ID:      orderItemID,
		OrderID: orderID,
		Status:  "pending",
	}
	itemRepo.SaveFunc = func(ctx context.Context, item *OrderItem) error {
		return fmt.Errorf("db error")
	}

	sub := NewKitchenTicketSubscriber(mockSub, itemRepo, nil)

	evt := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:   event.EventKitchenTicketStatusChange,
			TicketID:    uuid.New().String(),
			OrderItemID: orderItemID.String(),
		},
		NewStatus:      "ready",
		PreviousStatus: "started",
	}
	msg, _ := json.Marshal(evt)

	err := sub.handleStatusChange(context.Background(), msg)
	if err == nil {
		t.Error("handleStatusChange() with save error should return error")
	}
}

func TestOrderEventStreamServerBroadcastToSubscribers(t *testing.T) {
	orderItemRepo := NewMockOrderItemRepo()
	server := NewOrderEventStreamServer(orderItemRepo, aqm.NewNoopLogger())

	// Add a subscriber with a buffer
	testChan := make(chan *proto.OrderItemEvent, 10)
	server.mu.Lock()
	server.subscribers["test-sub"] = testChan
	server.mu.Unlock()

	item := &OrderItem{
		ID:                 uuid.MustParse("550e8400-e29b-41d4-a716-446655440300"),
		OrderID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440301"),
		DishName:           "Test Dish",
		Category:           "Main",
		Status:             "ready",
		Quantity:           2,
		Price:              15.99,
		RequiresProduction: true,
		Notes:              "Extra sauce",
	}

	server.BroadcastOrderItemEvent(item, "order.item.created", "pending")

	// Check if event was received
	select {
	case evt := <-testChan:
		if evt.OrderItemId != item.ID.String() {
			t.Errorf("BroadcastOrderItemEvent() OrderItemId = %q, want %q", evt.OrderItemId, item.ID.String())
		}
		if evt.DishName != "Test Dish" {
			t.Errorf("BroadcastOrderItemEvent() DishName = %q, want %q", evt.DishName, "Test Dish")
		}
	default:
		t.Error("BroadcastOrderItemEvent() should send event to subscriber")
	}
}

func TestOrderEventStreamServerBroadcastChannelFull(t *testing.T) {
	orderItemRepo := NewMockOrderItemRepo()
	server := NewOrderEventStreamServer(orderItemRepo, aqm.NewNoopLogger())

	// Add a subscriber with a buffer of 1, already full
	fullChan := make(chan *proto.OrderItemEvent, 1)
	fullChan <- &proto.OrderItemEvent{} // Fill the buffer

	server.mu.Lock()
	server.subscribers["full-sub"] = fullChan
	server.mu.Unlock()

	item := &OrderItem{
		ID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440310"),
		OrderID:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440311"),
		DishName: "Test Dish",
		Status:   "ready",
		Quantity: 1,
	}

	// Should not panic when channel is full
	server.BroadcastOrderItemEvent(item, "order.item.created", "pending")

	// The event should be dropped (channel full)
	// No assertion needed - just verify it doesn't panic
}

func TestTableStatusSubscriberHandleEventCacheUpdate(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440202")

	cache := NewTableStateCache(nil, nil)
	sub := NewTableStatusSubscriber(nil, cache, nil)

	// Test status change that updates cache
	evt := pkg.TableStatusEvent{
		TableID:    tableID.String(),
		Status:     "reserved",
		OccurredAt: time.Now(),
	}
	msg, _ := json.Marshal(evt)

	err := sub.handleEvent(context.Background(), msg)
	if err != nil {
		t.Errorf("handleEvent() unexpected error: %v", err)
	}

	// Verify cache was updated
	status, ok := cache.Get(tableID)
	if !ok {
		t.Error("handleEvent() should cache the table status")
	}
	if status != "reserved" {
		t.Errorf("handleEvent() cached status = %q, want %q", status, "reserved")
	}
}
