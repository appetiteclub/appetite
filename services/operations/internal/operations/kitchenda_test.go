package operations

import (
	"context"
	"testing"
	"time"
)

func TestNewKitchenDataAccess(t *testing.T) {
	da := NewKitchenDataAccess(nil)
	if da == nil {
		t.Error("NewKitchenDataAccess() returned nil")
	}
}

func TestKitchenDataAccessListTicketsNilClient(t *testing.T) {
	da := &KitchenDataAccess{client: nil}

	_, err := da.ListTickets(context.Background())
	if err == nil {
		t.Error("ListTickets() with nil client should return error")
	}
}

func TestKitchenDataAccessListTicketsNilDA(t *testing.T) {
	var da *KitchenDataAccess

	_, err := da.ListTickets(context.Background())
	if err == nil {
		t.Error("ListTickets() with nil DA should return error")
	}
}

func TestKitchenDataAccessListTicketsByOrderNilClient(t *testing.T) {
	da := &KitchenDataAccess{client: nil}

	_, err := da.ListTicketsByOrder(context.Background(), "order-1")
	if err == nil {
		t.Error("ListTicketsByOrder() with nil client should return error")
	}
}

func TestKitchenDataAccessListTicketsByOrderEmptyOrderID(t *testing.T) {
	da := &KitchenDataAccess{client: nil}

	_, err := da.ListTicketsByOrder(context.Background(), "")
	if err == nil {
		t.Error("ListTicketsByOrder() with empty orderID should return error")
	}
}

func TestKitchenDataAccessGetTicketNilClient(t *testing.T) {
	da := &KitchenDataAccess{client: nil}

	_, err := da.GetTicket(context.Background(), "ticket-1")
	if err == nil {
		t.Error("GetTicket() with nil client should return error")
	}
}

func TestKitchenDataAccessGetTicketNilDA(t *testing.T) {
	var da *KitchenDataAccess

	_, err := da.GetTicket(context.Background(), "ticket-1")
	if err == nil {
		t.Error("GetTicket() with nil DA should return error")
	}
}

func TestKitchenDataAccessTransitionTicketNilClient(t *testing.T) {
	da := &KitchenDataAccess{client: nil}

	_, err := da.TransitionTicket(context.Background(), "ticket-1", "start")
	if err == nil {
		t.Error("TransitionTicket() with nil client should return error")
	}
}

func TestKitchenDataAccessTransitionTicketEmptyParams(t *testing.T) {
	da := &KitchenDataAccess{client: nil}

	tests := []struct {
		name     string
		ticketID string
		action   string
	}{
		{
			name:     "emptyTicketID",
			ticketID: "",
			action:   "start",
		},
		{
			name:     "emptyAction",
			ticketID: "ticket-1",
			action:   "",
		},
		{
			name:     "bothEmpty",
			ticketID: "",
			action:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := da.TransitionTicket(context.Background(), tt.ticketID, tt.action)
			if err == nil {
				t.Error("TransitionTicket() with empty params should return error")
			}
		})
	}
}

func TestKitchenDataAccessUpdateTicketStatusNilClient(t *testing.T) {
	da := &KitchenDataAccess{client: nil}

	err := da.UpdateTicketStatus(context.Background(), nil)
	if err == nil {
		t.Error("UpdateTicketStatus() with nil client should return error")
	}
}

func TestKitchenTicketResourceFields(t *testing.T) {
	now := time.Now()
	startedAt := now.Add(-10 * time.Minute)
	finishedAt := now.Add(-5 * time.Minute)
	deliveredAt := now
	reasonCode := "out-of-stock"

	ticket := kitchenTicketResource{
		ID:               "ticket-1",
		OrderID:          "order-1",
		OrderItemID:      "item-1",
		MenuItemID:       "menu-1",
		Station:          "grill",
		Quantity:         2,
		Status:           "ready",
		ReasonCodeID:     &reasonCode,
		Notes:            "no onions",
		DecisionRequired: true,
		DecisionPayload:  []byte(`{"action":"confirm"}`),
		MenuItemName:     "Burger",
		StationName:      "Grill Station",
		TableNumber:      "5",
		CreatedAt:        now.Add(-15 * time.Minute),
		UpdatedAt:        now,
		StartedAt:        &startedAt,
		FinishedAt:       &finishedAt,
		DeliveredAt:      &deliveredAt,
		ModelVersion:     2,
	}

	if ticket.ID != "ticket-1" {
		t.Errorf("ID = %q, want %q", ticket.ID, "ticket-1")
	}
	if ticket.OrderID != "order-1" {
		t.Errorf("OrderID = %q, want %q", ticket.OrderID, "order-1")
	}
	if ticket.Station != "grill" {
		t.Errorf("Station = %q, want %q", ticket.Station, "grill")
	}
	if ticket.Quantity != 2 {
		t.Errorf("Quantity = %d, want %d", ticket.Quantity, 2)
	}
	if ticket.Status != "ready" {
		t.Errorf("Status = %q, want %q", ticket.Status, "ready")
	}
	if ticket.ReasonCodeID == nil || *ticket.ReasonCodeID != "out-of-stock" {
		t.Error("ReasonCodeID not set correctly")
	}
	if ticket.MenuItemName != "Burger" {
		t.Errorf("MenuItemName = %q, want %q", ticket.MenuItemName, "Burger")
	}
	if ticket.StationName != "Grill Station" {
		t.Errorf("StationName = %q, want %q", ticket.StationName, "Grill Station")
	}
	if !ticket.DecisionRequired {
		t.Error("DecisionRequired should be true")
	}
	if ticket.StartedAt == nil {
		t.Error("StartedAt should not be nil")
	}
	if ticket.FinishedAt == nil {
		t.Error("FinishedAt should not be nil")
	}
	if ticket.DeliveredAt == nil {
		t.Error("DeliveredAt should not be nil")
	}
	if ticket.ModelVersion != 2 {
		t.Errorf("ModelVersion = %d, want %d", ticket.ModelVersion, 2)
	}
}

func TestKitchenTicketResourceWithNilPointers(t *testing.T) {
	ticket := kitchenTicketResource{
		ID:           "ticket-1",
		Status:       "created",
		ReasonCodeID: nil,
		StartedAt:    nil,
		FinishedAt:   nil,
		DeliveredAt:  nil,
	}

	if ticket.ReasonCodeID != nil {
		t.Error("ReasonCodeID should be nil")
	}
	if ticket.StartedAt != nil {
		t.Error("StartedAt should be nil")
	}
	if ticket.FinishedAt != nil {
		t.Error("FinishedAt should be nil")
	}
	if ticket.DeliveredAt != nil {
		t.Error("DeliveredAt should be nil")
	}
}
