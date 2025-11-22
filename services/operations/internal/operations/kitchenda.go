package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/aquamarinepk/aqm"
	"github.com/go-chi/chi/v5"
)

// kitchenTicketResource mirrors the ticket JSON returned by the kitchen service.
type kitchenTicketResource struct {
	ID               string     `json:"id"`
	OrderID          string     `json:"order_id"`
	OrderItemID      string     `json:"order_item_id"`
	MenuItemID       string     `json:"menu_item_id"`
	Station          string     `json:"station"`
	Quantity         int        `json:"quantity"`
	Status           string     `json:"status"`
	ReasonCodeID     *string    `json:"reason_code_id"`
	Notes            string     `json:"notes"`
	DecisionRequired bool       `json:"decision_required"`
	DecisionPayload  []byte     `json:"decision_payload"`

	// Denormalized data for display
	MenuItemName string `json:"menu_item_name"`
	StationName  string `json:"station_name"`
	TableNumber  string `json:"table_number"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at"`
	DeliveredAt *time.Time `json:"delivered_at"`

	ModelVersion int `json:"model_version"`
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

func (da *KitchenDataAccess) ListTicketsByOrder(ctx context.Context, orderID string) ([]kitchenTicketResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("kitchen client not configured")
	}
	if orderID == "" {
		return nil, fmt.Errorf("missing order id")
	}

	path := fmt.Sprintf("/tickets?order_id=%s", url.QueryEscape(orderID))
	resp, err := da.client.Request(ctx, "GET", path, nil)
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

func (da *KitchenDataAccess) UpdateTicketStatus(ctx context.Context, r *http.Request) error {
	if da == nil || da.client == nil {
		return fmt.Errorf("kitchen client not configured")
	}

	// Extract ticket ID from URL
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		return fmt.Errorf("missing ticket ID")
	}

	// Read request body - expects {"status": "status-code"}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	// Parse the body to get the status
	var reqBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &reqBody); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}

	// Forward to Kitchen service
	path := fmt.Sprintf("/tickets/%s/status", ticketID)
	_, err = da.client.Request(ctx, "PATCH", path, reqBody)
	if err != nil {
		return fmt.Errorf("kitchen service request failed: %w", err)
	}

	return nil
}
