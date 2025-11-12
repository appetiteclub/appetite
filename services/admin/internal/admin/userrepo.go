package admin

import (
	"context"

	"github.com/google/uuid"
)

// UserRepo defines the interface for user management operations in admin
type UserRepo interface {
	// Create creates a new user
	Create(ctx context.Context, req *CreateUserRequest) (*User, error)

	// Get retrieves a user by ID
	Get(ctx context.Context, id uuid.UUID) (*User, error)

	// List retrieves all users
	List(ctx context.Context) ([]*User, error)

	// Update updates an existing user
	Update(ctx context.Context, id uuid.UUID, req *UpdateUserRequest) (*User, error)

	// Delete removes a user
	Delete(ctx context.Context, id uuid.UUID) error

	// ListByStatus retrieves users filtered by status
	ListByStatus(ctx context.Context, status string) ([]*User, error)
}
