package tables

import (
	"time"

	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
)

type Order struct {
	ID        uuid.UUID `json:"id" bson:"_id"`
	TableID   uuid.UUID `json:"table_id" bson:"table_id"`
	Status    string    `json:"status" bson:"status"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	CreatedBy string    `json:"created_by" bson:"created_by"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
	UpdatedBy string    `json:"updated_by" bson:"updated_by"`
}

func (o *Order) GetID() uuid.UUID {
	return o.ID
}

func (o *Order) ResourceType() string {
	return "order"
}

func (o *Order) SetID(id uuid.UUID) {
	o.ID = id
}

func NewOrder() *Order {
	return &Order{
		ID:     apt.GenerateNewID(),
		Status: "pending",
	}
}

func (o *Order) EnsureID() {
	if o.ID == uuid.Nil {
		o.ID = apt.GenerateNewID()
	}
}

func (o *Order) BeforeCreate() {
	o.EnsureID()
	o.CreatedAt = time.Now()
	o.UpdatedAt = time.Now()
}

func (o *Order) BeforeUpdate() {
	o.UpdatedAt = time.Now()
}

func (o *Order) MarkAsPreparing() {
	o.Status = "preparing"
	o.UpdatedAt = time.Now()
}

func (o *Order) MarkAsReady() {
	o.Status = "ready"
	o.UpdatedAt = time.Now()
}

func (o *Order) MarkAsDelivered() {
	o.Status = "delivered"
	o.UpdatedAt = time.Now()
}

func (o *Order) Cancel() {
	o.Status = "cancelled"
	o.UpdatedAt = time.Now()
}
