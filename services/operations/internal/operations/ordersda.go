package operations

import (
	"context"
	"fmt"
	"time"

	"github.com/appetiteclub/apt"
)

// orderResource mirrors the aggregate returned by the order service.
type orderResource struct {
	ID        string    `json:"id"`
	TableID   string    `json:"table_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// orderItemResource represents a single item inside an order.
type orderItemResource struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	GroupID   *string   `json:"group_id"`
	DishName  string    `json:"dish_name"`
	Category  string    `json:"category"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	Status    string    `json:"status"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type orderGroupResource struct {
	ID        string `json:"id"`
	OrderID   string `json:"order_id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

// CreateOrderRequest defines the payload supported by the order service.
type CreateOrderRequest struct {
	TableID string `json:"table_id"`
}

// OrderDataAccess centralizes decoding of order service responses.
type OrderDataAccess struct {
	client *apt.ServiceClient
}

func NewOrderDataAccess(client *apt.ServiceClient) *OrderDataAccess {
	return &OrderDataAccess{client: client}
}

func (da *OrderDataAccess) ListOrders(ctx context.Context) ([]orderResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("order client not configured")
	}

	resp, err := da.client.List(ctx, "orders")
	if err != nil {
		return nil, err
	}

	var orders []orderResource
	if err := decodeSuccessResponse(resp, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

func (da *OrderDataAccess) GetOrder(ctx context.Context, id string) (*orderResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("order client not configured")
	}

	resp, err := da.client.Get(ctx, "orders", id)
	if err != nil {
		return nil, err
	}

	var order orderResource
	if err := decodeSuccessResponse(resp, &order); err != nil {
		return nil, err
	}

	return &order, nil
}

func (da *OrderDataAccess) ListOrderItems(ctx context.Context, orderID string) ([]orderItemResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("order client not configured")
	}
	if orderID == "" {
		return nil, fmt.Errorf("missing order id")
	}

	path := fmt.Sprintf("/orders/%s/items", orderID)
	resp, err := da.client.Request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var items []orderItemResource
	if err := decodeSuccessResponse(resp, &items); err != nil {
		return nil, err
	}

	return items, nil
}

func (da *OrderDataAccess) CreateOrder(ctx context.Context, payload CreateOrderRequest) (*orderResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("order client not configured")
	}

	resp, err := da.client.Create(ctx, "orders", payload)
	if err != nil {
		return nil, err
	}

	var order orderResource
	if err := decodeSuccessResponse(resp, &order); err != nil {
		return nil, err
	}

	return &order, nil
}

func (da *OrderDataAccess) GetOrderItem(ctx context.Context, itemID string) (*orderItemResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("order client not configured")
	}
	if itemID == "" {
		return nil, fmt.Errorf("missing item id")
	}

	path := fmt.Sprintf("/order-items/%s", itemID)
	resp, err := da.client.Request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var item orderItemResource
	if err := decodeSuccessResponse(resp, &item); err != nil {
		return nil, err
	}

	return &item, nil
}

func (da *OrderDataAccess) ListOrderGroups(ctx context.Context, orderID string) ([]orderGroupResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("order client not configured")
	}
	if orderID == "" {
		return nil, fmt.Errorf("missing order id")
	}

	path := fmt.Sprintf("/orders/%s/groups", orderID)
	resp, err := da.client.Request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var groups []orderGroupResource
	if err := decodeSuccessResponse(resp, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}
