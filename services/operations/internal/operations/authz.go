package operations

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Role represents a role in the authorization system
type Role struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   string    `json:"updated_by"`
}

// Grant represents a grant of a role or permission to a user
type Grant struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	GrantType string     `json:"grant_type"` // "role" or "permission"
	Value     string     `json:"value"`      // role ID or permission string
	Scope     Scope      `json:"scope"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	CreatedBy string     `json:"created_by"`
	UpdatedAt time.Time  `json:"updated_at"`
	UpdatedBy string     `json:"updated_by"`
}

// Scope represents the scope of a grant
type Scope struct {
	Type string `json:"type"` // "global", "property", etc.
	ID   string `json:"id"`   // property ID, etc.
}

// RoleRepo defines the interface for role operations
type RoleRepo interface {
	Get(ctx context.Context, id uuid.UUID) (*Role, error)
	List(ctx context.Context) ([]*Role, error)
	ListByStatus(ctx context.Context, status string) ([]*Role, error)
}

// GrantRepo defines the interface for grant operations
type GrantRepo interface {
	Get(ctx context.Context, id uuid.UUID) (*Grant, error)
	List(ctx context.Context) ([]*Grant, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Grant, error)
}
