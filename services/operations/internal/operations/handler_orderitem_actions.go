package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aquamarinepk/aqm"
	"github.com/go-chi/chi/v5"
)

// MarkOrderItemDelivered marks an order item as delivered
func (h *Handler) MarkOrderItemDelivered(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.MarkOrderItemDelivered")
	defer finish()

	log := h.log()
	ctx := r.Context()

	itemID := chi.URLParam(r, "id")
	if itemID == "" {
		http.Error(w, "Missing item ID", http.StatusBadRequest)
		return
	}

	// Call Order service to mark as delivered
	orderServiceURL, _ := h.config.GetString("services.order.url")
	if orderServiceURL == "" {
		log.Error("Order service URL not configured")
		http.Error(w, "Order service not configured", http.StatusServiceUnavailable)
		return
	}

	client := aqm.NewServiceClient(orderServiceURL)
	path := fmt.Sprintf("/items/%s/deliver", itemID)

	_, err := client.Request(ctx, "PATCH", path, nil)
	if err != nil {
		log.Errorf("Failed to mark item as delivered: %v", err)
		http.Error(w, "Failed to mark item as delivered", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// CancelOrderItem cancels an order item
func (h *Handler) CancelOrderItem(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.CancelOrderItem")
	defer finish()

	log := h.log()
	ctx := r.Context()

	itemID := chi.URLParam(r, "id")
	if itemID == "" {
		http.Error(w, "Missing item ID", http.StatusBadRequest)
		return
	}

	// Call Order service to cancel the item
	orderServiceURL, _ := h.config.GetString("services.order.url")
	if orderServiceURL == "" {
		log.Error("Order service URL not configured")
		http.Error(w, "Order service not configured", http.StatusServiceUnavailable)
		return
	}

	client := aqm.NewServiceClient(orderServiceURL)
	path := fmt.Sprintf("/items/%s/cancel", itemID)

	_, err := client.Request(ctx, "PATCH", path, nil)
	if err != nil {
		log.Errorf("Failed to cancel item: %v", err)
		http.Error(w, "Failed to cancel item", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// CloseOrder closes an order, auto-processing pending/ready items
// Query params:
//   - force=true: auto-process pending (cancel) and ready (deliver) items
//   - takeaway=true: treat preparing items as takeaway (table goes to clearing)
func (h *Handler) CloseOrder(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.CloseOrder")
	defer finish()

	log := h.log()
	ctx := r.Context()

	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		http.Error(w, "Missing order ID", http.StatusBadRequest)
		return
	}

	// Check for flags
	force := r.URL.Query().Get("force") == "true"
	takeaway := r.URL.Query().Get("takeaway") == "true"

	orderServiceURL, _ := h.config.GetString("services.order.url")
	if orderServiceURL == "" {
		log.Error("Order service URL not configured")
		http.Error(w, "Order service not configured", http.StatusServiceUnavailable)
		return
	}

	client := aqm.NewServiceClient(orderServiceURL)
	path := fmt.Sprintf("/orders/%s/close", orderID)

	// Build query params
	params := []string{}
	if force {
		params = append(params, "force=true")
	}
	if takeaway {
		params = append(params, "takeaway=true")
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	resp, err := client.Request(ctx, "POST", path, nil)
	if err != nil {
		errStr := err.Error()
		log.Errorf("Failed to close order: %v", errStr)

		// Try to extract the error message from the backend response
		if strings.Contains(errStr, `"message":"`) {
			start := strings.Index(errStr, `"message":"`) + len(`"message":"`)
			end := strings.Index(errStr[start:], `"`)
			if end > 0 {
				errorMsg := errStr[start : start+end]
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": errorMsg})
				return
			}
		}

		http.Error(w, "Failed to close order", http.StatusInternalServerError)
		return
	}

	// Update table status based on response
	if data, ok := resp.Data.(map[string]interface{}); ok {
		if tableID, ok := data["table_id"].(string); ok && tableID != "" {
			h.updateTableStatus(ctx, log, tableID, data)
		}
	}

	// Return the response from order service
	w.Header().Set("Content-Type", "application/json")
	aqm.RespondSuccess(w, resp.Data)
}

// updateTableStatus updates the table status after order close
func (h *Handler) updateTableStatus(ctx context.Context, log aqm.Logger, tableID string, data map[string]interface{}) {
	tableServiceURL, _ := h.config.GetString("services.table.url")
	if tableServiceURL == "" {
		log.Info("Table service URL not configured, skipping table update")
		return
	}

	tableClient := aqm.NewServiceClient(tableServiceURL)

	// Check if has_takeaway is true -> set to clearing, otherwise close
	hasTakeaway, _ := data["has_takeaway"].(bool)

	var path string
	if hasTakeaway {
		path = fmt.Sprintf("/tables/%s/clearing", tableID)
		log.Info("Setting table to clearing (has takeaway items)", "table_id", tableID)
	} else {
		path = fmt.Sprintf("/tables/%s/close", tableID)
		log.Info("Closing table (no takeaway items)", "table_id", tableID)
	}

	_, err := tableClient.Request(ctx, "POST", path, nil)
	if err != nil {
		log.Errorf("Failed to update table status: %v", err)
	}
}
