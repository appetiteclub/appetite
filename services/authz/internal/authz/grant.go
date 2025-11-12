package authz

import (
	"time"

	authpkg "github.com/aquamarinepk/aqm/auth"
	"github.com/google/uuid"
)

// GrantType represents the type of grant
type GrantType string

const (
	GrantTypeRole       GrantType = "role"
	GrantTypePermission GrantType = "permission"
)

// Scope represents the context scope for a grant
type Scope struct {
	Type string `json:"type"` // "global", "team", "organization"
	ID   string `json:"id"`   // Specific ID or empty for global
}

// Grant represents a permission grant to a user
type Grant struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	GrantType GrantType
	Value     string     // RoleID or PermissionCode
	Scope     Scope      // Context scope
	ExpiresAt *time.Time // Optional expiration
	Status    authpkg.UserStatus
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	UpdatedBy string
}

// NewGrant creates a new Grant
func NewGrant() *Grant {
	return &Grant{
		Status: authpkg.UserStatusActive,
		Scope:  Scope{Type: "global"},
	}
}

// EnsureID ensures the grant has an ID
func (g *Grant) EnsureID() {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
}

// BeforeCreate sets up the grant before creation
func (g *Grant) BeforeCreate() {
	g.EnsureID()
	now := time.Now()
	g.CreatedAt = now
	g.UpdatedAt = now
	if g.Status == "" {
		g.Status = authpkg.UserStatusActive
	}
}

// BeforeUpdate sets up the grant before update
func (g *Grant) BeforeUpdate() {
	g.UpdatedAt = time.Now()
}

// IsActive returns true if the grant is active and not expired
func (g *Grant) IsActive() bool {
	if g.Status != authpkg.UserStatusActive {
		return false
	}

	if g.ExpiresAt != nil && g.ExpiresAt.Before(time.Now()) {
		return false
	}

	return true
}

// IsExpired returns true if the grant has expired
func (g *Grant) IsExpired() bool {
	return g.ExpiresAt != nil && g.ExpiresAt.Before(time.Now())
}

// MatchesScope checks if the grant scope matches the requested scope
func (g *Grant) MatchesScope(requestedScope Scope) bool {
	// Global scope matches everything
	if g.Scope.Type == "global" {
		return true
	}

	// Exact match required for specific scopes
	return g.Scope.Type == requestedScope.Type && g.Scope.ID == requestedScope.ID
}
