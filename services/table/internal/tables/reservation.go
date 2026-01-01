package tables

import (
	"time"

	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
)

type Reservation struct {
	ID          uuid.UUID  `json:"id" bson:"_id"`
	TableID     *uuid.UUID `json:"table_id,omitempty" bson:"table_id,omitempty"`
	GuestCount  int        `json:"guest_count" bson:"guest_count"`
	ReservedFor time.Time  `json:"reserved_for" bson:"reserved_for"`
	ContactName string     `json:"contact_name" bson:"contact_name"`
	ContactInfo string     `json:"contact_info" bson:"contact_info"`
	Status      string     `json:"status" bson:"status"`
	Notes       string     `json:"notes,omitempty" bson:"notes,omitempty"`
	CreatedAt   time.Time  `json:"created_at" bson:"created_at"`
	CreatedBy   string     `json:"created_by" bson:"created_by"`
	UpdatedAt   time.Time  `json:"updated_at" bson:"updated_at"`
	UpdatedBy   string     `json:"updated_by" bson:"updated_by"`
}

func (r *Reservation) GetID() uuid.UUID {
	return r.ID
}

func (r *Reservation) ResourceType() string {
	return "reservation"
}

func (r *Reservation) SetID(id uuid.UUID) {
	r.ID = id
}

func NewReservation() *Reservation {
	return &Reservation{
		ID:     apt.GenerateNewID(),
		Status: "confirmed",
	}
}

func (r *Reservation) EnsureID() {
	if r.ID == uuid.Nil {
		r.ID = apt.GenerateNewID()
	}
}

func (r *Reservation) BeforeCreate() {
	r.EnsureID()
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
}

func (r *Reservation) BeforeUpdate() {
	r.UpdatedAt = time.Now()
}

func (r *Reservation) MarkAsSeated() {
	r.Status = "seated"
	r.UpdatedAt = time.Now()
}

func (r *Reservation) Cancel() {
	r.Status = "cancelled"
	r.UpdatedAt = time.Now()
}

func (r *Reservation) MarkAsNoShow() {
	r.Status = "no_show"
	r.UpdatedAt = time.Now()
}
