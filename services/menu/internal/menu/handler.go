package menu

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/appetiteclub/appetite/services/menu/internal/dictionary"
	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const MaxBodyBytes = 2 << 20 // 2 MB (larger for menu items with images)

// Handler handles HTTP requests for the Menu service
type Handler struct {
	config     *apt.Config
	logger     apt.Logger
	tlm        *telemetry.HTTP
	itemRepo   MenuItemRepo
	menuRepo   MenuRepo
	dictClient dictionary.Client
}

type HandlerDeps struct {
	ItemRepo   MenuItemRepo
	MenuRepo   MenuRepo
	DictClient dictionary.Client
}

// NewHandler creates a new Handler for Menu operations
// Fails fast and returns an error if the dictionary client is not provided (no Noop fallback).
func NewHandler(hd HandlerDeps, config *apt.Config, logger apt.Logger) (*Handler, error) {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}

	if hd.DictClient == nil {
		return nil, fmt.Errorf("dictionary service unavailable")
	}

	return &Handler{
		config:     config,
		logger:     logger,
		tlm:        telemetry.NewHTTP(),
		itemRepo:   hd.ItemRepo,
		menuRepo:   hd.MenuRepo,
		dictClient: hd.DictClient,
	}, nil
}

// RegisterRoutes registers all routes for the menu service
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/menu", func(r chi.Router) {
		// Menu Item routes
		r.Route("/items", func(r chi.Router) {
			r.Post("/", h.CreateMenuItem)
			r.Get("/", h.ListMenuItems)
			r.Get("/{id}", h.GetMenuItem)
			r.Put("/{id}", h.UpdateMenuItem)
			r.Delete("/{id}", h.DeleteMenuItem)
			r.Get("/code/{shortCode}", h.GetMenuItemByCode)
			r.Get("/category/{categoryID}", h.ListMenuItemsByCategory)
		})

		// Menu routes
		r.Route("/menus", func(r chi.Router) {
			r.Post("/", h.CreateMenu)
			r.Get("/", h.ListMenus)
			r.Get("/{id}", h.GetMenu)
			r.Put("/{id}", h.UpdateMenu)
			r.Delete("/{id}", h.DeleteMenu)
		})
	})
}

// MenuItem Handlers

// CreateMenuItem handles POST /menu/items
func (h *Handler) CreateMenuItem(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CreateMenuItem")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	item, ok := h.decodeMenuItemPayload(w, r, log)
	if !ok {
		return
	}

	item.EnsureID()
	item.BeforeCreate()

	// Validation
	if validationErrors := ValidateCreateMenuItem(ctx, item, h.dictClient); len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors)
		h.respondValidationErrors(w, validationErrors)
		return
	}

	// Create in repository
	if err := h.itemRepo.Create(ctx, item); err != nil {
		log.Error("cannot create menu item", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not create menu item")
		return
	}

	links := apt.RESTfulLinksFor(item)
	w.WriteHeader(http.StatusCreated)
	apt.RespondSuccess(w, item, links...)
}

// GetMenuItem handles GET /menu/items/{id}
func (h *Handler) GetMenuItem(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.GetMenuItem")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	item, err := h.itemRepo.Get(ctx, id)
	if err != nil {
		log.Error("error loading menu item", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusNotFound, "Menu item not found")
		return
	}

	if item == nil {
		apt.RespondError(w, http.StatusNotFound, "Menu item not found")
		return
	}

	links := apt.RESTfulLinksFor(item)
	apt.RespondSuccess(w, item, links...)
}

// GetMenuItemByCode handles GET /menu/items/code/{shortCode}
func (h *Handler) GetMenuItemByCode(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.GetMenuItemByCode")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	shortCode := chi.URLParam(r, "shortCode")
	if shortCode == "" {
		log.Debug("missing shortCode parameter")
		apt.RespondError(w, http.StatusBadRequest, "Missing shortCode parameter")
		return
	}

	item, err := h.itemRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		log.Error("error loading menu item by code", "error", err, "code", shortCode)
		apt.RespondError(w, http.StatusNotFound, "Menu item not found")
		return
	}

	if item == nil {
		apt.RespondError(w, http.StatusNotFound, "Menu item not found")
		return
	}

	links := apt.RESTfulLinksFor(item)
	apt.RespondSuccess(w, item, links...)
}

// ListMenuItems handles GET /menu/items
func (h *Handler) ListMenuItems(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ListMenuItems")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	// Check for active query parameter
	activeOnly := r.URL.Query().Get("active") == "true"

	var items []*MenuItem
	var err error

	if activeOnly {
		items, err = h.itemRepo.ListActive(ctx)
	} else {
		items, err = h.itemRepo.List(ctx)
	}

	if err != nil {
		log.Error("cannot list menu items", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not list menu items")
		return
	}

	apt.RespondCollection(w, items, "menu/items")
}

// ListMenuItemsByCategory handles GET /menu/items/category/{categoryID}
func (h *Handler) ListMenuItemsByCategory(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ListMenuItemsByCategory")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	categoryIDStr := chi.URLParam(r, "categoryID")
	if categoryIDStr == "" {
		log.Debug("missing categoryID parameter")
		apt.RespondError(w, http.StatusBadRequest, "Missing categoryID parameter")
		return
	}

	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		log.Debug("invalid categoryID parameter", "categoryID", categoryIDStr, "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid categoryID parameter")
		return
	}

	items, err := h.itemRepo.ListByCategory(ctx, categoryID)
	if err != nil {
		log.Error("cannot list menu items by category", "error", err, "categoryID", categoryID)
		apt.RespondError(w, http.StatusInternalServerError, "Could not list menu items by category")
		return
	}

	apt.RespondCollection(w, items, "menu/items")
}

// UpdateMenuItem handles PUT /menu/items/{id}
func (h *Handler) UpdateMenuItem(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.UpdateMenuItem")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	item, ok := h.decodeMenuItemPayload(w, r, log)
	if !ok {
		return
	}

	item.ID = id
	item.BeforeUpdate()

	// Validation
	if validationErrors := ValidateUpdateMenuItem(ctx, item, h.dictClient); len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors)
		h.respondValidationErrors(w, validationErrors)
		return
	}

	// Update in repository
	if err := h.itemRepo.Save(ctx, item); err != nil {
		log.Error("cannot update menu item", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not update menu item")
		return
	}

	links := apt.RESTfulLinksFor(item)
	apt.RespondSuccess(w, item, links...)
}

// DeleteMenuItem handles DELETE /menu/items/{id}
func (h *Handler) DeleteMenuItem(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.DeleteMenuItem")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	if err := h.itemRepo.Delete(ctx, id); err != nil {
		log.Error("cannot delete menu item", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not delete menu item")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Menu Handlers

// CreateMenu handles POST /menu/menus
func (h *Handler) CreateMenu(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CreateMenu")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	menu, ok := h.decodeMenuPayload(w, r, log)
	if !ok {
		return
	}

	menu.EnsureID()
	menu.BeforeCreate()

	// Validation
	if validationErrors := ValidateCreateMenu(ctx, menu, h.dictClient); len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors)
		h.respondValidationErrors(w, validationErrors)
		return
	}

	// Create in repository
	if err := h.menuRepo.Create(ctx, menu); err != nil {
		log.Error("cannot create menu", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not create menu")
		return
	}

	links := apt.RESTfulLinksFor(menu)
	w.WriteHeader(http.StatusCreated)
	apt.RespondSuccess(w, menu, links...)
}

// GetMenu handles GET /menu/menus/{id}
func (h *Handler) GetMenu(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.GetMenu")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	menu, err := h.menuRepo.Get(ctx, id)
	if err != nil {
		log.Error("error loading menu", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusNotFound, "Menu not found")
		return
	}

	if menu == nil {
		apt.RespondError(w, http.StatusNotFound, "Menu not found")
		return
	}

	links := apt.RESTfulLinksFor(menu)
	apt.RespondSuccess(w, menu, links...)
}

// ListMenus handles GET /menu/menus
func (h *Handler) ListMenus(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ListMenus")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	// Check for published query parameter
	publishedOnly := r.URL.Query().Get("published") == "true"

	var menus []*Menu
	var err error

	if publishedOnly {
		menus, err = h.menuRepo.ListPublished(ctx)
	} else {
		menus, err = h.menuRepo.List(ctx)
	}

	if err != nil {
		log.Error("cannot list menus", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not list menus")
		return
	}

	apt.RespondCollection(w, menus, "menu/menus")
}

// UpdateMenu handles PUT /menu/menus/{id}
func (h *Handler) UpdateMenu(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.UpdateMenu")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	menu, ok := h.decodeMenuPayload(w, r, log)
	if !ok {
		return
	}

	menu.ID = id
	menu.BeforeUpdate()

	// Validation
	if validationErrors := ValidateUpdateMenu(ctx, menu, h.dictClient); len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors)
		h.respondValidationErrors(w, validationErrors)
		return
	}

	// Update in repository
	if err := h.menuRepo.Save(ctx, menu); err != nil {
		log.Error("cannot update menu", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not update menu")
		return
	}

	links := apt.RESTfulLinksFor(menu)
	apt.RespondSuccess(w, menu, links...)
}

// DeleteMenu handles DELETE /menu/menus/{id}
func (h *Handler) DeleteMenu(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.DeleteMenu")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	if err := h.menuRepo.Delete(ctx, id); err != nil {
		log.Error("cannot delete menu", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not delete menu")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *Handler) log(r *http.Request) apt.Logger {
	return h.logger.With("request_id", r.Context().Value("request_id"))
}

func (h *Handler) parseIDParam(w http.ResponseWriter, r *http.Request, log apt.Logger) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		log.Debug("missing id parameter")
		apt.RespondError(w, http.StatusBadRequest, "Missing id parameter")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Debug("invalid id parameter", "id", idStr, "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid id parameter")
		return uuid.Nil, false
	}

	return id, true
}

func (h *Handler) decodeMenuItemPayload(w http.ResponseWriter, r *http.Request, log apt.Logger) (*MenuItem, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer func() { _ = r.Body.Close() }()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("error reading request body", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return nil, false
	}

	var item MenuItem
	if err := json.Unmarshal(body, &item); err != nil {
		log.Debug("error decoding JSON", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return nil, false
	}

	return &item, true
}

func (h *Handler) decodeMenuPayload(w http.ResponseWriter, r *http.Request, log apt.Logger) (*Menu, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer func() { _ = r.Body.Close() }()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("error reading request body", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return nil, false
	}

	var menu Menu
	if err := json.Unmarshal(body, &menu); err != nil {
		log.Debug("error decoding JSON", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return nil, false
	}

	return &menu, true
}

func (h *Handler) respondValidationErrors(w http.ResponseWriter, errors []ValidationError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"error":  "Validation failed",
		"errors": errors,
	}); err != nil {
		h.logger.Debug("failed to encode validation errors", "error", err)
	}
}
