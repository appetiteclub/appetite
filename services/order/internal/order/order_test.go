package order

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewOrder(t *testing.T) {
	order := NewOrder()

	if order == nil {
		t.Fatal("NewOrder() returned nil")
	}

	if order.ID == uuid.Nil {
		t.Error("NewOrder() should generate a non-nil UUID")
	}

	if order.Status != "pending" {
		t.Errorf("NewOrder() Status = %q, want %q", order.Status, "pending")
	}
}

func TestOrderGetID(t *testing.T) {
	tests := []struct {
		name  string
		order *Order
		want  uuid.UUID
	}{
		{
			name:  "returnsCorrectID",
			order: &Order{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")},
			want:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
		{
			name:  "returnsNilUUIDWhenNotSet",
			order: &Order{},
			want:  uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.order.GetID(); got != tt.want {
				t.Errorf("Order.GetID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrderSetID(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID
	}{
		{
			name: "setsValidUUID",
			id:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		},
		{
			name: "setsNilUUID",
			id:   uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{}
			order.SetID(tt.id)

			if order.ID != tt.id {
				t.Errorf("Order.SetID() ID = %v, want %v", order.ID, tt.id)
			}
		})
	}
}

func TestOrderResourceType(t *testing.T) {
	order := &Order{}
	got := order.ResourceType()
	want := "order"

	if got != want {
		t.Errorf("Order.ResourceType() = %q, want %q", got, want)
	}
}

func TestOrderEnsureID(t *testing.T) {
	tests := []struct {
		name        string
		order       *Order
		expectNewID bool
	}{
		{
			name:        "generatesIDWhenNil",
			order:       &Order{ID: uuid.Nil},
			expectNewID: true,
		},
		{
			name:        "preservesExistingID",
			order:       &Order{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002")},
			expectNewID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.order.ID
			tt.order.EnsureID()

			if tt.expectNewID {
				if tt.order.ID == uuid.Nil {
					t.Error("EnsureID() should generate non-nil UUID")
				}
			} else {
				if tt.order.ID != originalID {
					t.Errorf("EnsureID() changed existing ID from %v to %v", originalID, tt.order.ID)
				}
			}
		})
	}
}

func TestOrderBeforeCreate(t *testing.T) {
	order := &Order{ID: uuid.Nil}
	beforeTime := time.Now()

	order.BeforeCreate()

	afterTime := time.Now()

	if order.ID == uuid.Nil {
		t.Error("BeforeCreate() should generate UUID")
	}

	if order.CreatedAt.IsZero() {
		t.Error("BeforeCreate() should set CreatedAt")
	}
	if order.UpdatedAt.IsZero() {
		t.Error("BeforeCreate() should set UpdatedAt")
	}

	if order.CreatedAt.Before(beforeTime) || order.CreatedAt.After(afterTime) {
		t.Error("BeforeCreate() CreatedAt timestamp is out of expected range")
	}
	if order.UpdatedAt.Before(beforeTime) || order.UpdatedAt.After(afterTime) {
		t.Error("BeforeCreate() UpdatedAt timestamp is out of expected range")
	}
}

func TestOrderBeforeUpdate(t *testing.T) {
	order := &Order{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"),
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	originalCreatedAt := order.CreatedAt
	beforeTime := time.Now()

	order.BeforeUpdate()

	afterTime := time.Now()

	if !order.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("BeforeUpdate() changed CreatedAt from %v to %v", originalCreatedAt, order.CreatedAt)
	}

	if order.UpdatedAt.Before(beforeTime) || order.UpdatedAt.After(afterTime) {
		t.Error("BeforeUpdate() UpdatedAt timestamp is out of expected range")
	}
}

func TestOrderStatusTransitions(t *testing.T) {
	tests := []struct {
		name           string
		action         func(*Order)
		expectedStatus string
	}{
		{
			name:           "markAsPreparing",
			action:         func(o *Order) { o.MarkAsPreparing() },
			expectedStatus: "preparing",
		},
		{
			name:           "markAsReady",
			action:         func(o *Order) { o.MarkAsReady() },
			expectedStatus: "ready",
		},
		{
			name:           "markAsDelivered",
			action:         func(o *Order) { o.MarkAsDelivered() },
			expectedStatus: "delivered",
		},
		{
			name:           "cancel",
			action:         func(o *Order) { o.Cancel() },
			expectedStatus: "cancelled",
		},
		{
			name:           "close",
			action:         func(o *Order) { o.Close() },
			expectedStatus: "closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{
				ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440004"),
				Status:    "pending",
				UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			}

			beforeTime := time.Now()
			tt.action(order)
			afterTime := time.Now()

			if order.Status != tt.expectedStatus {
				t.Errorf("Status = %q, want %q", order.Status, tt.expectedStatus)
			}

			if order.UpdatedAt.Before(beforeTime) || order.UpdatedAt.After(afterTime) {
				t.Error("UpdatedAt timestamp should be updated")
			}
		})
	}
}
