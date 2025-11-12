package admin

import (
	"context"

	"github.com/google/uuid"
)

// PropertyRepo defines the interface for property management operations in admin.
type PropertyRepo interface {
	// Create creates a new property
	Create(ctx context.Context, req *CreatePropertyRequest) (*Property, error)

	// Get retrieves a property by ID
	Get(ctx context.Context, id uuid.UUID) (*Property, error)

	// List retrieves all properties
	List(ctx context.Context) ([]*Property, error)

	// Update updates an existing property
	Update(ctx context.Context, id uuid.UUID, req *UpdatePropertyRequest) (*Property, error)

	// Delete removes a property
	Delete(ctx context.Context, id uuid.UUID) error

	// ListByOwner retrieves properties filtered by owner
	ListByOwner(ctx context.Context, ownerID string) ([]*Property, error)

	// ListByStatus retrieves properties filtered by status
	ListByStatus(ctx context.Context, status string) ([]*Property, error)
}
