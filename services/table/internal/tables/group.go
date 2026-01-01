package tables

import (
	"time"

	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
)

type Group struct {
	ID        uuid.UUID `json:"id" bson:"_id"`
	TableID   uuid.UUID `json:"table_id" bson:"table_id"`
	Name      string    `json:"name" bson:"name"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	CreatedBy string    `json:"created_by" bson:"created_by"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
	UpdatedBy string    `json:"updated_by" bson:"updated_by"`
}

func (g *Group) GetID() uuid.UUID {
	return g.ID
}

func (g *Group) ResourceType() string {
	return "group"
}

func (g *Group) SetID(id uuid.UUID) {
	g.ID = id
}

func NewGroup() *Group {
	return &Group{
		ID: apt.GenerateNewID(),
	}
}

func (g *Group) EnsureID() {
	if g.ID == uuid.Nil {
		g.ID = apt.GenerateNewID()
	}
}

func (g *Group) BeforeCreate() {
	g.EnsureID()
	g.CreatedAt = time.Now()
	g.UpdatedAt = time.Now()
}

func (g *Group) BeforeUpdate() {
	g.UpdatedAt = time.Now()
}
