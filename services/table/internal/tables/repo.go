package tables

import (
	"context"

	"github.com/google/uuid"
)

type TableRepo interface {
	Create(ctx context.Context, table *Table) error
	Get(ctx context.Context, id uuid.UUID) (*Table, error)
	GetByNumber(ctx context.Context, number string) (*Table, error)
	List(ctx context.Context) ([]*Table, error)
	ListByStatus(ctx context.Context, status string) ([]*Table, error)
	Save(ctx context.Context, table *Table) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type GroupRepo interface {
	Create(ctx context.Context, group *Group) error
	Get(ctx context.Context, id uuid.UUID) (*Group, error)
	ListByTable(ctx context.Context, tableID uuid.UUID) ([]*Group, error)
	Save(ctx context.Context, group *Group) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type OrderRepo interface {
	Create(ctx context.Context, order *Order) error
	Get(ctx context.Context, id uuid.UUID) (*Order, error)
	ListByTable(ctx context.Context, tableID uuid.UUID) ([]*Order, error)
	ListByStatus(ctx context.Context, status string) ([]*Order, error)
	Save(ctx context.Context, order *Order) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type OrderItemRepo interface {
	Create(ctx context.Context, item *OrderItem) error
	Get(ctx context.Context, id uuid.UUID) (*OrderItem, error)
	ListByOrder(ctx context.Context, orderID uuid.UUID) ([]*OrderItem, error)
	ListByGroup(ctx context.Context, groupID uuid.UUID) ([]*OrderItem, error)
	Save(ctx context.Context, item *OrderItem) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ReservationRepo interface {
	Create(ctx context.Context, reservation *Reservation) error
	Get(ctx context.Context, id uuid.UUID) (*Reservation, error)
	List(ctx context.Context) ([]*Reservation, error)
	ListByDate(ctx context.Context, date string) ([]*Reservation, error)
	ListByTable(ctx context.Context, tableID uuid.UUID) ([]*Reservation, error)
	Save(ctx context.Context, reservation *Reservation) error
	Delete(ctx context.Context, id uuid.UUID) error
}
