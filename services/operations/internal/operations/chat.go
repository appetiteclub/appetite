package operations

import (
	"encoding/json"
	"net/http"
)

type ChatMessageRequest struct {
	Message string `json:"message"`
}

type ChatMessageResponse struct {
	HTML    string `json:"html"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// HandleChatMessage processes incoming chat messages and returns command responses
func (h *Handler) HandleChatMessage(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.HandleChatMessage")
	defer finish()

	// Parse request
	if err := r.ParseForm(); err != nil {
		h.log().Debug("failed to parse form", "error", err)
		http.Error(w, "Failed to parse request", http.StatusBadRequest)
		return
	}

	message := r.FormValue("message")
	if message == "" {
		h.log().Debug("empty message received")
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Process command through command processor
	response, err := h.commandProcessor.Process(r.Context(), message)
	if err != nil {
		h.log().Error("failed to process command", "error", err, "message", message)
		http.Error(w, "Failed to process command", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatMessageResponse{
		HTML:    response.HTML,
		Success: response.Success,
		Message: response.Message,
	})
}
