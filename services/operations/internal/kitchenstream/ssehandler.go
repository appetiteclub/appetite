package kitchenstream

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	kitchenproto "github.com/appetiteclub/appetite/services/operations/internal/kitchenstream/proto"
	orderproto "github.com/appetiteclub/appetite/services/operations/internal/orderstream/proto"
	"github.com/appetiteclub/apt"
	aqmtemplate "github.com/appetiteclub/apt/template"
	"github.com/google/uuid"
)

// OrderDataProvider is an interface for fetching order item data
type OrderDataProvider interface {
	GetOrderItem(itemID string) (OrderItemData, error)
}

// OrderItemData represents the view model for an order item
type OrderItemData struct {
	ID                 string
	DishName           string
	Quantity           int
	UnitPrice          string
	Total              string
	Status             string
	StatusLabel        string
	StatusClass        string
	Category           string
	GroupName          string
	Notes              string
	CreatedAt          string
	RequiresProduction bool
}

// OrderStreamClient interface for Order stream subscription
type OrderStreamClient interface {
	Subscribe(subscriberID string) <-chan *orderproto.OrderItemEvent
	Unsubscribe(subscriberID string)
}

// SSEHandler handles Server-Sent Events for Kitchen ticket updates and Order item updates
type SSEHandler struct {
	kitchenClient *Client
	orderClient   OrderStreamClient
	logger        apt.Logger
	tmplMgr       *aqmtemplate.Manager
	orderDataProv OrderDataProvider
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler(kitchenClient *Client, logger apt.Logger, tmplMgr *aqmtemplate.Manager, orderDataProv OrderDataProvider) *SSEHandler {
	return &SSEHandler{
		kitchenClient: kitchenClient,
		logger:        logger,
		tmplMgr:       tmplMgr,
		orderDataProv: orderDataProv,
	}
}

// SetOrderClient sets the Order stream client (optional)
func (h *SSEHandler) SetOrderClient(client OrderStreamClient) {
	h.orderClient = client
}

// ServeHTTP implements http.Handler for SSE endpoint
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	subscriberID := uuid.New().String()
	h.logger.Info("new SSE connection", "subscriber_id", subscriberID)

	// Subscribe to Kitchen events
	kitchenEventChan := h.kitchenClient.Subscribe(subscriberID)
	defer h.kitchenClient.Unsubscribe(subscriberID)

	// Subscribe to Order events if client is available
	var orderEventChan <-chan *orderproto.OrderItemEvent
	if h.orderClient != nil {
		orderEventChan = h.orderClient.Subscribe(subscriberID)
		defer h.orderClient.Unsubscribe(subscriberID)
	}

	// Send initial comment to establish connection
	fmt.Fprintf(w, ": connected\n\n")

	// Configure retry interval for reconnection (in milliseconds)
	fmt.Fprintf(w, "retry: 2000\n\n")

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Send keepalive every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			h.logger.Info("SSE client disconnected", "subscriber_id", subscriberID)
			return

		case <-ticker.C:
			// Send keepalive comment
			fmt.Fprintf(w, ": keepalive\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

		case evt, ok := <-kitchenEventChan:
			if !ok {
				h.logger.Info("kitchen event channel closed", "subscriber_id", subscriberID)
				return
			}

			// Send two types of SSE events:
			// ticket-update for Kitchen Kanban (always)
			// order-item-update for Order Modal (only for status changes)

			// Render ticket card for Kanban dashboard
			ticketHTML, err := h.renderTicketCard(evt)
			if err != nil {
				h.logger.Error("failed to render ticket card", "error", err)
			} else {
				fmt.Fprintf(w, "event: ticket-update\n")
				fmt.Fprintf(w, "data: %s\n\n", ticketHTML)
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}

			// If this is a status change and has order_item_id, send order-item-update
			if evt.EventType == "kitchen.ticket.status_changed" && evt.OrderItemId != "" {
				itemHTML, err := h.renderOrderItemRowFromKitchen(evt)
				if err != nil {
					h.logger.Error("failed to render order item row", "error", err)
				} else {
					sendSSEEvent(w, "order-item-update", itemHTML)
				}
			}

		case evt, ok := <-orderEventChan:
			if !ok {
				h.logger.Info("order event channel closed", "subscriber_id", subscriberID)
				return
			}

			h.logger.Info("received order item event from Order stream",
				"subscriber_id", subscriberID,
				"event_type", evt.EventType,
				"order_item_id", evt.OrderItemId,
				"new_status", evt.NewStatus,
			)

			// Order item status changed - send order-item-update
			itemHTML, err := h.renderOrderItemRowFromOrder(evt)
			if err != nil {
				h.logger.Error("failed to render order item row from order event", "error", err)
			} else {
				sendSSEEvent(w, "order-item-update", itemHTML)
			}
		}
	}
}

// sendSSEEvent sends an SSE event with properly formatted multi-line data
func sendSSEEvent(w http.ResponseWriter, eventType string, data string) {
	// Remove any trailing/leading whitespace
	data = strings.TrimSpace(data)

	// SSE format: each line of data must be prefixed with "data: "
	fmt.Fprintf(w, "event: %s\n", eventType)

	// Split data into lines and prefix each with "data: "
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		fmt.Fprintf(w, "data: %s\n", line)
	}

	// Empty line marks end of event
	fmt.Fprintf(w, "\n")

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// renderTicketCard renders a ticket card for the Kitchen Kanban dashboard
func (h *SSEHandler) renderTicketCard(evt *kitchenproto.KitchenTicketEvent) (string, error) {
	// For now, trigger page reload to update Kanban
	// TODO: Render actual ticket_card.html fragment with proper data
	return `<script>window.location.reload();</script>`, nil
}

// renderOrderItemRowFromKitchen renders an order item row from a Kitchen ticket event
func (h *SSEHandler) renderOrderItemRowFromKitchen(evt *kitchenproto.KitchenTicketEvent) (string, error) {
	// Fetch the current order item data
	itemData, err := h.orderDataProv.GetOrderItem(evt.OrderItemId)
	if err != nil {
		h.logger.Error("failed to fetch order item data", "order_item_id", evt.OrderItemId, "error", err)
		return "", err
	}

	// Render the order_item_row fragment
	tmpl, err := h.tmplMgr.Get("order_item_row.html")
	if err != nil {
		h.logger.Error("failed to load template", "error", err)
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "order_item_row", itemData); err != nil {
		h.logger.Error("failed to render template", "error", err)
		return "", err
	}

	// Return just the order_item_row HTML without wrapper div
	return buf.String(), nil
}

// renderOrderItemRowFromOrder renders an order item row from an Order service event
func (h *SSEHandler) renderOrderItemRowFromOrder(evt *orderproto.OrderItemEvent) (string, error) {
	// Fetch the current order item data
	itemData, err := h.orderDataProv.GetOrderItem(evt.OrderItemId)
	if err != nil {
		h.logger.Error("failed to fetch order item data", "order_item_id", evt.OrderItemId, "error", err)
		return "", err
	}

	// Render the order_item_row fragment
	tmpl, err := h.tmplMgr.Get("order_item_row.html")
	if err != nil {
		h.logger.Error("failed to load template", "error", err)
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "order_item_row", itemData); err != nil {
		h.logger.Error("failed to render template", "error", err)
		return "", err
	}

	// Return just the order_item_row HTML without wrapper div
	return buf.String(), nil
}
