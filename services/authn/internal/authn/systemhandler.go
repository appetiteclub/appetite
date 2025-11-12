package authn

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/telemetry"
	"github.com/go-chi/chi/v5"
)

// SystemHandler manages system-level operations like bootstrap
type SystemHandler struct {
	userRepo UserRepo
	logger   aqm.Logger
	config   *aqm.Config
	tlm      *telemetry.HTTP
}

// BootstrapStatusResponse represents the current bootstrap status
type BootstrapStatusResponse struct {
	NeedsBootstrap bool   `json:"needs_bootstrap"`
	SuperadminID   string `json:"superadmin_id,omitempty"` // Only if !needs_bootstrap
}

// BootstrapResponse represents the result of bootstrap operation
type BootstrapResponse struct {
	SuperadminID string `json:"superadmin_id"`
	Email        string `json:"email"`
	Password     string `json:"password"` // Generated password
}

const SuperadminEmail = "superadmin@system"

func NewSystemHandler(userRepo UserRepo, config *aqm.Config, logger aqm.Logger) *SystemHandler {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &SystemHandler{
		userRepo: userRepo,
		logger:   logger,
		config:   config,
		tlm:      telemetry.NewHTTP(),
	}
}

// RegisterRoutes registers system management routes
func (h *SystemHandler) RegisterRoutes(r chi.Router) {
	h.log().Info("Registering system routes...")

	r.Get("/system/bootstrap-status", h.GetBootstrapStatus)
	r.Post("/system/bootstrap", h.Bootstrap)

	h.log().Info("System routes registered successfully")
}

// GetBootstrapStatus checks if the system needs bootstrap
func (h *SystemHandler) GetBootstrapStatus(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "SystemHandler.GetBootstrapStatus")
	defer finish()

	log := h.log(r)

	superadmin, err := GenerateBootstrapStatus(r.Context(), h.userRepo, h.config)
	if err != nil {
		log.Error("failed to check superadmin user", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Failed to check bootstrap state")
		return
	}

	if superadmin == nil {
		aqm.RespondSuccess(w, BootstrapStatusResponse{NeedsBootstrap: true})
		return
	}

	aqm.RespondSuccess(w, BootstrapStatusResponse{
		NeedsBootstrap: false,
		SuperadminID:   superadmin.ID.String(),
	})
}

// Bootstrap creates the superadmin user if it doesn't exist
func (h *SystemHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "SystemHandler.Bootstrap")
	defer finish()

	log := h.log(r)

	user, password, err := BootstrapSuperadmin(r.Context(), h.userRepo, h.config)
	if err != nil {
		log.Error("failed to bootstrap superadmin", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Failed to bootstrap superadmin")
		return
	}

	if password == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(BootstrapResponse{
			SuperadminID: user.ID.String(),
			Email:        SuperadminEmail,
			Password:     "",
		})
		return
	}

	bannerLines := []string{
		"═══════════════════════════════════════════════════════════",
		"  SUPERADMIN BOOTSTRAP CREDENTIALS",
		"═══════════════════════════════════════════════════════════",
		fmt.Sprintf("  Email:    %s", SuperadminEmail),
		fmt.Sprintf("  Password: %s", password),
		fmt.Sprintf("  UserID:   %s", user.ID.String()),
		"═══════════════════════════════════════════════════════════",
		"  IMPORTANT: Save these credentials securely!",
		"  TODO: Implement mandatory password change on first login",
		"═══════════════════════════════════════════════════════════",
	}

	for _, line := range bannerLines {
		log.Info(line)
	}

	log.Info("superadmin bootstrap credentials",
		"email", SuperadminEmail,
		"user_id", user.ID,
	)

	log.Info("superadmin created successfully", "id", user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BootstrapResponse{
		SuperadminID: user.ID.String(),
		Email:        SuperadminEmail,
		Password:     password,
	})
}

func generateSecurePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()"
	b := make([]byte, length)

	for i := range b {
		randomByte := make([]byte, 1)
		rand.Read(randomByte)
		b[i] = charset[int(randomByte[0])%len(charset)]
	}

	return string(b)
}

func (h *SystemHandler) log(req ...*http.Request) aqm.Logger {
	if len(req) > 0 && req[0] != nil {
		r := req[0]
		return h.logger.With(
			"request_id", aqm.RequestIDFrom(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
		)
	}
	return h.logger
}
