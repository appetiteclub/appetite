package authz

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/seed"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	authpkg "github.com/appetiteclub/apt/auth"
)

// BootstrapService handles the coordination of system bootstrap
type BootstrapService struct {
	roleRepo   RoleRepo
	grantRepo  GrantRepo
	httpClient *http.Client
	seedFS     embed.FS
	logger     apt.Logger
	config     *apt.Config
}

// BootstrapStatusResponse matches AuthN response
type BootstrapStatusResponse struct {
	NeedsBootstrap bool   `json:"needs_bootstrap"`
	SuperadminID   string `json:"superadmin_id,omitempty"`
}

// BootstrapResponse matches AuthN response
type BootstrapResponse struct {
	SuperadminID string `json:"superadmin_id"`
	Email        string `json:"email"`
	Password     string `json:"password"`
}

func NewBootstrapService(roleRepo RoleRepo, grantRepo GrantRepo, seedFS embed.FS, config *apt.Config, logger apt.Logger) *BootstrapService {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}
	return &BootstrapService{
		roleRepo:  roleRepo,
		grantRepo: grantRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		seedFS: seedFS,
		logger: logger,
		config: config,
	}
}

const authzSeedApplication = "authz"

// Bootstrap orchestrates the complete bootstrap process
func (s *BootstrapService) Bootstrap(ctx context.Context) error {
	s.log().Info("Starting bootstrap process...")

	status, err := s.getBootstrapStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get bootstrap status: %w", err)
	}

	var superadminID string

	if status.NeedsBootstrap {
		s.log().Info("System needs bootstrap, triggering AuthN bootstrap...")

		response, err := s.triggerBootstrap(ctx)
		if err != nil {
			return fmt.Errorf("failed to trigger bootstrap: %w", err)
		}

		superadminID = response.SuperadminID
		s.log().Info("Bootstrap triggered successfully",
			"superadmin_id", response.SuperadminID,
			"email", response.Email,
			"password", response.Password) // log credentials for initial setup
	} else {
		s.log().Info("System already bootstrapped", "superadmin_id", status.SuperadminID)
		superadminID = status.SuperadminID
	}

	if err := s.bootstrapRolesAndGrants(ctx, superadminID); err != nil {
		return fmt.Errorf("failed to bootstrap roles and grants: %w", err)
	}

	s.log().Info("Bootstrap process completed successfully")
	return nil
}

// getBootstrapStatus calls AuthN to check bootstrap status
func (s *BootstrapService) getBootstrapStatus(ctx context.Context) (*BootstrapStatusResponse, error) {
	authNURL, _ := s.config.GetString("auth.authn.url")
	url := authNURL + "/system/bootstrap-status"
	s.log().Info("AuthN URL: " + url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bootstrap status request failed: %d", resp.StatusCode)
	}

	// Parse wrapped response
	var wrapped struct {
		Data BootstrapStatusResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapped); err != nil {
		return nil, err
	}

	return &wrapped.Data, nil
}

// triggerBootstrap calls AuthN to create superadmin
func (s *BootstrapService) triggerBootstrap(ctx context.Context) (*BootstrapResponse, error) {
	authNURL, _ := s.config.GetString("auth.authn.url")
	url := authNURL + "/system/bootstrap"

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bootstrap request failed: %d", resp.StatusCode)
	}

	var response BootstrapResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// bootstrapRolesAndGrants seeds roles and creates system user grants
func (s *BootstrapService) bootstrapRolesAndGrants(ctx context.Context, superadminID string) error {
	if err := s.applyRoleSeeds(ctx); err != nil {
		return fmt.Errorf("failed to seed roles: %w", err)
	}

	// Step 2: Ensure superadmin grant exists (idempotent)
	if err := s.ensureSuperadminGrant(ctx, superadminID); err != nil {
		return fmt.Errorf("failed to ensure superadmin grant: %w", err)
	}

	// Step 3: Ensure system user grants exist (idempotent)
	if err := s.ensureSystemUserGrants(ctx); err != nil {
		s.log().Info("⚠️  failed to ensure system user grants (non-fatal)", "error", err)
		// Don't fail bootstrap if system user grants fail - they can be created manually
	}

	return nil
}

// ensureSuperadminGrant creates grant for superadmin if it doesn't exist
func (s *BootstrapService) ensureSuperadminGrant(ctx context.Context, superadminID string) error {
	if strings.TrimSpace(superadminID) == "" {
		return fmt.Errorf("superadmin id is required")
	}

	// Get superadmin role
	role, err := s.roleRepo.GetByName(ctx, "superadmin")
	if err != nil {
		return fmt.Errorf("superadmin role not found: %w", err)
	}
	if role == nil {
		return fmt.Errorf("superadmin role not found")
	}

	// Parse superadmin UUID
	userID, err := uuid.Parse(superadminID)
	if err != nil {
		return fmt.Errorf("invalid superadmin ID: %w", err)
	}

	// Check if grant already exists (idempotent)
	grants, err := s.grantRepo.ListByUserID(ctx, userID)
	if err != nil {
		s.log().Error("Failed to check existing grants, proceeding anyway", "error", err)
	} else {
		for _, g := range grants {
			if g.GrantType == GrantTypeRole && g.Value == role.ID.String() {
				s.log().Info("Superadmin grant already exists", "grant_id", g.ID)
				return nil
			}
		}
	}

	// Create grant
	grant := &Grant{
		ID:        uuid.New(),
		UserID:    userID,
		GrantType: GrantTypeRole,
		Value:     role.ID.String(),
		Scope:     Scope{Type: "global", ID: ""},
		ExpiresAt: nil,
		Status:    authpkg.UserStatusActive,
		CreatedAt: time.Now(),
		CreatedBy: "system",
		UpdatedAt: time.Now(),
		UpdatedBy: "system",
	}

	if err := s.grantRepo.Create(ctx, grant); err != nil {
		return fmt.Errorf("failed to create superadmin grant: %w", err)
	}

	s.log().Info("Superadmin grant created successfully",
		"grant_id", grant.ID,
		"user_id", userID,
		"role_id", role.ID)

	return nil
}

// ensureSystemUserGrants creates grants for system users (agent, demo users)
func (s *BootstrapService) ensureSystemUserGrants(ctx context.Context) error {
	// Define system users and their roles
	systemUsers := []struct {
		email    string
		roleName string
	}{
		{"agent@system", "conversational-interface-manager"},
		{"admin@example.com", "admin"},
		{"user@example.com", "user"},
	}

	for _, u := range systemUsers {
		userID, err := s.getUserIDByEmail(ctx, u.email)
		if err != nil {
			s.log().Info("⚠️  failed to get user ID", "email", u.email, "error", err)
			continue
		}

		if err := s.ensureUserGrant(ctx, userID, u.roleName); err != nil {
			s.log().Info("⚠️  failed to ensure grant", "email", u.email, "role", u.roleName, "error", err)
			continue
		}

		// Special banner for agent user
		if u.email == "agent@system" {
			bannerLines := []string{
				"═══════════════════════════════════════════════════════════",
				"  AGENT USER GRANT ASSIGNED",
				"═══════════════════════════════════════════════════════════",
				fmt.Sprintf("  Email:    %s", u.email),
				fmt.Sprintf("  UserID:   %s", userID.String()),
				fmt.Sprintf("  Role:     %s", u.roleName),
				"═══════════════════════════════════════════════════════════",
				"  IMPORTANT: Agent credentials are in AuthN service logs",
				"  Agent can delegate to other users via .login PIN command",
				"═══════════════════════════════════════════════════════════",
			}

			for _, line := range bannerLines {
				s.log().Info(line)
			}
		}

		s.log().Info("System user grant ensured", "email", u.email, "role", u.roleName)
	}

	return nil
}

// getUserIDByEmail queries AuthN to get user ID by email
func (s *BootstrapService) getUserIDByEmail(ctx context.Context, email string) (uuid.UUID, error) {
	authNURL, _ := s.config.GetString("auth.authn.url")
	url := fmt.Sprintf("%s/system/users/by-email/%s", authNURL, email)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return uuid.Nil, fmt.Errorf("user not found: %s", email)
	}

	if resp.StatusCode != http.StatusOK {
		return uuid.Nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var wrapper struct {
		Data struct {
			UserID string `json:"user_id"`
			Email  string `json:"email"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return uuid.Nil, fmt.Errorf("decode response: %w", err)
	}

	userID, err := uuid.Parse(wrapper.Data.UserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID: %w", err)
	}

	return userID, nil
}

// ensureUserGrant creates a grant for a user with a specific role
func (s *BootstrapService) ensureUserGrant(ctx context.Context, userID uuid.UUID, roleName string) error {
	if userID == uuid.Nil {
		return fmt.Errorf("invalid user ID")
	}

	// Get role by name
	role, err := s.roleRepo.GetByName(ctx, roleName)
	if err != nil {
		return fmt.Errorf("role %s not found: %w", roleName, err)
	}
	if role == nil {
		return fmt.Errorf("role %s not found", roleName)
	}

	// Check if grant already exists (idempotent)
	grants, err := s.grantRepo.ListByUserID(ctx, userID)
	if err != nil {
		s.log().Error("Failed to check existing grants", "error", err)
	} else {
		for _, g := range grants {
			if g.GrantType == GrantTypeRole && g.Value == role.ID.String() {
				s.log().Debug("Grant already exists", "user_id", userID, "role", roleName)
				return nil
			}
		}
	}

	// Create grant
	grant := &Grant{
		ID:        uuid.New(),
		UserID:    userID,
		GrantType: GrantTypeRole,
		Value:     role.ID.String(),
		Scope:     Scope{Type: "global", ID: ""},
		ExpiresAt: nil,
		Status:    authpkg.UserStatusActive,
		CreatedAt: time.Now(),
		CreatedBy: "system",
		UpdatedAt: time.Now(),
		UpdatedBy: "system",
	}

	if err := s.grantRepo.Create(ctx, grant); err != nil {
		return fmt.Errorf("failed to create grant: %w", err)
	}

	s.log().Info("Grant created successfully",
		"grant_id", grant.ID,
		"user_id", userID,
		"role", roleName)

	return nil
}

func (s *BootstrapService) applyRoleSeeds(ctx context.Context) error {
	tracker, db, err := s.newSeedTracker()
	if err != nil {
		return err
	}

	defs, err := s.roleSeedDefinitions()
	if err != nil {
		return err
	}

	if len(defs) == 0 {
		s.log().Info("No AuthZ role seeds to apply")
		return nil
	}

	// NOTE: when the roles collection gets wiped but the _seeds tracker remains,
	// seed.Apply would treat every seed as already executed and skip re-creating
	// the data. We detect that scenario and drop the tracker entries so the
	// bootstrap path can restore the roles without manual intervention. A future
	// aqm/seed enhancement (e.g., forced re-seeding) would make this workaround
	// unnecessary.
	empty, err := s.rolesMissing(ctx)
	if err != nil {
		return err
	}
	if empty {
		if err := s.resetRoleSeedRecords(ctx, db); err != nil {
			return err
		}
	}

	s.log().Info("Applying AuthZ role seeds", "count", len(defs))
	if err := seed.Apply(ctx, tracker, defs, authzSeedApplication); err != nil {
		return fmt.Errorf("apply role seeds: %w", err)
	}
	s.log().Info("AuthZ role seeds applied successfully")
	return nil
}

func (s *BootstrapService) roleSeedDefinitions() ([]seed.Seed, error) {
	extra, err := loadRoleSeeds(s.seedFS)
	if err != nil {
		return nil, fmt.Errorf("load role seeds: %w", err)
	}

	all := append(defaultRoleSeeds(), extra...)
	defs := make([]seed.Seed, 0, len(all))
	for _, role := range all {
		roleData := role
		name := strings.TrimSpace(roleData.Name)
		if name == "" {
			continue
		}

		defs = append(defs, seed.Seed{
			ID:          fmt.Sprintf("2024-11-15_authz_role_%s", seedIdentifier(name)),
			Description: fmt.Sprintf("Ensure AuthZ role %s", name),
			Run: func(ctx context.Context) error {
				return s.ensureRoleSeed(ctx, roleData)
			},
		})
	}

	return defs, nil
}

func (s *BootstrapService) rolesMissing(ctx context.Context) (bool, error) {
	roles, err := s.roleRepo.List(ctx)
	if err != nil {
		return false, fmt.Errorf("list roles: %w", err)
	}
	return len(roles) == 0, nil
}

func (s *BootstrapService) resetRoleSeedRecords(ctx context.Context, db *mongo.Database) error {
	if db == nil {
		return errors.New("mongo database is nil")
	}
	collection := db.Collection("_seeds")
	if collection == nil {
		return errors.New("seed tracker collection missing")
	}
	filter := bson.M{"_id": bson.M{"$regex": "^2024-11-15_authz_role_"}}
	if _, err := collection.DeleteMany(ctx, filter); err != nil {
		return fmt.Errorf("reset role seed records: %w", err)
	}
	return nil
}

func (s *BootstrapService) newSeedTracker() (seed.Tracker, *mongo.Database, error) {
	provider, ok := s.roleRepo.(mongoDatabaseProvider)
	if !ok {
		return nil, nil, errors.New("role repository does not expose MongoDB access for seeding")
	}
	db := provider.Database()
	if db == nil {
		return nil, nil, errors.New("role repository database is not initialized")
	}
	return seed.NewMongoTracker(db), db, nil
}

type mongoDatabaseProvider interface {
	Database() *mongo.Database
}

func seedIdentifier(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}

	replacer := strings.NewReplacer("@", "_", ".", "_", "-", "_", "+", "_", " ", "_")
	value = replacer.Replace(value)
	var builder strings.Builder
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_':
			builder.WriteRune(r)
		}
	}
	result := builder.String()
	if result == "" {
		return "seed"
	}
	return result
}

func (s *BootstrapService) log() apt.Logger {
	return s.logger
}

// BootstrapFunc returns a function suitable for apt.LifecycleHooks.OnStart.
// It wraps the Bootstrap method and performs logging so callers (like main)
// can pass it directly to OnStart.
func BootstrapFunc(s *BootstrapService, logger apt.Logger) func(ctx context.Context) error {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}

	return func(ctx context.Context) error {
		if err := s.Bootstrap(ctx); err != nil {
			logger.Errorf("Bootstrap failed: %v", err)
		} else {
			logger.Infof("Bootstrap completed successfully")
		}
		// Keep behavior compatible with previous inline handler which did
		// not return the bootstrap error to the lifecycle runner.
		return nil
	}
}
