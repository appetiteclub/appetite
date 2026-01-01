package authz

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/appetiteclub/apt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/appetiteclub/apt/telemetry"
)

// RoleHandler handles role-related HTTP requests
type RoleHandler struct {
	roleRepo RoleRepo
	logger   apt.Logger
	config   *apt.Config
	tlm      *telemetry.HTTP
}

// NewRoleHandler creates a new RoleHandler
func NewRoleHandler(roleRepo RoleRepo, config *apt.Config, logger apt.Logger) *RoleHandler {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}
	return &RoleHandler{
		roleRepo: roleRepo,
		logger:   logger,
		config:   config,
		tlm:      telemetry.NewHTTP(),
	}
}

// RegisterRoutes registers role routes
func (h *RoleHandler) RegisterRoutes(r chi.Router) {
	r.Route("/authz/roles", func(r chi.Router) {
		r.Get("/", h.ListRoles)
		r.Post("/", h.CreateRole)
		r.Get("/{id}", h.GetRole)
		r.Put("/{id}", h.UpdateRole)
		r.Delete("/{id}", h.DeleteRole)
	})
}

// RoleRequest represents the request payload for creating/updating roles
type RoleRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// ListRoles handles GET /authz/roles
func (h *RoleHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "RoleHandler.ListRoles")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	// Parse query parameters
	status := r.URL.Query().Get("status")
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	var roles []*Role
	var err error

	if status != "" {
		roles, err = h.roleRepo.ListByStatus(ctx, status)
	} else {
		roles, err = h.roleRepo.List(ctx)
	}

	if err != nil {
		log.Error("failed to list roles", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to retrieve roles")
		return
	}

	// Apply pagination if specified
	if page != "" && limit != "" {
		pageNum, _ := strconv.Atoi(page)
		limitNum, _ := strconv.Atoi(limit)
		roles = h.paginateRoles(roles, pageNum, limitNum)
	}

	// Generate HATEOAS links
	links := []apt.Link{
		{Rel: "self", Href: "/authz/roles"},
		{Rel: "create", Href: "/authz/roles"},
	}

	response := apt.SuccessResponse{
		Data:  roles,
		Links: links,
	}

	apt.RespondSuccess(w, response.Data)
}

// CreateRole handles POST /authz/roles
func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "RoleHandler.CreateRole")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	var req RoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Debug("invalid request payload", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate request
	if req.Name == "" {
		apt.RespondError(w, http.StatusBadRequest, "Role name is required")
		return
	}

	// Check if role already exists
	existing, err := h.roleRepo.GetByName(ctx, req.Name)
	if err != nil {
		log.Error("error checking existing role", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to create role")
		return
	}
	if existing != nil {
		apt.RespondError(w, http.StatusConflict, "Role already exists")
		return
	}

	// Create new role
	role := NewRole()
	role.Name = req.Name
	role.Permissions = req.Permissions

	if err := h.roleRepo.Create(ctx, role); err != nil {
		log.Error("failed to create role", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to create role")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(apt.SuccessResponse{Data: role}); err != nil {
		log.Error("failed to encode response", "error", err)
	}
}

// GetRole handles GET /authz/roles/{id}
func (h *RoleHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "RoleHandler.GetRole")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	role, err := h.roleRepo.Get(ctx, id)
	if err != nil {
		log.Error("failed to get role", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to retrieve role")
		return
	}

	if role == nil {
		apt.RespondError(w, http.StatusNotFound, "Role not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(apt.SuccessResponse{Data: role}); err != nil {
		log.Error("failed to encode response", "error", err)
	}
}

// UpdateRole handles PUT /authz/roles/{id}
func (h *RoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "RoleHandler.UpdateRole")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	var req RoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Get existing role
	role, err := h.roleRepo.Get(ctx, id)
	if err != nil {
		log.Error("failed to get role", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to update role")
		return
	}

	if role == nil {
		apt.RespondError(w, http.StatusNotFound, "Role not found")
		return
	}

	// Update role fields
	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Permissions != nil {
		role.Permissions = req.Permissions
	}

	if err := h.roleRepo.Save(ctx, role); err != nil {
		log.Error("failed to save role", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to update role")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(apt.SuccessResponse{Data: role}); err != nil {
		log.Error("failed to encode response", "error", err)
	}
}

// DeleteRole handles DELETE /authz/roles/{id}
func (h *RoleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "RoleHandler.DeleteRole")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	if err := h.roleRepo.Delete(ctx, id); err != nil {
		log.Error("failed to delete role", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to delete role")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *RoleHandler) paginateRoles(roles []*Role, page, limit int) []*Role {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	start := (page - 1) * limit
	end := start + limit

	if start >= len(roles) {
		return []*Role{}
	}
	if end > len(roles) {
		end = len(roles)
	}

	return roles[start:end]
}

func (h *RoleHandler) log(req ...*http.Request) apt.Logger {
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
