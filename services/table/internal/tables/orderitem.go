package tables

import (
	"time"

	"github.com/aquamarinepk/aqm"
	"github.com/google/uuid"
)

type OrderItem struct {
	ID          uuid.UUID  `json:"id" bson:"_id"`
	OrderID     uuid.UUID  `json:"order_id" bson:"order_id"`
	GroupID     *uuid.UUID `json:"group_id,omitempty" bson:"group_id,omitempty"`
	DishName    string     `json:"dish_name" bson:"dish_name"`
	Category    string     `json:"category" bson:"category"`
	Quantity    int        `json:"quantity" bson:"quantity"`
	Price       float64    `json:"price" bson:"price"`
	Status      string     `json:"status" bson:"status"`
	Notes       string     `json:"notes,omitempty" bson:"notes,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty" bson:"delivered_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at" bson:"created_at"`
	CreatedBy   string     `json:"created_by" bson:"created_by"`
	UpdatedAt   time.Time  `json:"updated_at" bson:"updated_at"`
	UpdatedBy   string     `json:"updated_by" bson:"updated_by"`
}

func (oi *OrderItem) GetID() uuid.UUID {
	return oi.ID
}

func (oi *OrderItem) ResourceType() string {
	return "order-item"
}

func (oi *OrderItem) SetID(id uuid.UUID) {
	oi.ID = id
}

func NewOrderItem() *OrderItem {
	return &OrderItem{
		ID:     aqm.GenerateNewID(),
		Status: "pending",
	}
}

func (oi *OrderItem) EnsureID() {
	if oi.ID == uuid.Nil {
		oi.ID = aqm.GenerateNewID()
	}
}

func (oi *OrderItem) BeforeCreate() {
	oi.EnsureID()
	oi.CreatedAt = time.Now()
	oi.UpdatedAt = time.Now()
}

func (oi *OrderItem) BeforeUpdate() {
	oi.UpdatedAt = time.Now()
}

func (oi *OrderItem) MarkAsPreparing() {
	oi.Status = "preparing"
	oi.UpdatedAt = time.Now()
}

func (oi *OrderItem) MarkAsReady() {
	oi.Status = "ready"
	oi.UpdatedAt = time.Now()
}

func (oi *OrderItem) MarkAsDelivered() {
	now := time.Now()
	oi.Status = "delivered"
	oi.DeliveredAt = &now
	oi.UpdatedAt = now
}

func (oi *OrderItem) Cancel() {
	oi.Status = "cancelled"
	oi.UpdatedAt = time.Now()
}
