package order

import (
	"time"

	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
)

type OrderGroup struct {
	ID        uuid.UUID `json:"id" bson:"_id"`
	OrderID   uuid.UUID `json:"order_id" bson:"order_id"`
	Name      string    `json:"name" bson:"name"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	CreatedBy string    `json:"created_by" bson:"created_by"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
	UpdatedBy string    `json:"updated_by" bson:"updated_by"`
	IsDefault bool      `json:"is_default" bson:"is_default"`
}

func (g *OrderGroup) GetID() uuid.UUID {
	return g.ID
}

func (g *OrderGroup) ResourceType() string {
	return "order-group"
}

func NewOrderGroup(orderID uuid.UUID, name string) *OrderGroup {
	group := &OrderGroup{
		ID:      apt.GenerateNewID(),
		OrderID: orderID,
		Name:    name,
	}
	group.BeforeCreate()
	return group
}

func (g *OrderGroup) EnsureID() {
	if g.ID == uuid.Nil {
		g.ID = apt.GenerateNewID()
	}
}

func (g *OrderGroup) BeforeCreate() {
	g.EnsureID()
	g.CreatedAt = time.Now()
	g.UpdatedAt = time.Now()
}

func (g *OrderGroup) BeforeUpdate() {
	g.UpdatedAt = time.Now()
}

func (g *OrderGroup) MarkDefault() {
	g.IsDefault = true
}
