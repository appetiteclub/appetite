package order

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewOrderItem(t *testing.T) {
	item := NewOrderItem()

	if item == nil {
		t.Fatal("NewOrderItem() returned nil")
	}

	if item.ID == uuid.Nil {
		t.Error("NewOrderItem() should generate a non-nil UUID")
	}

	if item.Status != "pending" {
		t.Errorf("NewOrderItem() Status = %q, want %q", item.Status, "pending")
	}
}

func TestOrderItemGetID(t *testing.T) {
	tests := []struct {
		name string
		item *OrderItem
		want uuid.UUID
	}{
		{
			name: "returnsCorrectID",
			item: &OrderItem{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440010")},
			want: uuid.MustParse("550e8400-e29b-41d4-a716-446655440010"),
		},
		{
			name: "returnsNilUUIDWhenNotSet",
			item: &OrderItem{},
			want: uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.GetID(); got != tt.want {
				t.Errorf("OrderItem.GetID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrderItemSetID(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID
	}{
		{
			name: "setsValidUUID",
			id:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440011"),
		},
		{
			name: "setsNilUUID",
			id:   uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &OrderItem{}
			item.SetID(tt.id)

			if item.ID != tt.id {
				t.Errorf("OrderItem.SetID() ID = %v, want %v", item.ID, tt.id)
			}
		})
	}
}

func TestOrderItemResourceType(t *testing.T) {
	item := &OrderItem{}
	got := item.ResourceType()
	want := "order-item"

	if got != want {
		t.Errorf("OrderItem.ResourceType() = %q, want %q", got, want)
	}
}

func TestOrderItemEnsureID(t *testing.T) {
	tests := []struct {
		name        string
		item        *OrderItem
		expectNewID bool
	}{
		{
			name:        "generatesIDWhenNil",
			item:        &OrderItem{ID: uuid.Nil},
			expectNewID: true,
		},
		{
			name:        "preservesExistingID",
			item:        &OrderItem{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440012")},
			expectNewID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.item.ID
			tt.item.EnsureID()

			if tt.expectNewID {
				if tt.item.ID == uuid.Nil {
					t.Error("EnsureID() should generate non-nil UUID")
				}
			} else {
				if tt.item.ID != originalID {
					t.Errorf("EnsureID() changed existing ID from %v to %v", originalID, tt.item.ID)
				}
			}
		})
	}
}

func TestOrderItemBeforeCreate(t *testing.T) {
	item := &OrderItem{ID: uuid.Nil}
	beforeTime := time.Now()

	item.BeforeCreate()

	afterTime := time.Now()

	if item.ID == uuid.Nil {
		t.Error("BeforeCreate() should generate UUID")
	}

	if item.CreatedAt.IsZero() {
		t.Error("BeforeCreate() should set CreatedAt")
	}
	if item.UpdatedAt.IsZero() {
		t.Error("BeforeCreate() should set UpdatedAt")
	}

	if item.CreatedAt.Before(beforeTime) || item.CreatedAt.After(afterTime) {
		t.Error("BeforeCreate() CreatedAt timestamp is out of expected range")
	}
	if item.UpdatedAt.Before(beforeTime) || item.UpdatedAt.After(afterTime) {
		t.Error("BeforeCreate() UpdatedAt timestamp is out of expected range")
	}
}

func TestOrderItemBeforeUpdate(t *testing.T) {
	item := &OrderItem{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440013"),
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	originalCreatedAt := item.CreatedAt
	beforeTime := time.Now()

	item.BeforeUpdate()

	afterTime := time.Now()

	if !item.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("BeforeUpdate() changed CreatedAt from %v to %v", originalCreatedAt, item.CreatedAt)
	}

	if item.UpdatedAt.Before(beforeTime) || item.UpdatedAt.After(afterTime) {
		t.Error("BeforeUpdate() UpdatedAt timestamp is out of expected range")
	}
}

func TestOrderItemStatusTransitions(t *testing.T) {
	tests := []struct {
		name           string
		action         func(*OrderItem)
		expectedStatus string
		checkDelivered bool
	}{
		{
			name:           "markAsPreparing",
			action:         func(oi *OrderItem) { oi.MarkAsPreparing() },
			expectedStatus: "preparing",
			checkDelivered: false,
		},
		{
			name:           "markAsReady",
			action:         func(oi *OrderItem) { oi.MarkAsReady() },
			expectedStatus: "ready",
			checkDelivered: false,
		},
		{
			name:           "markAsDelivered",
			action:         func(oi *OrderItem) { oi.MarkAsDelivered() },
			expectedStatus: "delivered",
			checkDelivered: true,
		},
		{
			name:           "cancel",
			action:         func(oi *OrderItem) { oi.Cancel() },
			expectedStatus: "cancelled",
			checkDelivered: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &OrderItem{
				ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440014"),
				Status:    "pending",
				UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			}

			beforeTime := time.Now()
			tt.action(item)
			afterTime := time.Now()

			if item.Status != tt.expectedStatus {
				t.Errorf("Status = %q, want %q", item.Status, tt.expectedStatus)
			}

			if item.UpdatedAt.Before(beforeTime) || item.UpdatedAt.After(afterTime) {
				t.Error("UpdatedAt timestamp should be updated")
			}

			if tt.checkDelivered {
				if item.DeliveredAt == nil {
					t.Error("DeliveredAt should be set when marking as delivered")
				} else if item.DeliveredAt.Before(beforeTime) || item.DeliveredAt.After(afterTime) {
					t.Error("DeliveredAt timestamp is out of expected range")
				}
			}
		})
	}
}

func TestOrderItemWithOptionalFields(t *testing.T) {
	groupID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440015")
	menuItemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440016")
	station := "grill"

	item := &OrderItem{
		ID:                 uuid.MustParse("550e8400-e29b-41d4-a716-446655440017"),
		OrderID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440018"),
		GroupID:            &groupID,
		DishName:           "Grilled Steak",
		Category:           "Main",
		Quantity:           2,
		Price:              29.99,
		Status:             "pending",
		Notes:              "Medium rare",
		MenuItemID:         &menuItemID,
		ProductionStation:  &station,
		RequiresProduction: true,
	}

	if item.GroupID == nil || *item.GroupID != groupID {
		t.Error("GroupID should be set correctly")
	}

	if item.MenuItemID == nil || *item.MenuItemID != menuItemID {
		t.Error("MenuItemID should be set correctly")
	}

	if item.ProductionStation == nil || *item.ProductionStation != station {
		t.Error("ProductionStation should be set correctly")
	}

	if !item.RequiresProduction {
		t.Error("RequiresProduction should be true")
	}

	if item.DishName != "Grilled Steak" {
		t.Errorf("DishName = %q, want %q", item.DishName, "Grilled Steak")
	}

	if item.Quantity != 2 {
		t.Errorf("Quantity = %d, want %d", item.Quantity, 2)
	}

	if item.Price != 29.99 {
		t.Errorf("Price = %f, want %f", item.Price, 29.99)
	}
}
