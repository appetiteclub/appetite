package menu

import (
	"context"

	"github.com/google/uuid"
)

// MenuItemRepo defines the repository interface for menu items
type MenuItemRepo interface {
	Create(ctx context.Context, item *MenuItem) error
	Get(ctx context.Context, id uuid.UUID) (*MenuItem, error)
	GetByShortCode(ctx context.Context, shortCode string) (*MenuItem, error)
	List(ctx context.Context) ([]*MenuItem, error)
	ListActive(ctx context.Context) ([]*MenuItem, error)
	ListByCategory(ctx context.Context, categoryID uuid.UUID) ([]*MenuItem, error)
	Save(ctx context.Context, item *MenuItem) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// MenuRepo defines the repository interface for menus
type MenuRepo interface {
	Create(ctx context.Context, menu *Menu) error
	Get(ctx context.Context, id uuid.UUID) (*Menu, error)
	List(ctx context.Context) ([]*Menu, error)
	ListPublished(ctx context.Context) ([]*Menu, error)
	Save(ctx context.Context, menu *Menu) error
	Delete(ctx context.Context, id uuid.UUID) error
}
