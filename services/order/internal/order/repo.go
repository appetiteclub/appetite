package order

import (
	"context"

	"github.com/google/uuid"
)

type OrderRepo interface {
	Create(ctx context.Context, order *Order) error
	Get(ctx context.Context, id uuid.UUID) (*Order, error)
	List(ctx context.Context) ([]*Order, error)
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
