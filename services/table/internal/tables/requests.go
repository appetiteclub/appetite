package tables

import (
	"time"

	"github.com/google/uuid"
)

type TableCreateRequest struct {
	Number     string     `json:"number"`
	GuestCount int        `json:"guest_count,omitempty"`
	AssignedTo *uuid.UUID `json:"assigned_to,omitempty"`
}

type TableUpdateRequest struct {
	Number     string     `json:"number,omitempty"`
	Status     string     `json:"status,omitempty"`
	GuestCount int        `json:"guest_count,omitempty"`
	AssignedTo *uuid.UUID `json:"assigned_to,omitempty"`
}

type TableOpenRequest struct {
	GuestCount int        `json:"guest_count"`
	AssignedTo *uuid.UUID `json:"assigned_to,omitempty"`
}

type GroupCreateRequest struct {
	TableID uuid.UUID `json:"table_id"`
	Name    string    `json:"name"`
}

type OrderCreateRequest struct {
	TableID uuid.UUID `json:"table_id"`
}

type OrderItemCreateRequest struct {
	OrderID  uuid.UUID  `json:"order_id"`
	GroupID  *uuid.UUID `json:"group_id,omitempty"`
	DishName string     `json:"dish_name"`
	Category string     `json:"category"`
	Quantity int        `json:"quantity"`
	Price    float64    `json:"price"`
	Notes    string     `json:"notes,omitempty"`
}

type OrderItemUpdateRequest struct {
	Status string `json:"status,omitempty"`
	Notes  string `json:"notes,omitempty"`
}

type ReservationCreateRequest struct {
	TableID     *uuid.UUID `json:"table_id,omitempty"`
	GuestCount  int        `json:"guest_count"`
	ReservedFor time.Time  `json:"reserved_for"`
	ContactName string     `json:"contact_name"`
	ContactInfo string     `json:"contact_info"`
	Notes       string     `json:"notes,omitempty"`
}

type ReservationUpdateRequest struct {
	TableID     *uuid.UUID `json:"table_id,omitempty"`
	GuestCount  int        `json:"guest_count,omitempty"`
	ReservedFor *time.Time `json:"reserved_for,omitempty"`
	ContactName string     `json:"contact_name,omitempty"`
	ContactInfo string     `json:"contact_info,omitempty"`
	Status      string     `json:"status,omitempty"`
	Notes       string     `json:"notes,omitempty"`
}

type BillSplitRequest struct {
	Mode string `json:"mode"` // "evenly" or "by_item"
}

type PaymentRequest struct {
	Amount float64 `json:"amount"`
	Method string  `json:"method"`
}
