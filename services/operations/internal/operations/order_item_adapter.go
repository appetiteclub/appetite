package operations

import (
	"context"
	"fmt"

	"github.com/appetiteclub/appetite/services/operations/internal/kitchenstream"
)

// OrderItemAdapter adapts OrderDataAccess to the kitchenstream.OrderDataProvider interface
type OrderItemAdapter struct {
	orderData *OrderDataAccess
}

func NewOrderItemAdapter(orderData *OrderDataAccess) *OrderItemAdapter {
	return &OrderItemAdapter{
		orderData: orderData,
	}
}

// GetOrderItem implements kitchenstream.OrderDataProvider
func (a *OrderItemAdapter) GetOrderItem(itemID string) (kitchenstream.OrderItemData, error) {
	ctx := context.Background()

	item, err := a.orderData.GetOrderItem(ctx, itemID)
	if err != nil {
		return kitchenstream.OrderItemData{}, err
	}

	// Map orderItemResource to kitchenstream.OrderItemData
	return kitchenstream.OrderItemData{
		ID:                 item.ID,
		DishName:           item.DishName,
		Quantity:           item.Quantity,
		UnitPrice:          fmt.Sprintf("$%.2f", item.Price),
		Total:              fmt.Sprintf("$%.2f", float64(item.Quantity)*item.Price),
		Status:             item.Status,
		StatusLabel:        formatOrderItemStatus(item.Status),
		StatusClass:        formatOrderItemStatusClass(item.Status),
		Category:           item.Category,
		Notes:              item.Notes,
		CreatedAt:          item.CreatedAt.Format("3:04 PM"),
		RequiresProduction: item.Category != "beverage" && item.Category != "dessert",
	}, nil
}

// formatOrderItemStatus converts status codes to human-readable labels
func formatOrderItemStatus(status string) string {
	switch status {
	case "pending":
		return "Pending"
	case "preparing":
		return "Preparing"
	case "ready":
		return "Ready"
	case "delivered":
		return "Delivered"
	case "cancelled":
		return "Cancelled"
	default:
		return status
	}
}

// formatOrderItemStatusClass returns CSS class for status badge
func formatOrderItemStatusClass(status string) string {
	switch status {
	case "pending":
		return "status-pending"
	case "preparing":
		return "status-preparing"
	case "ready":
		return "status-ready"
	case "delivered":
		return "status-delivered"
	case "cancelled":
		return "status-cancelled"
	default:
		return ""
	}
}
