package operations

import (
	"net/http"
	"time"

	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/telemetry"
	aqmtemplate "github.com/aquamarinepk/aqm/template"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	tmplMgr         *aqmtemplate.Manager
	authnClient     *aqm.ServiceClient
	tableClient     *aqm.ServiceClient
	orderClient     *aqm.ServiceClient
	logger          aqm.Logger
	config          *aqm.Config
	http            *telemetry.HTTP
	sessionStore    *SessionStore
	commandProcessor CommandProcessor
}

func NewHandler(
	tmplMgr *aqmtemplate.Manager,
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

	// Initialize session store
	sessionSecret, _ := config.GetString("auth.session.secret")
	sessionTTLStr, _ := config.GetString("auth.session.ttl")
	sessionTTL, _ := time.ParseDuration(sessionTTLStr)
	if sessionTTL == 0 {
		sessionTTL = 8 * time.Hour
	}
	sessionStore := NewSessionStore([]byte(sessionSecret), sessionTTL)

	// Initialize command processor (deterministic parser for Phase 1)
	commandProcessor := NewDeterministicParser(
		&ServiceClientWrapper{baseURL: tableURL},
		&ServiceClientWrapper{baseURL: orderURL},
	)

	return &Handler{
		tmplMgr:          tmplMgr,
		authnClient:      authnClient,
		tableClient:      tableClient,
		orderClient:      orderClient,
		logger:           logger,
		config:           config,
		http:             telemetry.NewHTTP(),
		sessionStore:     sessionStore,
		commandProcessor: commandProcessor,
	}
}

// RegisterRoutes registers all operations routes using Command/Query pattern
func (h *Handler) RegisterRoutes(r chi.Router) {
	h.log().Info("Registering operations routes...")

	// Public routes
	r.Get("/signin", h.ShowSignIn)
	r.Post("/signin", h.HandleSignIn)
	r.Post("/signout", h.HandleSignOut)

	// Protected routes (require session)
	r.Group(func(r chi.Router) {
		r.Use(h.SessionMiddleware)

		h.log().Info("Registering operational routes...")
		r.Get("/", h.Home)
		r.Get("/chat", h.Chat)
		r.Post("/chat/message", h.HandleChatMessage)
		r.Get("/tables", h.Tables)
		r.Get("/orders", h.Orders)
	})

	h.log().Info("Operations routes registered successfully")
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
func (h *Handler) Tables(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.Tables")
	defer finish()

	data := map[string]interface{}{
		"Title":    "Tables",
		"User":     h.getUserFromSession(r),
		"Template": "tables",
	}

	h.renderTemplate(w, "tables.html", "base.html", data)
}

// Orders displays order management interface
func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.Orders")
	defer finish()

	data := map[string]interface{}{
		"Title":    "Orders",
		"User":     h.getUserFromSession(r),
		"Template": "orders",
	}

	h.renderTemplate(w, "orders.html", "base.html", data)
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
