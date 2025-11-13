package operations

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type ChatMessageRequest struct {
	Message string `json:"message"`
	Token   string `json:"token"`
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

	token := r.FormValue("token")

	// Add token to context for commands that need it
	ctx := r.Context()
	if token != "" {
		ctx = context.WithValue(ctx, contextKeyToken, token)

		// Validate token and add userID to context
		userID, err := h.tokenStore.Validate(token)
		if err == nil {
			ctx = context.WithValue(ctx, contextKeyUserID, userID)
		}
	}

	// Check if command requires authentication
	requiresAuth := h.commandRequiresAuth(message)
	if requiresAuth {
		// Check if user is authenticated
		userID := getUserIDFromContext(ctx)
		if userID == uuid.Nil {
			// Not authenticated, return error
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ChatMessageResponse{
				HTML: `
					<div style="padding: 1rem; background: #fef2f2; border-radius: 0.5rem; border-left: 4px solid #ef4444;">
						<p style="margin: 0;"><strong>🔒 Authentication Required</strong></p>
						<p style="margin: 0.5rem 0 0 0; font-size: 0.9em;">Please log in first.</p>
						<p style="margin: 0.5rem 0 0 0; font-size: 0.85em; color: #666;"><em>Use 'login [pin]' or '.[pin]' to authenticate.</em></p>
					</div>
				`,
				Success: false,
				Message: "Authentication required",
			})
			return
		}
	}

	// Process command through command processor
	response, err := h.commandProcessor.Process(ctx, message)
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

func (h *Handler) commandRequiresAuth(message string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(message))

	// Authentication commands don't require auth
	if strings.HasPrefix(trimmed, "login") || strings.HasPrefix(trimmed, ".") || trimmed == "exit" || trimmed == "help" {
		return false
	}

	// All other commands require authentication
	return true
}
