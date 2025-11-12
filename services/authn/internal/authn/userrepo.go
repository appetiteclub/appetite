package authn

import (
	"context"

	"github.com/google/uuid"
)

// UserRepo defines the interface for User aggregate operations.
// This repository manages user data with proper encryption and security.
type UserRepo interface {
	// Create creates a new User aggregate.
	Create(ctx context.Context, user *User) error

	// Get retrieves a User by ID.
	Get(ctx context.Context, id uuid.UUID) (*User, error)

	// GetByEmailLookup retrieves a User by encrypted email lookup hash.
	// This is the primary method for login operations.
	GetByEmailLookup(ctx context.Context, lookup []byte) (*User, error)

	// GetByUsername retrieves a User by normalized username handle.
	GetByUsername(ctx context.Context, username string) (*User, error)

	// GetByPINLookup retrieves a User by PIN lookup hash.
	// This is used for lightweight authentication in conversational contexts.
	GetByPINLookup(ctx context.Context, lookup []byte) (*User, error)

	// Save updates an existing User aggregate.
	Save(ctx context.Context, user *User) error

	// Delete removes a User (soft delete by changing status).
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all Users (for admin operations).
	List(ctx context.Context) ([]*User, error)

	// ListByStatus retrieves Users filtered by status.
	ListByStatus(ctx context.Context, status string) ([]*User, error)
}
