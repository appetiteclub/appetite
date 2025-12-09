package operations

import (
	"context"
	"testing"
	"time"
)

func TestNewOrderDataAccess(t *testing.T) {
	da := NewOrderDataAccess(nil)
	if da == nil {
		t.Error("NewOrderDataAccess() returned nil")
	}
}

func TestOrderDataAccessListOrdersNilClient(t *testing.T) {
	da := &OrderDataAccess{client: nil}

	_, err := da.ListOrders(context.Background())
	if err == nil {
		t.Error("ListOrders() with nil client should return error")
	}
}

func TestOrderDataAccessListOrdersNilDA(t *testing.T) {
	var da *OrderDataAccess

	_, err := da.ListOrders(context.Background())
	if err == nil {
		t.Error("ListOrders() with nil DA should return error")
	}
}

func TestOrderDataAccessGetOrderNilClient(t *testing.T) {
	da := &OrderDataAccess{client: nil}

	_, err := da.GetOrder(context.Background(), "order-1")
	if err == nil {
		t.Error("GetOrder() with nil client should return error")
	}
}

func TestOrderDataAccessGetOrderNilDA(t *testing.T) {
	var da *OrderDataAccess

	_, err := da.GetOrder(context.Background(), "order-1")
	if err == nil {
		t.Error("GetOrder() with nil DA should return error")
	}
}

func TestOrderDataAccessListOrderItemsNilClient(t *testing.T) {
	da := &OrderDataAccess{client: nil}

	_, err := da.ListOrderItems(context.Background(), "order-1")
	if err == nil {
		t.Error("ListOrderItems() with nil client should return error")
	}
}

func TestOrderDataAccessListOrderItemsEmptyOrderID(t *testing.T) {
	da := &OrderDataAccess{client: nil}

	_, err := da.ListOrderItems(context.Background(), "")
	if err == nil {
		t.Error("ListOrderItems() with empty orderID should return error")
	}
}

func TestOrderDataAccessCreateOrderNilClient(t *testing.T) {
	da := &OrderDataAccess{client: nil}

	_, err := da.CreateOrder(context.Background(), CreateOrderRequest{TableID: "table-1"})
	if err == nil {
		t.Error("CreateOrder() with nil client should return error")
	}
}

func TestOrderDataAccessCreateOrderNilDA(t *testing.T) {
	var da *OrderDataAccess

	_, err := da.CreateOrder(context.Background(), CreateOrderRequest{TableID: "table-1"})
	if err == nil {
		t.Error("CreateOrder() with nil DA should return error")
	}
}

func TestOrderDataAccessGetOrderItemNilClient(t *testing.T) {
	da := &OrderDataAccess{client: nil}

	_, err := da.GetOrderItem(context.Background(), "item-1")
	if err == nil {
		t.Error("GetOrderItem() with nil client should return error")
	}
}

func TestOrderDataAccessGetOrderItemEmptyItemID(t *testing.T) {
	da := &OrderDataAccess{client: nil}

	_, err := da.GetOrderItem(context.Background(), "")
	if err == nil {
		t.Error("GetOrderItem() with empty itemID should return error")
	}
}

func TestOrderDataAccessListOrderGroupsNilClient(t *testing.T) {
	da := &OrderDataAccess{client: nil}

	_, err := da.ListOrderGroups(context.Background(), "order-1")
	if err == nil {
		t.Error("ListOrderGroups() with nil client should return error")
	}
}

func TestOrderDataAccessListOrderGroupsEmptyOrderID(t *testing.T) {
	da := &OrderDataAccess{client: nil}

	_, err := da.ListOrderGroups(context.Background(), "")
	if err == nil {
		t.Error("ListOrderGroups() with empty orderID should return error")
	}
}

func TestOrderResourceFields(t *testing.T) {
	now := time.Now()
	order := orderResource{
		ID:        "order-1",
		TableID:   "table-1",
		Status:    "pending",
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now,
	}

	if order.ID != "order-1" {
		t.Errorf("ID = %q, want %q", order.ID, "order-1")
	}
	if order.TableID != "table-1" {
		t.Errorf("TableID = %q, want %q", order.TableID, "table-1")
	}
	if order.Status != "pending" {
		t.Errorf("Status = %q, want %q", order.Status, "pending")
	}
}

func TestOrderItemResourceFields(t *testing.T) {
	now := time.Now()
	groupID := "group-1"
	item := orderItemResource{
		ID:        "item-1",
		OrderID:   "order-1",
		GroupID:   &groupID,
		DishName:  "Burger",
		Category:  "entree",
		Quantity:  2,
		Price:     15.99,
		Status:    "pending",
		Notes:     "no onions",
		CreatedAt: now.Add(-30 * time.Minute),
		UpdatedAt: now,
	}

	if item.ID != "item-1" {
		t.Errorf("ID = %q, want %q", item.ID, "item-1")
	}
	if item.OrderID != "order-1" {
		t.Errorf("OrderID = %q, want %q", item.OrderID, "order-1")
	}
	if item.GroupID == nil || *item.GroupID != "group-1" {
		t.Error("GroupID not set correctly")
	}
	if item.DishName != "Burger" {
		t.Errorf("DishName = %q, want %q", item.DishName, "Burger")
	}
	if item.Category != "entree" {
		t.Errorf("Category = %q, want %q", item.Category, "entree")
	}
	if item.Quantity != 2 {
		t.Errorf("Quantity = %d, want %d", item.Quantity, 2)
	}
	if item.Price != 15.99 {
		t.Errorf("Price = %v, want %v", item.Price, 15.99)
	}
	if item.Status != "pending" {
		t.Errorf("Status = %q, want %q", item.Status, "pending")
	}
	if item.Notes != "no onions" {
		t.Errorf("Notes = %q, want %q", item.Notes, "no onions")
	}
}

func TestOrderItemResourceWithNilGroupID(t *testing.T) {
	item := orderItemResource{
		ID:      "item-1",
		GroupID: nil,
	}

	if item.GroupID != nil {
		t.Error("GroupID should be nil for ungrouped items")
	}
}

func TestOrderGroupResourceFields(t *testing.T) {
	group := orderGroupResource{
		ID:        "group-1",
		OrderID:   "order-1",
		Name:      "Main",
		IsDefault: true,
	}

	if group.ID != "group-1" {
		t.Errorf("ID = %q, want %q", group.ID, "group-1")
	}
	if group.OrderID != "order-1" {
		t.Errorf("OrderID = %q, want %q", group.OrderID, "order-1")
	}
	if group.Name != "Main" {
		t.Errorf("Name = %q, want %q", group.Name, "Main")
	}
	if !group.IsDefault {
		t.Error("IsDefault should be true")
	}
}

func TestCreateOrderRequestFields(t *testing.T) {
	req := CreateOrderRequest{
		TableID: "table-5",
	}

	if req.TableID != "table-5" {
		t.Errorf("TableID = %q, want %q", req.TableID, "table-5")
	}
}
