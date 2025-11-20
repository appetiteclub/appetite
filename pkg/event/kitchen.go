package event

import "time"

const (
	KitchenTicketsTopic            = "kitchen.tickets"
	EventKitchenTicketCreated      = "kitchen.ticket.created"
	EventKitchenTicketStatusChange = "kitchen.ticket.status_changed"
)

type KitchenTicketEventMetadata struct {
	EventType   string    `json:"event_type"`
	OccurredAt  time.Time `json:"occurred_at"`
	TicketID    string    `json:"ticket_id"`
	OrderID     string    `json:"order_id"`
	OrderItemID string    `json:"order_item_id,omitempty"`
	MenuItemID  string    `json:"menu_item_id,omitempty"`
	StationID   string    `json:"station_id"`

	// Denormalized data for display (Kanban UI)
	MenuItemName string `json:"menu_item_name,omitempty"`
	StationName  string `json:"station_name,omitempty"`
	TableNumber  string `json:"table_number,omitempty"`
}

type KitchenTicketCreatedEvent struct {
	KitchenTicketEventMetadata
	StatusID string `json:"status_id"`
	Quantity int    `json:"quantity"`
	Notes    string `json:"notes,omitempty"`
}

type KitchenTicketStatusChangedEvent struct {
	KitchenTicketEventMetadata
	NewStatusID      string     `json:"new_status_id"`
	PreviousStatusID string     `json:"previous_status_id"`
	ReasonCodeID     string     `json:"reason_code_id,omitempty"`
	Notes            string     `json:"notes,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	DeliveredAt      *time.Time `json:"delivered_at,omitempty"`
}
