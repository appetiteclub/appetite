package admin

import (
	"context"

	"github.com/google/uuid"
)

// GrantRepo defines the interface for grant management operations in admin
type GrantRepo interface {
	// Create creates a new grant
	Create(ctx context.Context, req *CreateGrantRequest) (*Grant, error)

	// Get retrieves a grant by ID
	Get(ctx context.Context, id uuid.UUID) (*Grant, error)

	// List retrieves all grants
	List(ctx context.Context) ([]*Grant, error)

	// ListByUser retrieves grants for a specific user
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Grant, error)

	// Delete removes a grant
	Delete(ctx context.Context, id uuid.UUID) error
}
