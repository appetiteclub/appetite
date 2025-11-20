package operations

import (
	"fmt"
	"net/http"

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
