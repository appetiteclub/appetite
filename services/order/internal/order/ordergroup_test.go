package order

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewOrderGroup(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440020")
	name := "Appetizers"

	group := NewOrderGroup(orderID, name)

	if group == nil {
		t.Fatal("NewOrderGroup() returned nil")
	}

	if group.ID == uuid.Nil {
		t.Error("NewOrderGroup() should generate a non-nil UUID")
	}

	if group.OrderID != orderID {
		t.Errorf("NewOrderGroup() OrderID = %v, want %v", group.OrderID, orderID)
	}

	if group.Name != name {
		t.Errorf("NewOrderGroup() Name = %q, want %q", group.Name, name)
	}

	if group.CreatedAt.IsZero() {
		t.Error("NewOrderGroup() should set CreatedAt via BeforeCreate()")
	}

	if group.UpdatedAt.IsZero() {
		t.Error("NewOrderGroup() should set UpdatedAt via BeforeCreate()")
	}
}

func TestOrderGroupGetID(t *testing.T) {
	tests := []struct {
		name  string
		group *OrderGroup
		want  uuid.UUID
	}{
		{
			name:  "returnsCorrectID",
			group: &OrderGroup{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440021")},
			want:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440021"),
		},
		{
			name:  "returnsNilUUIDWhenNotSet",
			group: &OrderGroup{},
			want:  uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.group.GetID(); got != tt.want {
				t.Errorf("OrderGroup.GetID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrderGroupResourceType(t *testing.T) {
	group := &OrderGroup{}
	got := group.ResourceType()
	want := "order-group"

	if got != want {
		t.Errorf("OrderGroup.ResourceType() = %q, want %q", got, want)
	}
}

func TestOrderGroupEnsureID(t *testing.T) {
	tests := []struct {
		name        string
		group       *OrderGroup
		expectNewID bool
	}{
		{
			name:        "generatesIDWhenNil",
			group:       &OrderGroup{ID: uuid.Nil},
			expectNewID: true,
		},
		{
			name:        "preservesExistingID",
			group:       &OrderGroup{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440022")},
			expectNewID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.group.ID
			tt.group.EnsureID()

			if tt.expectNewID {
				if tt.group.ID == uuid.Nil {
					t.Error("EnsureID() should generate non-nil UUID")
				}
			} else {
				if tt.group.ID != originalID {
					t.Errorf("EnsureID() changed existing ID from %v to %v", originalID, tt.group.ID)
				}
			}
		})
	}
}

func TestOrderGroupBeforeCreate(t *testing.T) {
	group := &OrderGroup{ID: uuid.Nil}
	beforeTime := time.Now()

	group.BeforeCreate()

	afterTime := time.Now()

	if group.ID == uuid.Nil {
		t.Error("BeforeCreate() should generate UUID")
	}

	if group.CreatedAt.IsZero() {
		t.Error("BeforeCreate() should set CreatedAt")
	}
	if group.UpdatedAt.IsZero() {
		t.Error("BeforeCreate() should set UpdatedAt")
	}

	if group.CreatedAt.Before(beforeTime) || group.CreatedAt.After(afterTime) {
		t.Error("BeforeCreate() CreatedAt timestamp is out of expected range")
	}
	if group.UpdatedAt.Before(beforeTime) || group.UpdatedAt.After(afterTime) {
		t.Error("BeforeCreate() UpdatedAt timestamp is out of expected range")
	}
}

func TestOrderGroupBeforeUpdate(t *testing.T) {
	group := &OrderGroup{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440023"),
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	originalCreatedAt := group.CreatedAt
	beforeTime := time.Now()

	group.BeforeUpdate()

	afterTime := time.Now()

	if !group.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("BeforeUpdate() changed CreatedAt from %v to %v", originalCreatedAt, group.CreatedAt)
	}

	if group.UpdatedAt.Before(beforeTime) || group.UpdatedAt.After(afterTime) {
		t.Error("BeforeUpdate() UpdatedAt timestamp is out of expected range")
	}
}

func TestOrderGroupMarkDefault(t *testing.T) {
	tests := []struct {
		name           string
		initialDefault bool
	}{
		{
			name:           "setsDefaultWhenFalse",
			initialDefault: false,
		},
		{
			name:           "remainsDefaultWhenTrue",
			initialDefault: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := &OrderGroup{
				ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440024"),
				IsDefault: tt.initialDefault,
			}

			group.MarkDefault()

			if !group.IsDefault {
				t.Error("MarkDefault() should set IsDefault to true")
			}
		})
	}
}

func TestOrderGroupAllFields(t *testing.T) {
	testTime := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)

	group := &OrderGroup{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440025"),
		OrderID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440026"),
		Name:      "Main Course",
		CreatedAt: testTime,
		CreatedBy: "user-123",
		UpdatedAt: testTime,
		UpdatedBy: "user-456",
		IsDefault: true,
	}

	if group.Name != "Main Course" {
		t.Errorf("Name = %q, want %q", group.Name, "Main Course")
	}

	if group.CreatedBy != "user-123" {
		t.Errorf("CreatedBy = %q, want %q", group.CreatedBy, "user-123")
	}

	if group.UpdatedBy != "user-456" {
		t.Errorf("UpdatedBy = %q, want %q", group.UpdatedBy, "user-456")
	}

	if !group.IsDefault {
		t.Error("IsDefault should be true")
	}
}
