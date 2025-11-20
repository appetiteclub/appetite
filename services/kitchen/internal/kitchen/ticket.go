package kitchen

import (
	"time"

	"github.com/google/uuid"
)

type TicketID = uuid.UUID
type OrderID = uuid.UUID
type OrderItemID = uuid.UUID
type MenuItemID = uuid.UUID
type StationID = uuid.UUID
type StatusID = uuid.UUID
type ReasonCodeID = uuid.UUID

type Ticket struct {
	ID               TicketID      `bson:"_id" json:"id"`
	OrderID          OrderID       `bson:"order_id" json:"order_id"`
	OrderItemID      OrderItemID   `bson:"order_item_id" json:"order_item_id"`
	MenuItemID       MenuItemID    `bson:"menu_item_id" json:"menu_item_id"`
	StationID        StationID     `bson:"station_id" json:"station_id"`
	Quantity         int           `bson:"quantity" json:"quantity"`
	StatusID         StatusID      `bson:"status_id" json:"status_id"`
	ReasonCodeID     *ReasonCodeID `bson:"reason_code_id,omitempty" json:"reason_code_id,omitempty"`
	Notes            string        `bson:"notes,omitempty" json:"notes,omitempty"`
	DecisionRequired bool          `bson:"decision_required" json:"decision_required"`
	DecisionPayload  []byte        `bson:"decision_payload,omitempty" json:"decision_payload,omitempty"`

	// Denormalized data for display purposes
	MenuItemName string `bson:"menu_item_name,omitempty" json:"menu_item_name,omitempty"`
	StationName  string `bson:"station_name,omitempty" json:"station_name,omitempty"`
	TableNumber  string `bson:"table_number,omitempty" json:"table_number,omitempty"`

	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updated_at"`
	StartedAt   *time.Time `bson:"started_at,omitempty" json:"started_at,omitempty"`
	FinishedAt  *time.Time `bson:"finished_at,omitempty" json:"finished_at,omitempty"`
	DeliveredAt *time.Time `bson:"delivered_at,omitempty" json:"delivered_at,omitempty"`

	ModelVersion int `bson:"model_version" json:"model_version"`
}
