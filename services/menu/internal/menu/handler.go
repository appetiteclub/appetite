package menu

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/appetiteclub/appetite/services/menu/internal/dictionary"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const MaxBodyBytes = 2 << 20 // 2 MB (larger for menu items with images)

// Handler handles HTTP requests for the Menu service
type Handler struct {
	itemRepo   MenuItemRepo
	menuRepo   MenuRepo
	dictClient dictionary.Client
	logger     aqm.Logger
	config     *aqm.Config
	tlm        *telemetry.HTTP
}

// NewHandler creates a new Handler for Menu operations
func NewHandler(itemRepo MenuItemRepo, menuRepo MenuRepo, dictClient dictionary.Client, config *aqm.Config, logger aqm.Logger) *Handler {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	if dictClient == nil {
		dictClient = dictionary.NewNoopClient()
	}
	return &Handler{
		itemRepo:   itemRepo,
		menuRepo:   menuRepo,
		dictClient: dictClient,
		logger:     logger,
		config:     config,
		tlm:        telemetry.NewHTTP(),
	}
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
		aqm.RespondError(w, http.StatusInternalServerError, "Could not create menu item")
		return
	}

	links := aqm.RESTfulLinksFor(item)
	w.WriteHeader(http.StatusCreated)
	aqm.RespondSuccess(w, item, links...)
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
		aqm.RespondError(w, http.StatusNotFound, "Menu item not found")
		return
	}

	if item == nil {
		aqm.RespondError(w, http.StatusNotFound, "Menu item not found")
		return
	}

	links := aqm.RESTfulLinksFor(item)
	aqm.RespondSuccess(w, item, links...)
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
		aqm.RespondError(w, http.StatusBadRequest, "Missing shortCode parameter")
		return
	}

	item, err := h.itemRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		log.Error("error loading menu item by code", "error", err, "code", shortCode)
		aqm.RespondError(w, http.StatusNotFound, "Menu item not found")
		return
	}

	if item == nil {
		aqm.RespondError(w, http.StatusNotFound, "Menu item not found")
		return
	}

	links := aqm.RESTfulLinksFor(item)
	aqm.RespondSuccess(w, item, links...)
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
		aqm.RespondError(w, http.StatusInternalServerError, "Could not list menu items")
		return
	}

	aqm.RespondCollection(w, items, "menu/items")
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
		aqm.RespondError(w, http.StatusBadRequest, "Missing categoryID parameter")
		return
	}

	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		log.Debug("invalid categoryID parameter", "categoryID", categoryIDStr, "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid categoryID parameter")
		return
	}

	items, err := h.itemRepo.ListByCategory(ctx, categoryID)
	if err != nil {
		log.Error("cannot list menu items by category", "error", err, "categoryID", categoryID)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not list menu items by category")
		return
	}

	aqm.RespondCollection(w, items, "menu/items")
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
		aqm.RespondError(w, http.StatusInternalServerError, "Could not update menu item")
		return
	}

	links := aqm.RESTfulLinksFor(item)
	aqm.RespondSuccess(w, item, links...)
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
		aqm.RespondError(w, http.StatusInternalServerError, "Could not delete menu item")
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
		aqm.RespondError(w, http.StatusInternalServerError, "Could not create menu")
		return
	}

	links := aqm.RESTfulLinksFor(menu)
	w.WriteHeader(http.StatusCreated)
	aqm.RespondSuccess(w, menu, links...)
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
		aqm.RespondError(w, http.StatusNotFound, "Menu not found")
		return
	}

	if menu == nil {
		aqm.RespondError(w, http.StatusNotFound, "Menu not found")
		return
	}

	links := aqm.RESTfulLinksFor(menu)
	aqm.RespondSuccess(w, menu, links...)
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
		aqm.RespondError(w, http.StatusInternalServerError, "Could not list menus")
		return
	}

	aqm.RespondCollection(w, menus, "menu/menus")
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
		aqm.RespondError(w, http.StatusInternalServerError, "Could not update menu")
		return
	}

	links := aqm.RESTfulLinksFor(menu)
	aqm.RespondSuccess(w, menu, links...)
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
		aqm.RespondError(w, http.StatusInternalServerError, "Could not delete menu")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *Handler) log(r *http.Request) aqm.Logger {
	return h.logger.With("request_id", r.Context().Value("request_id"))
}

func (h *Handler) parseIDParam(w http.ResponseWriter, r *http.Request, log aqm.Logger) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		log.Debug("missing id parameter")
		aqm.RespondError(w, http.StatusBadRequest, "Missing id parameter")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Debug("invalid id parameter", "id", idStr, "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid id parameter")
		return uuid.Nil, false
	}

	return id, true
}

func (h *Handler) decodeMenuItemPayload(w http.ResponseWriter, r *http.Request, log aqm.Logger) (*MenuItem, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("error reading request body", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return nil, false
	}

	var item MenuItem
	if err := json.Unmarshal(body, &item); err != nil {
		log.Debug("error decoding JSON", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return nil, false
	}

	return &item, true
}

func (h *Handler) decodeMenuPayload(w http.ResponseWriter, r *http.Request, log aqm.Logger) (*Menu, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("error reading request body", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return nil, false
	}

	var menu Menu
	if err := json.Unmarshal(body, &menu); err != nil {
		log.Debug("error decoding JSON", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return nil, false
	}

	return &menu, true
}

func (h *Handler) respondValidationErrors(w http.ResponseWriter, errors []ValidationError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":  "Validation failed",
		"errors": errors,
	})
}
