package dictionary

import (
	"context"

	"github.com/google/uuid"
)

// SetRepo defines the interface for Set aggregate operations.
type SetRepo interface {
	// Create creates a new Set aggregate.
	Create(ctx context.Context, set *Set) error

	// Get retrieves a complete Set aggregate by ID.
	Get(ctx context.Context, id uuid.UUID) (*Set, error)

	// GetByName retrieves a Set by its unique name (returns first match, use GetByNameAndLocale for specific locale).
	GetByName(ctx context.Context, name string) (*Set, error)

	// GetByNameAndLocale retrieves a Set by its unique name and locale.
	GetByNameAndLocale(ctx context.Context, name, locale string) (*Set, error)

	// Save performs a unit-of-work save operation on the aggregate.
	Save(ctx context.Context, set *Set) error

	// Delete removes the entire Set aggregate.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all Set aggregates.
	List(ctx context.Context) ([]*Set, error)

	// ListActive retrieves all active Set aggregates.
	ListActive(ctx context.Context) ([]*Set, error)
}

// OptionRepo defines the interface for Option aggregate operations.
type OptionRepo interface {
	// Create creates a new Option aggregate.
	Create(ctx context.Context, option *Option) error

	// Get retrieves a complete Option aggregate by ID.
	Get(ctx context.Context, id uuid.UUID) (*Option, error)

	// Save performs a unit-of-work save operation on the aggregate.
	Save(ctx context.Context, option *Option) error

	// Delete removes the entire Option aggregate.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all Option aggregates.
	List(ctx context.Context) ([]*Option, error)

	// ListBySet retrieves all options for a specific set.
	ListBySet(ctx context.Context, setID uuid.UUID) ([]*Option, error)

	// ListBySetName retrieves all options for a set by set name.
	ListBySetName(ctx context.Context, setName string) ([]*Option, error)

	// ListByParent retrieves all options with a specific parent ID.
	// If parentID is nil, returns root-level options (no parent).
	ListByParent(ctx context.Context, setID uuid.UUID, parentID *uuid.UUID) ([]*Option, error)

	// ListBySetAndParent retrieves all options in a set filtered by parent ID.
	ListBySetAndParent(ctx context.Context, setName string, parentID *uuid.UUID) ([]*Option, error)

	// ListActive retrieves all active Option aggregates.
	ListActive(ctx context.Context) ([]*Option, error)
}
