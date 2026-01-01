package authz

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/appetiteclub/apt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/appetiteclub/apt/telemetry"
)

// GrantHandler handles grant-related HTTP requests
type GrantHandler struct {
	grantRepo GrantRepo
	roleRepo  RoleRepo
	logger    apt.Logger
	config    *apt.Config
	tlm       *telemetry.HTTP
}

// NewGrantHandler creates a new GrantHandler
func NewGrantHandler(grantRepo GrantRepo, roleRepo RoleRepo, config *apt.Config, logger apt.Logger) *GrantHandler {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}
	return &GrantHandler{
		grantRepo: grantRepo,
		roleRepo:  roleRepo,
		logger:    logger,
		config:    config,
		tlm:       telemetry.NewHTTP(),
	}
}

// RegisterRoutes registers grant routes
func (h *GrantHandler) RegisterRoutes(r chi.Router) {
	r.Route("/authz/grants", func(r chi.Router) {
		r.Get("/", h.ListGrants)
		r.Post("/", h.CreateGrant)
		r.Get("/{id}", h.GetGrant)
		r.Delete("/{id}", h.RevokeGrant)

		// User-specific grants
		r.Get("/users/{user_id}", h.ListUserGrants)

		// Expired grants cleanup
		r.Get("/expired", h.ListExpiredGrants)
	})
}

// GrantRequest represents the request payload for creating grants
type GrantRequest struct {
	UserID    string  `json:"user_id"`
	RoleName  string  `json:"role_name"`
	Resource  string  `json:"resource"`
	ExpiresAt *string `json:"expires_at,omitempty"` // ISO8601 timestamp
}

// ListGrants handles GET /authz/grants
func (h *GrantHandler) ListGrants(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "GrantHandler.ListGrants")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	// Parse query parameters
	userID := r.URL.Query().Get("user_id")

	var grants []*Grant
	var err error

	if userID != "" {
		uid, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			apt.RespondError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}
		grants, err = h.grantRepo.ListByUserID(ctx, uid)
	} else {
		grants, err = h.grantRepo.List(ctx)
	}

	if err != nil {
		log.Error("failed to list grants", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to retrieve grants")
		return
	}

	// Generate HATEOAS links
	links := []apt.Link{
		{Rel: "self", Href: "/authz/grants"},
		{Rel: "create", Href: "/authz/grants"},
		{Rel: "expired", Href: "/authz/grants/expired"},
	}

	apt.RespondSuccess(w, grants, links...)
}

// CreateGrant handles POST /authz/grants
func (h *GrantHandler) CreateGrant(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "GrantHandler.CreateGrant")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	var req GrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Debug("invalid request payload", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate request
	if req.UserID == "" {
		apt.RespondError(w, http.StatusBadRequest, "User ID is required")
		return
	}
	if req.RoleName == "" {
		apt.RespondError(w, http.StatusBadRequest, "Role name is required")
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	// Since we have a role_name, this is always a role grant
	grantType := GrantTypeRole

	// Validate that the role exists by name
	role, err := h.roleRepo.GetByName(ctx, req.RoleName)
	if err != nil {
		log.Error("failed to get role by name", "error", err, "role_name", req.RoleName)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to validate role")
		return
	}
	if role == nil {
		apt.RespondError(w, http.StatusBadRequest, "Role '"+req.RoleName+"' does not exist")
		return
	}

	// Parse expiration if provided
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		parsed, parseErr := time.Parse(time.RFC3339, *req.ExpiresAt)
		if parseErr != nil {
			apt.RespondError(w, http.StatusBadRequest, "Invalid expiration date format. Use ISO8601/RFC3339")
			return
		}
		expiresAt = &parsed
	}

	// Create new grant
	grant := NewGrant()
	grant.UserID = userID
	grant.GrantType = grantType
	grant.Value = role.ID.String() // Store role UUID, not name
	grant.Scope = Scope{Type: "resource", ID: req.Resource}
	grant.ExpiresAt = expiresAt

	if err := h.grantRepo.Create(ctx, grant); err != nil {
		log.Error("failed to create grant", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to create grant")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(apt.SuccessResponse{Data: grant})
}

// GetGrant handles GET /authz/grants/{id}
func (h *GrantHandler) GetGrant(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "GrantHandler.GetGrant")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid grant ID")
		return
	}

	grant, err := h.grantRepo.Get(ctx, id)
	if err != nil {
		log.Error("failed to get grant", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to retrieve grant")
		return
	}

	if grant == nil {
		apt.RespondError(w, http.StatusNotFound, "Grant not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apt.SuccessResponse{Data: grant})
}

// RevokeGrant handles DELETE /authz/grants/{id}
func (h *GrantHandler) RevokeGrant(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "GrantHandler.RevokeGrant")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid grant ID")
		return
	}

	if err := h.grantRepo.Delete(ctx, id); err != nil {
		log.Error("failed to revoke grant", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to revoke grant")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListUserGrants handles GET /authz/grants/users/{user_id}
func (h *GrantHandler) ListUserGrants(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "GrantHandler.ListUserGrants")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	grants, err := h.grantRepo.ListByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to list user grants", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to retrieve user grants")
		return
	}

	// Generate HATEOAS links
	links := []apt.Link{
		{Rel: "self", Href: "/authz/grants/users/" + userIDStr},
		{Rel: "user", Href: "/users/" + userIDStr},
		{Rel: "create", Href: "/authz/grants"},
	}

	response := apt.SuccessResponse{
		Data:  grants,
		Links: links,
	}

	apt.RespondSuccess(w, response.Data)
}

// ListExpiredGrants handles GET /authz/grants/expired
func (h *GrantHandler) ListExpiredGrants(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "GrantHandler.ListExpiredGrants")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	grants, err := h.grantRepo.ListExpired(ctx)
	if err != nil {
		log.Error("failed to list expired grants", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to retrieve expired grants")
		return
	}

	// Generate HATEOAS links
	links := []apt.Link{
		{Rel: "self", Href: "/authz/grants/expired"},
		{Rel: "all", Href: "/authz/grants"},
	}

	response := apt.SuccessResponse{
		Data:  grants,
		Links: links,
	}

	apt.RespondSuccess(w, response.Data)
}

// Helper methods

func (h *GrantHandler) log(req ...*http.Request) apt.Logger {
	logger := h.logger
	if len(req) > 0 && req[0] != nil {
		r := req[0]
		return logger.With(
			"request_id", apt.RequestIDFrom(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
		)
	}
	return logger
}
