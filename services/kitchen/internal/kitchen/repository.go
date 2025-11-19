package kitchen

import "context"

type TicketFilter struct {
	StationID   *StationID
	StatusID    *StatusID
	OrderID     *OrderID
	OrderItemID *OrderItemID
	Limit       int
	Offset      int
}

type TicketRepository interface {
	Create(ctx context.Context, t *Ticket) error
	Update(ctx context.Context, t *Ticket) error
	FindByID(ctx context.Context, id TicketID) (*Ticket, error)
	FindByOrderItemID(ctx context.Context, id OrderItemID) (*Ticket, error)
	List(ctx context.Context, filter TicketFilter) ([]Ticket, error)
}
