package admin

import (
	"context"

	"github.com/google/uuid"
)

// RoleRepo defines the interface for role management operations in admin
type RoleRepo interface {
	// Create creates a new role
	Create(ctx context.Context, req *CreateRoleRequest) (*Role, error)

	// Get retrieves a role by ID
	Get(ctx context.Context, id uuid.UUID) (*Role, error)

	// List retrieves all roles
	List(ctx context.Context) ([]*Role, error)

	// Update updates an existing role
	Update(ctx context.Context, id uuid.UUID, req *UpdateRoleRequest) (*Role, error)

	// Delete removes a role
	Delete(ctx context.Context, id uuid.UUID) error

	// ListByStatus retrieves roles filtered by status
	ListByStatus(ctx context.Context, status string) ([]*Role, error)
}
