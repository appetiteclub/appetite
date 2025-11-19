package operations

import (
	"net/http"
	"time"

	"github.com/aquamarinepk/aqm"
	authpkg "github.com/aquamarinepk/aqm/auth"
	"github.com/aquamarinepk/aqm/telemetry"
	aqmtemplate "github.com/aquamarinepk/aqm/template"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	tmplMgr          *aqmtemplate.Manager
	authnClient      *aqm.ServiceClient
	tableClient      *aqm.ServiceClient
	orderClient      *aqm.ServiceClient
	menuClient       *aqm.ServiceClient
	tableData        *TableDataAccess
	orderData        *OrderDataAccess
	kitchenData      *KitchenDataAccess
	roleRepo         RoleRepo
	grantRepo        GrantRepo
	authzHelper      *authpkg.AuthzHelper
	logger           aqm.Logger
	config           *aqm.Config
	http             *telemetry.HTTP
	sessionStore     *SessionStore
	tokenStore       *TokenStore
	auditLogger      *AuditLogger
	commandProcessor CommandProcessor
}

func NewHandler(
	tmplMgr *aqmtemplate.Manager,
	roleRepo RoleRepo,
	grantRepo GrantRepo,
	config *aqm.Config,
	logger aqm.Logger,
) *Handler {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}

	// Initialize service clients
	authnURL, _ := config.GetString("services.authn.url")
	authnClient := aqm.NewServiceClient(authnURL)

	tableURL, _ := config.GetString("services.table.url")
	tableClient := aqm.NewServiceClient(tableURL)

	orderURL, _ := config.GetString("services.order.url")
	orderClient := aqm.NewServiceClient(orderURL)

	menuURL, _ := config.GetString("services.menu.url")
	if menuURL == "" {
		menuURL = "http://localhost:8088"
	}
	menuClient := aqm.NewServiceClient(menuURL)

	kitchenURL, _ := config.GetString("services.kitchen.url")
	var kitchenClient *aqm.ServiceClient
	if kitchenURL != "" {
		kitchenClient = aqm.NewServiceClient(kitchenURL)
	}

	authzHelper := newAuthzHelper(config, logger)

	// Initialize session store
	sessionSecret, _ := config.GetString("auth.session.secret")
	sessionTTLStr, _ := config.GetString("auth.session.ttl")
	sessionTTL, _ := time.ParseDuration(sessionTTLStr)
	if sessionTTL == 0 {
		sessionTTL = 8 * time.Hour
	}
	sessionStore := NewSessionStore([]byte(sessionSecret), sessionTTL)

	// Initialize token store for transient chat authentication
	tokenTTL := 30 * time.Minute
	tokenStore := NewTokenStore(tokenTTL)

	// Initialize audit logger
	auditLogger := NewAuditLogger(logger)

	handler := &Handler{
		tmplMgr:      tmplMgr,
		authnClient:  authnClient,
		tableClient:  tableClient,
		orderClient:  orderClient,
		menuClient:   menuClient,
		tableData:    NewTableDataAccess(tableClient),
		orderData:    NewOrderDataAccess(orderClient),
		kitchenData:  NewKitchenDataAccess(kitchenClient),
		roleRepo:     roleRepo,
		grantRepo:    grantRepo,
		authzHelper:  authzHelper,
		logger:       logger,
		config:       config,
		http:         telemetry.NewHTTP(),
		sessionStore: sessionStore,
		tokenStore:   tokenStore,
		auditLogger:  auditLogger,
	}

	// Initialize command processor with handler reference for auth commands
	commandProcessor := NewDeterministicParser(
		tableClient,
		orderClient,
		menuClient,
		handler,
	)

	handler.commandProcessor = commandProcessor

	return handler
}

func newAuthzHelper(config *aqm.Config, logger aqm.Logger) *authpkg.AuthzHelper {
	authzURL, _ := config.GetString("services.authz.url")
	if authzURL == "" {
		if logger != nil {
			logger.Info("services.authz.url not configured; authorization checks will fail")
		}
		return nil
	}

	cacheTTL := 5 * time.Minute
	if ttlStr, ok := config.GetString("authz.cache_ttl"); ok && ttlStr != "" {
		if parsed, err := time.ParseDuration(ttlStr); err == nil {
			cacheTTL = parsed
		} else if logger != nil {
			logger.Info("invalid authz.cache_ttl value", "value", ttlStr, "error", err)
		}
	}

	authzClient := aqm.NewAuthzClient(authzURL)
	return aqm.NewAuthzHelper(authzClient, cacheTTL)
}

// RegisterRoutes registers all operations routes using Command/Query pattern
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Public routes
	r.Get("/signin", h.ShowSignIn)
	r.Post("/signin", h.HandleSignIn)
	r.Post("/signout", h.HandleSignOut)

	// Protected routes (require session)
	r.Group(func(r chi.Router) {
		r.Use(h.SessionMiddleware)

		r.Get("/", h.Home)
		r.Get("/chat", h.Chat)
		r.Post("/chat/message", h.HandleChatMessage)
		r.Get("/list-tables", h.Tables)
		r.Get("/add-table", h.NewTableForm)
		r.Get("/edit-table/{id}", h.EditTableForm)
		r.Post("/add-table", h.CreateTable)
		r.Post("/update-table/{id}", h.UpdateTable)
		r.Post("/delete-table/{id}", h.DeleteTable)
		r.Get("/orders", h.Orders)
		r.Get("/add-order", h.NewOrderForm)
		r.Post("/add-order", h.CreateOrder)
		r.Get("/orders/{id}/items/new", h.NewOrderItemForm)
		r.Post("/orders/{id}/items", h.CreateOrderItem)
		r.Get("/orders/{id}/groups/new", h.NewOrderGroupForm)
		r.Post("/orders/{id}/groups", h.CreateOrderGroup)
		r.Get("/orders/{id}/modal", h.OrderModal)
		r.Get("/orders/menu/match", h.OrderMenuMatch)
		r.Get("/menu", h.Menu)
	})
}

func (h *Handler) log() aqm.Logger {
	return h.logger
}

func (h *Handler) renderTemplate(w http.ResponseWriter, templateName, layout string, data map[string]interface{}) {
	tmpl, err := h.tmplMgr.Get(templateName)
	if err != nil {
		h.log().Error("error loading template", "error", err, "template", templateName)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, layout, data); err != nil {
		h.log().Error("error rendering template", "error", err, "layout", layout)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Home displays the operations dashboard
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.Home")
	defer finish()

	data := map[string]interface{}{
		"Title":    "Operations Dashboard",
		"User":     h.getUserFromSession(r),
		"Template": "home",
	}

	h.renderTemplate(w, "home.html", "base.html", data)
}

// Chat displays the conversational interface
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.Chat")
	defer finish()

	data := map[string]interface{}{
		"Title":    "Chat - Conversational Interface",
		"User":     h.getUserFromSession(r),
		"Template": "chat",
	}

	h.renderTemplate(w, "chat.html", "base.html", data)
}

// Tables displays table management interface
// Menu displays menu management interface
func (h *Handler) Menu(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.Menu")
	defer finish()

	data := map[string]interface{}{
		"Title":    "Menu Management",
		"User":     h.getUserFromSession(r),
		"Template": "menu",
	}

	h.renderTemplate(w, "menu.html", "base.html", data)
}

func (h *Handler) getUserFromSession(r *http.Request) map[string]interface{} {
	session, ok := r.Context().Value("session").(*Session)
	if !ok || session == nil {
		return nil
	}

	return map[string]interface{}{
		"ID":       session.UserID,
		"Username": session.Username,
		"Name":     session.Name,
	}
}
