package authz

import (
	"encoding/json"
	"net/http"

	"github.com/aquamarinepk/aqm"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/aquamarinepk/aqm/telemetry"
)

// PolicyHandler handles policy evaluation HTTP requests
type PolicyHandler struct {
	policyEngine *PolicyEngine
	logger       aqm.Logger
	config       *aqm.Config
	tlm          *telemetry.HTTP
}

// NewPolicyHandler creates a new PolicyHandler
func NewPolicyHandler(policyEngine *PolicyEngine, config *aqm.Config, logger aqm.Logger) *PolicyHandler {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &PolicyHandler{
		policyEngine: policyEngine,
		logger:       logger,
		config:       config,
		tlm:          telemetry.NewHTTP(),
	}
}

// RegisterRoutes registers policy evaluation routes
func (h *PolicyHandler) RegisterRoutes(r chi.Router) {
	r.Route("/authz/policy", func(r chi.Router) {
		r.Post("/evaluate", h.EvaluatePermission)
		r.Get("/users/{user_id}/permissions", h.GetUserPermissions)
	})
}

// PermissionRequest represents the request payload for permission evaluation
type PermissionRequest struct {
	UserID     string `json:"user_id"`
	Permission string `json:"permission"`
	Scope      Scope  `json:"scope"`
}

// PermissionResponse represents the response for permission evaluation
type PermissionResponse struct {
	UserID     string `json:"user_id"`
	Permission string `json:"permission"`
	Scope      Scope  `json:"scope"`
	Allowed    bool   `json:"allowed"`
}

// UserPermissionsResponse represents the response for user permissions
type UserPermissionsResponse struct {
	UserID      string   `json:"user_id"`
	Scope       Scope    `json:"scope"`
	Permissions []string `json:"permissions"`
}

// EvaluatePermission handles POST /authz/policy/evaluate
// This is the core endpoint that other services will call to check permissions
func (h *PolicyHandler) EvaluatePermission(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "PolicyHandler.EvaluatePermission")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	var req PermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Debug("invalid request payload", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate request
	if req.UserID == "" {
		aqm.RespondError(w, http.StatusBadRequest, "User ID is required")
		return
	}
	if req.Permission == "" {
		aqm.RespondError(w, http.StatusBadRequest, "Permission is required")
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	// Set default scope if not provided
	scope := req.Scope
	if req.Scope.Type == "" {
		scope = Scope{Type: "global", ID: ""}
	}

	// Evaluate permission using policy engine
	allowed, err := h.policyEngine.Has(ctx, userID, req.Permission, scope)
	if err != nil {
		log.Error("failed to evaluate permission", "error", err,
			"user_id", req.UserID,
			"permission", req.Permission,
			"scope", scope)
		aqm.RespondError(w, http.StatusInternalServerError, "Failed to evaluate permission")
		return
	}

	log.Info("permission evaluated",
		"user_id", req.UserID,
		"permission", req.Permission,
		"scope", scope,
		"allowed", allowed)

	response := PermissionResponse{
		UserID:     req.UserID,
		Permission: req.Permission,
		Scope:      scope,
		Allowed:    allowed,
	}

	// Return 200 OK regardless of permission result
	// The client should check the "allowed" field
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(aqm.SuccessResponse{Data: response})
}

// GetUserPermissions handles GET /authz/policy/users/{user_id}/permissions
// Returns all permissions for a user in a given scope
func (h *PolicyHandler) GetUserPermissions(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "PolicyHandler.GetUserPermissions")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Parse scope from query parameters
	scopeType := r.URL.Query().Get("scope_type")
	scopeID := r.URL.Query().Get("scope_id")

	// Set default scope if not provided
	scope := Scope{Type: "global", ID: ""}
	if scopeType != "" {
		scope.Type = scopeType
		scope.ID = scopeID
	}

	// Get user permissions from policy engine
	permissions, err := h.policyEngine.GetUserPermissions(ctx, userID, scope)
	if err != nil {
		log.Error("failed to get user permissions", "error", err,
			"user_id", userIDStr,
			"scope", scope)
		aqm.RespondError(w, http.StatusInternalServerError, "Failed to retrieve user permissions")
		return
	}

	response := UserPermissionsResponse{
		UserID:      userIDStr,
		Scope:       scope,
		Permissions: permissions,
	}

	// Generate HATEOAS links
	links := []aqm.Link{
		{Rel: "self", Href: "/authz/policy/users/" + userIDStr + "/permissions"},
		{Rel: "user", Href: "/users/" + userIDStr},
		{Rel: "grants", Href: "/authz/grants/users/" + userIDStr},
		{Rel: "evaluate", Href: "/authz/policy/evaluate"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(aqm.SuccessResponse{Data: response, Links: links})
}

// Helper methods

func (h *PolicyHandler) log(req ...*http.Request) aqm.Logger {
	logger := h.logger
	if len(req) > 0 && req[0] != nil {
		r := req[0]
		return logger.With(
			"request_id", aqm.RequestIDFrom(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
		)
	}
	return logger
}
