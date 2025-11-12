package authz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	authpkg "github.com/aquamarinepk/aqm/auth"
)

// Simple in-memory repo implementations for testing
type testGrantRepo struct {
	grants []*Grant
}

func (r *testGrantRepo) Create(ctx context.Context, grant *Grant) error {
	grant.BeforeCreate()
	r.grants = append(r.grants, grant)
	return nil
}

func (r *testGrantRepo) Get(ctx context.Context, id uuid.UUID) (*Grant, error) {
	for _, grant := range r.grants {
		if grant.ID == id {
			return grant, nil
		}
	}
	return nil, nil
}

func (r *testGrantRepo) Save(ctx context.Context, grant *Grant) error {
	for i, g := range r.grants {
		if g.ID == grant.ID {
			grant.BeforeUpdate()
			r.grants[i] = grant
			return nil
		}
	}
	return nil
}

func (r *testGrantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	for i, grant := range r.grants {
		if grant.ID == id {
			r.grants = append(r.grants[:i], r.grants[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *testGrantRepo) List(ctx context.Context) ([]*Grant, error) {
	return r.grants, nil
}

func (r *testGrantRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*Grant, error) {
	var result []*Grant
	for _, grant := range r.grants {
		if grant.UserID == userID {
			result = append(result, grant)
		}
	}
	return result, nil
}

func (r *testGrantRepo) ListByScope(ctx context.Context, scope Scope) ([]*Grant, error) {
	var result []*Grant
	for _, grant := range r.grants {
		if grant.MatchesScope(scope) {
			result = append(result, grant)
		}
	}
	return result, nil
}

func (r *testGrantRepo) ListExpired(ctx context.Context) ([]*Grant, error) {
	var result []*Grant
	now := time.Now()
	for _, grant := range r.grants {
		if grant.ExpiresAt != nil && grant.ExpiresAt.Before(now) {
			result = append(result, grant)
		}
	}
	return result, nil
}

type testRoleRepo struct {
	roles []*Role
}

func (r *testRoleRepo) Create(ctx context.Context, role *Role) error {
	role.BeforeCreate()
	r.roles = append(r.roles, role)
	return nil
}

func (r *testRoleRepo) Get(ctx context.Context, id uuid.UUID) (*Role, error) {
	for _, role := range r.roles {
		if role.ID == id {
			return role, nil
		}
	}
	return nil, nil
}

func (r *testRoleRepo) GetByName(ctx context.Context, name string) (*Role, error) {
	for _, role := range r.roles {
		if role.Name == name {
			return role, nil
		}
	}
	return nil, nil
}

func (r *testRoleRepo) Save(ctx context.Context, role *Role) error {
	for i, rl := range r.roles {
		if rl.ID == role.ID {
			role.BeforeUpdate()
			r.roles[i] = role
			return nil
		}
	}
	return nil
}

func (r *testRoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	for i, role := range r.roles {
		if role.ID == id {
			r.roles = append(r.roles[:i], r.roles[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *testRoleRepo) List(ctx context.Context) ([]*Role, error) {
	return r.roles, nil
}

func (r *testRoleRepo) ListByStatus(ctx context.Context, status string) ([]*Role, error) {
	var result []*Role
	for _, role := range r.roles {
		if string(role.Status) == status {
			result = append(result, role)
		}
	}
	return result, nil
}

func TestNewPolicyEngine(t *testing.T) {
	grantRepo := &testGrantRepo{}
	roleRepo := &testRoleRepo{}

	engine := NewPolicyEngine(roleRepo, grantRepo)

	if engine == nil {
		t.Error("NewPolicyEngine() returned nil")
	}
	if engine.roleRepo != roleRepo {
		t.Error("roleRepo not set correctly")
	}
	if engine.grantRepo != grantRepo {
		t.Error("grantRepo not set correctly")
	}
}

func TestPolicyEngineHasDirectPermission(t *testing.T) {
	grantRepo := &testGrantRepo{}
	roleRepo := &testRoleRepo{}
	engine := NewPolicyEngine(roleRepo, grantRepo)

	userID := uuid.New()
	scope := Scope{Type: "team", ID: "123"}

	// Create a direct permission grant
	grant := &Grant{
		UserID:    userID,
		GrantType: GrantTypePermission,
		Value:     "users:read",
		Scope:     scope,
		Status:    authpkg.UserStatusActive,
	}
	grantRepo.Create(context.Background(), grant)

	hasPermission, err := engine.Has(context.Background(), userID, "users:read", scope)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !hasPermission {
		t.Error("Expected to have permission")
	}
}

func TestPolicyEngineHasRoleBasedPermission(t *testing.T) {
	grantRepo := &testGrantRepo{}
	roleRepo := &testRoleRepo{}
	engine := NewPolicyEngine(roleRepo, grantRepo)

	userID := uuid.New()
	roleID := uuid.New()
	scope := Scope{Type: "team", ID: "123"}

	// Create a role with permissions
	role := &Role{
		ID:          roleID,
		Name:        "admin",
		Permissions: []string{"users:read", "users:write", "users:delete"},
		Status:      authpkg.UserStatusActive,
	}
	roleRepo.Create(context.Background(), role)

	// Create a role grant
	grant := &Grant{
		UserID:    userID,
		GrantType: GrantTypeRole,
		Value:     roleID.String(),
		Scope:     scope,
		Status:    authpkg.UserStatusActive,
	}
	grantRepo.Create(context.Background(), grant)

	hasPermission, err := engine.Has(context.Background(), userID, "users:write", scope)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !hasPermission {
		t.Error("Expected to have permission via role")
	}
}

func TestPolicyEngineHasNoPermission(t *testing.T) {
	grantRepo := &testGrantRepo{}
	roleRepo := &testRoleRepo{}
	engine := NewPolicyEngine(roleRepo, grantRepo)

	userID := uuid.New()
	scope := Scope{Type: "team", ID: "123"}

	// No grants for this user
	hasPermission, err := engine.Has(context.Background(), userID, "users:delete", scope)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if hasPermission {
		t.Error("Expected not to have permission")
	}
}

func TestPolicyEngineHasExpiredGrant(t *testing.T) {
	grantRepo := &testGrantRepo{}
	roleRepo := &testRoleRepo{}
	engine := NewPolicyEngine(roleRepo, grantRepo)

	userID := uuid.New()
	scope := Scope{Type: "team", ID: "123"}
	expiredTime := time.Now().Add(-time.Hour)

	// Create an expired grant
	grant := &Grant{
		UserID:    userID,
		GrantType: GrantTypePermission,
		Value:     "users:read",
		Scope:     scope,
		ExpiresAt: &expiredTime,
		Status:    authpkg.UserStatusActive,
	}
	grantRepo.Create(context.Background(), grant)

	hasPermission, err := engine.Has(context.Background(), userID, "users:read", scope)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if hasPermission {
		t.Error("Expired grant should not give permission")
	}
}

func TestPolicyEngineHasGlobalScope(t *testing.T) {
	grantRepo := &testGrantRepo{}
	roleRepo := &testRoleRepo{}
	engine := NewPolicyEngine(roleRepo, grantRepo)

	userID := uuid.New()
	globalScope := Scope{Type: "global", ID: ""}
	requestedScope := Scope{Type: "team", ID: "123"}

	// Create a global scope grant
	grant := &Grant{
		UserID:    userID,
		GrantType: GrantTypePermission,
		Value:     "users:read",
		Scope:     globalScope,
		Status:    authpkg.UserStatusActive,
	}
	grantRepo.Create(context.Background(), grant)

	hasPermission, err := engine.Has(context.Background(), userID, "users:read", requestedScope)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !hasPermission {
		t.Error("Global scope should match any scope")
	}
}

func TestPolicyEngineGetUserPermissions(t *testing.T) {
	grantRepo := &testGrantRepo{}
	roleRepo := &testRoleRepo{}
	engine := NewPolicyEngine(roleRepo, grantRepo)

	userID := uuid.New()
	roleID := uuid.New()
	scope := Scope{Type: "team", ID: "123"}

	// Create a role with permissions
	role := &Role{
		ID:          roleID,
		Name:        "editor",
		Permissions: []string{"posts:read", "posts:write"},
		Status:      authpkg.UserStatusActive,
	}
	roleRepo.Create(context.Background(), role)

	// Create mixed grants (direct permission + role)
	directGrant := &Grant{
		UserID:    userID,
		GrantType: GrantTypePermission,
		Value:     "custom:permission",
		Scope:     scope,
		Status:    authpkg.UserStatusActive,
	}
	grantRepo.Create(context.Background(), directGrant)

	roleGrant := &Grant{
		UserID:    userID,
		GrantType: GrantTypeRole,
		Value:     roleID.String(),
		Scope:     scope,
		Status:    authpkg.UserStatusActive,
	}
	grantRepo.Create(context.Background(), roleGrant)

	permissions, err := engine.GetUserPermissions(context.Background(), userID, scope)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(permissions) != 3 {
		t.Errorf("Expected 3 permissions, got %d", len(permissions))
	}

	// Check that all expected permissions are present
	expectedPerms := map[string]bool{
		"custom:permission": false,
		"posts:read":        false,
		"posts:write":       false,
	}

	for _, perm := range permissions {
		if _, exists := expectedPerms[perm]; exists {
			expectedPerms[perm] = true
		} else {
			t.Errorf("Unexpected permission: %s", perm)
		}
	}

	for perm, found := range expectedPerms {
		if !found {
			t.Errorf("Missing expected permission: %s", perm)
		}
	}
}

func TestPolicyEngineFilterActiveGrants(t *testing.T) {
	grantRepo := &testGrantRepo{}
	roleRepo := &testRoleRepo{}
	engine := NewPolicyEngine(roleRepo, grantRepo)

	expiredTime := time.Now().Add(-time.Hour)
	futureTime := time.Now().Add(time.Hour)

	grants := []*Grant{
		{
			ID:     uuid.New(),
			Status: authpkg.UserStatusActive,
		},
		{
			ID:     uuid.New(),
			Status: authpkg.UserStatusSuspended,
		},
		{
			ID:        uuid.New(),
			Status:    authpkg.UserStatusActive,
			ExpiresAt: &expiredTime,
		},
		{
			ID:        uuid.New(),
			Status:    authpkg.UserStatusActive,
			ExpiresAt: &futureTime,
		},
	}

	activeGrants := engine.filterActiveGrants(grants)

	if len(activeGrants) != 2 {
		t.Errorf("Expected 2 active grants, got %d", len(activeGrants))
	}
}
