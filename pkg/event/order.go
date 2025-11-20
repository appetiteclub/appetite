package event

import "time"

const (
	OrderItemsTopic          = "orders.items"
	EventOrderItemCreated    = "order.item.created"
	EventOrderItemUpdated    = "order.item.updated"
	EventOrderItemCancelled  = "order.item.cancelled"
)

// OrderItemEvent represents an order item event published to NATS.
// This event is consumed by the Kitchen service to create tickets.
type OrderItemEvent struct {
	EventType          string    `json:"event_type"`
	OccurredAt         time.Time `json:"occurred_at"`
	OrderID            string    `json:"order_id"`
	OrderItemID        string    `json:"order_item_id"`
	MenuItemID         string    `json:"menu_item_id"`
	Quantity           int       `json:"quantity"`
	Notes              string    `json:"notes,omitempty"`
	RequiresProduction bool      `json:"requires_production"`
	ProductionStation  string    `json:"production_station,omitempty"`

	// Denormalized data for Kitchen/Operations display
	MenuItemName string `json:"menu_item_name,omitempty"`
	StationName  string `json:"station_name,omitempty"`
	TableNumber  string `json:"table_number,omitempty"`
	TableID      string `json:"table_id,omitempty"`
}
