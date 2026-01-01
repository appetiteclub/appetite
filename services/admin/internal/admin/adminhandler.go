package admin

import (
	"net/http"

	"github.com/appetiteclub/apt"
	authpkg "github.com/appetiteclub/apt/auth"
	"github.com/appetiteclub/apt/telemetry"
	aqmtemplate "github.com/appetiteclub/apt/template"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AdminHandler manages the web interface for system administration
type AdminHandler struct {
	logger    apt.Logger
	config     *apt.Config
	tmplMgr   *aqmtemplate.Manager
	userRepo  UserRepo
	roleRepo  RoleRepo
	grantRepo GrantRepo
	tlm       *telemetry.HTTP
}

func NewAdminHandler(tmplMgr *aqmtemplate.Manager, userRepo UserRepo, roleRepo RoleRepo, grantRepo GrantRepo, logger apt.Logger, config *apt.Config) *AdminHandler {
	return &AdminHandler{
		logger:    logger,
		config:     config,
		tmplMgr:   tmplMgr,
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		grantRepo: grantRepo,
		tlm:       telemetry.NewHTTP(),
	}
}

// RegisterRoutes registers all admin routes using Commands/Queries pattern
func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	h.log().Info("Registering admin routes...")

	r.Get("/", h.Home)
	r.Get("/health", h.Health)

	h.log().Info("Registering user management routes...")
	r.Get("/list-users", h.ListUsers)
	r.Get("/new-user", h.NewUser)
	r.Post("/create-user", h.CreateUser)
	r.Get("/show-user/{id}", h.ShowUser)
	r.Get("/edit-user/{id}", h.EditUser)
	r.Post("/update-user/{id}", h.UpdateUser)
	r.Post("/delete-user/{id}", h.DeleteUser)

	h.log().Info("Registering role management routes...")
	r.Get("/list-roles", h.ListRoles)
	r.Get("/new-role", h.NewRole)
	r.Post("/create-role", h.CreateRole)
	r.Get("/show-role/{id}", h.ShowRole)
	r.Get("/edit-role/{id}", h.EditRole)
	r.Post("/update-role/{id}", h.UpdateRole)
	r.Post("/delete-role/{id}", h.DeleteRole)

	h.log().Info("Registering grant management routes...")
	r.Get("/user-grants/{userId}", h.UserGrants)
	r.Post("/create-grant", h.CreateGrant)
	r.Post("/delete-grant/{id}", h.DeleteGrant)

	h.log().Info("Registering table management routes...")
	r.Get("/list-tables", h.ListTables)

	h.log().Info("Admin routes registered successfully")
}

// Health check endpoint
func (h *AdminHandler) Health(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.Health")
	defer finish()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Home renders the admin dashboard
func (h *AdminHandler) Home(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.Home")
	defer finish()

	log := h.log(r)

	tmpl, err := h.tmplMgr.Get("home.html")
	if err != nil {
		log.Error("error getting home template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":     "Dashboard",
		"ActiveNav": "dashboard",
	}

	if err := tmpl.ExecuteTemplate(w, "home.html", data); err != nil {
		log.Error("error executing home template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.ListUsers")
	defer finish()

	log := h.log(r)

	tmpl, err := h.tmplMgr.Get("users.html")
	if err != nil {
		log.Error("error getting users template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	users, err := h.userRepo.List(r.Context())
	if err != nil {
		log.Error("error fetching users", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":     "Users",
		"ActiveNav": "users",
		"Template":  "users-content",
		"Users":     users,
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error("error executing users template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) NewUser(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.NewUser")
	defer finish()

	log := h.log(r)

	tmpl, err := h.tmplMgr.Get("new-user.html")
	if err != nil {
		log.Error("error getting new-user template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":     "New User",
		"ActiveNav": "users",
		"Template":  "new-user",
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error("error executing base template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.CreateUser")
	defer finish()

	log := h.log(r)

	if err := r.ParseForm(); err != nil {
		log.Error("error parsing form", "error", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	req := &CreateUserRequest{
		Email:    r.FormValue("email"),
		Name:     r.FormValue("name"),
		Password: r.FormValue("password"),
	}

	if req.Email == "" || req.Name == "" || req.Password == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.Create(r.Context(), req)
	if err != nil {
		log.Error("error creating user", "error", err)
		http.Error(w, "Cannot create user: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("User created successfully", "id", user.ID, "email", user.Email)

	w.Header().Set("HX-Redirect", "/list-users")
	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) ShowUser(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.ShowUser")
	defer finish()

	log := h.log(r)

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.Get(r.Context(), id)
	if err != nil {
		log.Error("error fetching user", "error", err, "id", id)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	tmpl, err := h.tmplMgr.Get("show-user.html")
	if err != nil {
		log.Error("error getting show-user template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":     "View User",
		"ActiveNav": "users",
		"Template":  "show-user",
		"User":      user,
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error("error executing template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) EditUser(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.EditUser")
	defer finish()

	log := h.log(r)

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.Get(r.Context(), id)
	if err != nil {
		log.Error("error fetching user", "error", err, "id", id)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	tmpl, err := h.tmplMgr.Get("edit-user.html")
	if err != nil {
		log.Error("error getting edit-user template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":     "Edit User",
		"ActiveNav": "users",
		"Template":  "edit-user",
		"User":      user,
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error("error executing template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.UpdateUser")
	defer finish()

	log := h.log(r)

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	req := &UpdateUserRequest{
		Email:  r.FormValue("email"),
		Name:   r.FormValue("name"),
		Status: r.FormValue("status"),
	}

	user, err := h.userRepo.Update(r.Context(), id, req)
	if err != nil {
		log.Error("error updating user", "error", err)
		http.Error(w, "Cannot update user: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("User updated successfully", "id", user.ID)
	w.Header().Set("HX-Redirect", "/list-users")
	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.DeleteUser")
	defer finish()

	log := h.log(r)

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Error("invalid user ID", "error", err, "id", idStr)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := h.userRepo.Delete(r.Context(), id); err != nil {
		log.Error("error deleting user", "error", err, "id", id)
		http.Error(w, "Cannot delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.ListRoles")
	defer finish()

	log := h.log(r)

	tmpl, err := h.tmplMgr.Get("roles.html")
	if err != nil {
		log.Error("error getting roles template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	roles, err := h.roleRepo.List(r.Context())
	if err != nil {
		log.Error("error fetching roles", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":     "Roles",
		"ActiveNav": "roles",
		"Template":  "roles-content",
		"Roles":     roles,
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error("error executing roles template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) NewRole(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.NewRole")
	defer finish()

	log := h.log(r)

	tmpl, err := h.tmplMgr.Get("new-role.html")
	if err != nil {
		log.Error("error getting new-role template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":              "New Role",
		"ActiveNav":          "roles",
		"Template":           "new-role",
		"PermissionRegistry": authpkg.PermissionRegistry,
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error("error executing base template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.CreateRole")
	defer finish()

	log := h.log(r)

	if err := r.ParseForm(); err != nil {
		log.Error("error parsing form", "error", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	req := &CreateRoleRequest{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Permissions: r.Form["permissions"],
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	role, err := h.roleRepo.Create(r.Context(), req)
	if err != nil {
		log.Error("error creating role", "error", err)
		http.Error(w, "Cannot create role: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("Role created successfully", "id", role.ID, "name", role.Name)

	w.Header().Set("HX-Redirect", "/list-roles")
	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) ShowRole(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.ShowRole")
	defer finish()

	log := h.log(r)

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "Missing role ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	role, err := h.roleRepo.Get(r.Context(), id)
	if err != nil {
		log.Error("error fetching role", "error", err, "id", id)
		http.Error(w, "Role not found", http.StatusNotFound)
		return
	}

	tmpl, err := h.tmplMgr.Get("show-role.html")
	if err != nil {
		log.Error("error getting show-role template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":     "View Role",
		"ActiveNav": "roles",
		"Template":  "show-role",
		"Role":      role,
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error("error executing template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) EditRole(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.EditRole")
	defer finish()

	log := h.log(r)

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "Missing role ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	role, err := h.roleRepo.Get(r.Context(), id)
	if err != nil {
		log.Error("error fetching role", "error", err, "id", id)
		http.Error(w, "Role not found", http.StatusNotFound)
		return
	}

	tmpl, err := h.tmplMgr.Get("edit-role.html")
	if err != nil {
		log.Error("error getting edit-role template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":              "Edit Role",
		"ActiveNav":          "roles",
		"Template":           "edit-role",
		"Role":               role,
		"PermissionRegistry": authpkg.PermissionRegistry,
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error("error executing template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.UpdateRole")
	defer finish()

	log := h.log(r)

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	req := &UpdateRoleRequest{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Permissions: r.Form["permissions"],
		Status:      r.FormValue("status"),
	}

	role, err := h.roleRepo.Update(r.Context(), id, req)
	if err != nil {
		log.Error("error updating role", "error", err)
		http.Error(w, "Cannot update role: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("Role updated successfully", "id", role.ID)
	w.Header().Set("HX-Redirect", "/list-roles")
	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.DeleteRole")
	defer finish()

	log := h.log(r)

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "Missing role ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Error("invalid role ID", "error", err, "id", idStr)
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	if err := h.roleRepo.Delete(r.Context(), id); err != nil {
		log.Error("error deleting role", "error", err, "id", id)
		http.Error(w, "Cannot delete role", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) UserGrants(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.UserGrants")
	defer finish()

	log := h.log(r)

	idStr := chi.URLParam(r, "userId")
	if idStr == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.Get(r.Context(), userID)
	if err != nil {
		log.Error("error fetching user", "error", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	grants, err := h.grantRepo.ListByUser(r.Context(), userID)
	if err != nil {
		log.Error("error fetching grants", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	roles, err := h.roleRepo.List(r.Context())
	if err != nil {
		log.Error("error fetching roles", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err := h.tmplMgr.Get("user-grants.html")
	if err != nil {
		log.Error("error getting user-grants template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":              "User Grants",
		"ActiveNav":          "users",
		"Template":           "user-grants",
		"User":               user,
		"Grants":             grants,
		"Roles":              roles,
		"PermissionRegistry": authpkg.PermissionRegistry,
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error("error executing template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) CreateGrant(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.CreateGrant")
	defer finish()

	log := h.log(r)

	if err := r.ParseForm(); err != nil {
		log.Error("error parsing form", "error", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(r.FormValue("user_id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	req := &CreateGrantRequest{
		UserID:    userID,
		GrantType: r.FormValue("grant_type"),
		Value:     r.FormValue("value"),
		Scope: Scope{
			Type: r.FormValue("scope_type"),
			ID:   r.FormValue("scope_id"),
		},
	}

	grant, err := h.grantRepo.Create(r.Context(), req)
	if err != nil {
		log.Error("error creating grant", "error", err)
		http.Error(w, "Cannot create grant: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("Grant created successfully", "id", grant.ID, "user_id", grant.UserID)

	w.Header().Set("HX-Redirect", "/user-grants/"+userID.String())
	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) DeleteGrant(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.DeleteGrant")
	defer finish()

	log := h.log(r)

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "Missing grant ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Error("invalid grant ID", "error", err, "id", idStr)
		http.Error(w, "Invalid grant ID", http.StatusBadRequest)
		return
	}

	if err := h.grantRepo.Delete(r.Context(), id); err != nil {
		log.Error("error deleting grant", "error", err, "id", id)
		http.Error(w, "Cannot delete grant", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) ListTables(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "AdminHandler.ListTables")
	defer finish()

	log := h.log(r)

	tmpl, err := h.tmplMgr.Get("list-tables.html")
	if err != nil {
		log.Error("error getting list-tables template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":     "Tables",
		"ActiveNav": "tables",
		"Template":  "list-tables-content",
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error("error executing list-tables template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) log(req ...*http.Request) apt.Logger {
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
