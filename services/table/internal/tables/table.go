package tables

import (
	"time"

	"github.com/aquamarinepk/aqm"
	"github.com/google/uuid"
)

type Table struct {
	ID          uuid.UUID  `json:"id" bson:"_id"`
	Number      string     `json:"number" bson:"number"`
	Status      string     `json:"status" bson:"status"`
	GuestCount  int        `json:"guest_count" bson:"guest_count"`
	AssignedTo  *uuid.UUID `json:"assigned_to,omitempty" bson:"assigned_to,omitempty"`
	Notes       []Note     `json:"notes,omitempty" bson:"notes,omitempty"`
	// NOTE: CurrentBill denotes a denormalized view of charges for the table, but
	// no code populates it today. We should evaluate whether the field is worth
	// keeping or if billing should live exclusively in the order service.
	CurrentBill *Bill      `json:"current_bill,omitempty" bson:"current_bill,omitempty"`
	CreatedAt   time.Time  `json:"created_at" bson:"created_at"`
	CreatedBy   string     `json:"created_by" bson:"created_by"`
	UpdatedAt   time.Time  `json:"updated_at" bson:"updated_at"`
	UpdatedBy   string     `json:"updated_by" bson:"updated_by"`
}

type Note struct {
	ID        uuid.UUID `json:"id" bson:"id"`
	Content   string    `json:"content" bson:"content"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	CreatedBy string    `json:"created_by" bson:"created_by"`
}

type Bill struct {
	Subtotal float64 `json:"subtotal" bson:"subtotal"`
	Tax      float64 `json:"tax" bson:"tax"`
	Tip      float64 `json:"tip" bson:"tip"`
	Total    float64 `json:"total" bson:"total"`
}

func (t *Table) GetID() uuid.UUID {
	return t.ID
}

func (t *Table) ResourceType() string {
	return "table"
}

func (t *Table) SetID(id uuid.UUID) {
	t.ID = id
}

func NewTable() *Table {
	return &Table{
		ID:     aqm.GenerateNewID(),
		Status: "available",
		Notes:  []Note{},
	}
}

func (t *Table) EnsureID() {
	if t.ID == uuid.Nil {
		t.ID = aqm.GenerateNewID()
	}
}

func (t *Table) BeforeCreate() {
	t.EnsureID()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
}

func (t *Table) BeforeUpdate() {
	t.UpdatedAt = time.Now()
}

func (t *Table) AddNote(content, createdBy string) {
	if t.Notes == nil {
		t.Notes = []Note{}
	}
	note := Note{
		ID:        aqm.GenerateNewID(),
		Content:   content,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
	}
	t.Notes = append(t.Notes, note)
}

func (t *Table) Open(guestCount int, waiterID *uuid.UUID) {
	t.Status = "open"
	t.GuestCount = guestCount
	t.AssignedTo = waiterID
	t.UpdatedAt = time.Now()
}

func (t *Table) Close() {
	t.Status = "available"
	t.GuestCount = 0
	t.AssignedTo = nil
	t.CurrentBill = nil
	t.UpdatedAt = time.Now()
}

func (t *Table) UpdateBill(subtotal, tax, tip float64) {
	t.CurrentBill = &Bill{
		Subtotal: subtotal,
		Tax:      tax,
		Tip:      tip,
		Total:    subtotal + tax + tip,
	}
	t.UpdatedAt = time.Now()
}
