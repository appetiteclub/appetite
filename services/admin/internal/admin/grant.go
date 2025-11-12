package admin

import (
	"time"

	"github.com/google/uuid"
)

// Grant represents a permission grant to a user in the admin interface
type Grant struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	GrantType string     `json:"grant_type"` // "role" or "permission"
	Value     string     `json:"value"`      // RoleID or PermissionCode
	Scope     Scope      `json:"scope"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	CreatedBy string     `json:"created_by"`
	UpdatedAt time.Time  `json:"updated_at"`
	UpdatedBy string     `json:"updated_by"`
}

// Scope represents the context scope for a grant
type Scope struct {
	Type string `json:"type"` // "global", "team", "organization"
	ID   string `json:"id"`   // Specific ID or empty for global
}

// CreateGrantRequest represents the request for creating a new grant
type CreateGrantRequest struct {
	UserID    uuid.UUID  `json:"user_id"`
	GrantType string     `json:"grant_type"`
	Value     string     `json:"value"`
	Scope     Scope      `json:"scope"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// GrantWithDetails extends Grant with resolved details for display
type GrantWithDetails struct {
	Grant
	RoleName       string   // If grant_type is "role"
	PermissionName string   // If grant_type is "permission"
	Permissions    []string // Effective permissions from role
}
