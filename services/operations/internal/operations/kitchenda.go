package operations

import (
	"context"
	"fmt"
	"time"

	"github.com/aquamarinepk/aqm"
)

// kitchenTicketResource mirrors the ticket JSON returned by the kitchen service.
type kitchenTicketResource struct {
	ID               string     `json:"id"`
	OrderID          string     `json:"order_id"`
	OrderItemID      string     `json:"order_item_id"`
	MenuItemID       string     `json:"menu_item_id"`
	StationID        string     `json:"station_id"`
	Quantity         int        `json:"quantity"`
	StatusID         string     `json:"status_id"`
	ReasonCodeID     *string    `json:"reason_code_id"`
	Notes            string     `json:"notes"`
	DecisionRequired bool       `json:"decision_required"`
	DecisionPayload  []byte     `json:"decision_payload"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	StartedAt        *time.Time `json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at"`
	DeliveredAt      *time.Time `json:"delivered_at"`
	ModelVersion     int        `json:"model_version"`
}

// KitchenDataAccess wraps the low-level kitchen API.
type KitchenDataAccess struct {
	client *aqm.ServiceClient
}

func NewKitchenDataAccess(client *aqm.ServiceClient) *KitchenDataAccess {
	return &KitchenDataAccess{client: client}
}

func (da *KitchenDataAccess) ListTickets(ctx context.Context) ([]kitchenTicketResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("kitchen client not configured")
	}

	resp, err := da.client.List(ctx, "tickets")
	if err != nil {
		return nil, err
	}

	var payload struct {
		Tickets []kitchenTicketResource `json:"tickets"`
	}
	if err := decodeSuccessResponse(resp, &payload); err != nil {
		return nil, err
	}

	return payload.Tickets, nil
}

func (da *KitchenDataAccess) GetTicket(ctx context.Context, id string) (*kitchenTicketResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("kitchen client not configured")
	}

	resp, err := da.client.Get(ctx, "tickets", id)
	if err != nil {
		return nil, err
	}

	var ticket kitchenTicketResource
	if err := decodeSuccessResponse(resp, &ticket); err != nil {
		return nil, err
	}

	return &ticket, nil
}

func (da *KitchenDataAccess) TransitionTicket(ctx context.Context, ticketID, action string) (*kitchenTicketResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("kitchen client not configured")
	}
	if ticketID == "" || action == "" {
		return nil, fmt.Errorf("missing ticket transition information")
	}

	path := fmt.Sprintf("/tickets/%s/%s", ticketID, action)
	resp, err := da.client.Request(ctx, "PATCH", path, nil)
	if err != nil {
		return nil, err
	}

	var ticket kitchenTicketResource
	if err := decodeSuccessResponse(resp, &ticket); err != nil {
		return nil, err
	}

	return &ticket, nil
}
