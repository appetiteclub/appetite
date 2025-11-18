package pkg

import "time"

const (
	// TableStatusTopic delivers authoritative status changes for tables.
	TableStatusTopic = "tables.status"
	// TableIntentTopic communicates queued transitions that could not be applied immediately.
	TableIntentTopic = "tables.intent"
	// OrderTableTopic groups events emitted by the order service that relate to table operations.
	OrderTableTopic = "orders.tables"

	// EventTableStatusChanged identifies a table status change event payload.
	EventTableStatusChanged = "table.status.changed"
	// EventTableIntentQueued identifies a queued table intent payload.
	EventTableIntentQueued = "table.intent.queued"
	// EventOrderTableRejected identifies a rejection emitted by the order service.
	EventOrderTableRejected = "order.table.rejected"
)

// TableStatusEvent captures the minimal information the order service needs to
// reason about a table's availability.
type TableStatusEvent struct {
	EventType      string    `json:"event_type"`
	TableID        string    `json:"table_id"`
	Status         string    `json:"status"`
	PreviousStatus string    `json:"previous_status,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	Source         string    `json:"source,omitempty"`
	OccurredAt     time.Time `json:"occurred_at"`
}

// TableIntentEvent communicates that a requested transition was deferred.
type TableIntentEvent struct {
	EventType      string    `json:"event_type"`
	TableID        string    `json:"table_id"`
	RequestedState string    `json:"requested_state"`
	BlockedBy      string    `json:"blocked_by"`
	Reason         string    `json:"reason,omitempty"`
	Source         string    `json:"source,omitempty"`
	OccurredAt     time.Time `json:"occurred_at"`
}

// OrderTableRejectionEvent captures rejections performed by the order service
// whenever a table transition blocks an operation.
type OrderTableRejectionEvent struct {
	EventType  string    `json:"event_type"`
	TableID    string    `json:"table_id"`
	OrderID    string    `json:"order_id,omitempty"`
	Action     string    `json:"action"`
	Reason     string    `json:"reason"`
	Status     string    `json:"status"`
	OccurredAt time.Time `json:"occurred_at"`
}
