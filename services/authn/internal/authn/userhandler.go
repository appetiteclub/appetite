package authn

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/appetiteclub/apt"
	authpkg "github.com/appetiteclub/apt/auth"
	"github.com/appetiteclub/apt/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const UserMaxBodyBytes = 1 << 20

// NewUserHandler creates a new UserHandler for the User aggregate.
func NewUserHandler(repo UserRepo, config *apt.Config, logger apt.Logger) *UserHandler {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}
	return &UserHandler{
		repo:   repo,
		logger: logger,
		config: config,
		tlm:    telemetry.NewHTTP(),
	}
}

type UserHandler struct {
	repo   UserRepo
	logger apt.Logger
	config *apt.Config
	tlm    *telemetry.HTTP
}

func (h *UserHandler) RegisterRoutes(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Post("/", h.CreateUser)
		r.Get("/", h.GetAllUsers)
		r.Get("/{id}", h.GetUser)
		r.Put("/{id}", h.UpdateUser)
		r.Delete("/{id}", h.DeleteUser)
		r.Post("/{id}/generate-pin", h.GeneratePIN)
	})
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "UserHandler.CreateUser")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	req, ok := h.decodeUserCreatePayload(w, r)
	if !ok {
		return
	}

	validationErrors := ValidateCreateUserRequest(ctx, req)
	if len(validationErrors) > 0 {
		apt.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

	user := req.ToUser()
	user.EnsureID()
	user.BeforeCreate()

	if err := h.repo.Create(ctx, &user); err != nil {
		log.Error("cannot create user", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not create user")
		return
	}

	// Standard links
	links := apt.RESTfulLinksFor(&user)

	w.WriteHeader(http.StatusCreated)
	apt.RespondSuccess(w, user, links...)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "UserHandler.GetUser")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r)
	if !ok {
		return
	}

	user, err := h.repo.Get(ctx, id)
	if err != nil {
		log.Error("error loading user", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusInternalServerError, "Could not retrieve user")
		return
	}

	if user == nil {
		apt.RespondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Decrypt email before responding
	userData := h.userToMap(user)

	// Standard links
	links := apt.RESTfulLinksFor(user)

	apt.RespondSuccess(w, userData, links...)
}

func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "UserHandler.GetAllUsers")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	users, err := h.repo.List(ctx)
	if err != nil {
		log.Error("error retrieving users", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not list all users")
		return
	}

	// Decrypt emails before responding
	usersData := make([]map[string]interface{}, len(users))
	for i, user := range users {
		usersData[i] = h.userToMap(user)
	}

	// Collection response
	apt.RespondCollection(w, usersData, "user")
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "UserHandler.UpdateUser")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r)
	if !ok {
		return
	}

	req, ok := h.decodeUserUpdatePayload(w, r)
	if !ok {
		return
	}

	validationErrors := ValidateUpdateUserRequest(ctx, id, req)
	if len(validationErrors) > 0 {
		apt.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

	user := req.ToUser()
	user.SetID(id)
	user.BeforeUpdate()

	if err := h.repo.Save(ctx, &user); err != nil {
		log.Error("cannot save user", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusInternalServerError, "Could not update user")
		return
	}

	// Standard links
	links := apt.RESTfulLinksFor(&user)

	apt.RespondSuccess(w, user, links...)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "UserHandler.DeleteUser")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		log.Error("error deleting user", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusInternalServerError, "Could not delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) GeneratePIN(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "UserHandler.GeneratePIN")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r)
	if !ok {
		return
	}

	// Get the user
	user, err := h.repo.Get(ctx, id)
	if err != nil {
		log.Error("error loading user", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusInternalServerError, "Could not retrieve user")
		return
	}

	if user == nil {
		apt.RespondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Generate the PIN
	pin, err := GeneratePINForUser(ctx, h.repo, h.config, user)
	if err != nil {
		log.Error("error generating PIN", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusInternalServerError, "Could not generate PIN")
		return
	}

	// Save the user with the new PIN
	user.UpdatedBy = "pin:generation"
	if err := h.repo.Save(ctx, user); err != nil {
		log.Error("error saving user with PIN", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusInternalServerError, "Could not save PIN")
		return
	}

	// TODO: SECURITY - Remove PIN logging in production! This is only for development.
	log.Info("⚠️  DEVELOPMENT ONLY - PIN generated for user (REMOVE THIS LOG IN PRODUCTION!)", "id", id.String(), "pin", pin)

	// Return the PIN (only shown once)
	response := map[string]interface{}{
		"pin":     pin,
		"user_id": id,
		"message": "PIN generated successfully. This is the only time it will be displayed.",
	}

	apt.RespondSuccess(w, response)
}

// Helper methods following same patterns as ListHandler

func (h *UserHandler) log(req ...*http.Request) apt.Logger {
	if len(req) > 0 && req[0] != nil {
		r := req[0]
		return h.logger.With(
			"request_id", apt.RequestIDFrom(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
		)
	}
	return h.logger
}

func (h *UserHandler) parseIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, "id")
	if strings.TrimSpace(idStr) == "" {
		apt.RespondError(w, http.StatusBadRequest, "Missing or invalid id")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid id format")
		return uuid.Nil, false
	}

	return id, true
}

func (h *UserHandler) decodeUserCreatePayload(w http.ResponseWriter, r *http.Request) (UserCreateRequest, bool) {
	var req UserCreateRequest

	r.Body = http.MaxBytesReader(w, r.Body, UserMaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return req, false
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		apt.RespondError(w, http.StatusBadRequest, "Request body is empty")
		return req, false
	}

	if err := json.Unmarshal(body, &req); err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Could not parse JSON")
		return req, false
	}

	return req, true
}

func (h *UserHandler) decodeUserUpdatePayload(w http.ResponseWriter, r *http.Request) (UserUpdateRequest, bool) {
	var req UserUpdateRequest

	r.Body = http.MaxBytesReader(w, r.Body, UserMaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return req, false
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		apt.RespondError(w, http.StatusBadRequest, "Request body is empty")
		return req, false
	}

	if err := json.Unmarshal(body, &req); err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Could not parse JSON")
		return req, false
	}

	return req, true
}

func (h *UserHandler) userToMap(user *User) map[string]interface{} {
	email := ""
	if len(user.EmailCT) > 0 && len(user.EmailIV) > 0 && len(user.EmailTag) > 0 {
		encryptionKey, _ := h.config.GetString("auth.encryption.key")
		encrypted := &authpkg.EncryptedData{
			Ciphertext: user.EmailCT,
			IV:         user.EmailIV,
			Tag:        user.EmailTag,
		}
		decrypted, err := authpkg.DecryptEmail(encrypted, []byte(encryptionKey))
		if err != nil {
			h.log().Error("failed to decrypt email", "error", err, "user_id", user.ID)
			email = "[encrypted]"
		} else {
			email = decrypted
		}
	}

	return map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"name":       user.Name,
		"email":      email,
		"status":     user.Status,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
		"created_by": user.CreatedBy,
		"updated_by": user.UpdatedBy,
	}
}
