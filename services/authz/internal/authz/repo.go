package authz

import (
	"context"

	"github.com/google/uuid"
)

// RoleRepo defines the repository interface for Role entities
type RoleRepo interface {
	Create(ctx context.Context, role *Role) error
	Get(ctx context.Context, id uuid.UUID) (*Role, error)
	GetByName(ctx context.Context, name string) (*Role, error)
	Save(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*Role, error)
	ListByStatus(ctx context.Context, status string) ([]*Role, error)
}

// GrantRepo defines the repository interface for Grant entities
type GrantRepo interface {
	Create(ctx context.Context, grant *Grant) error
	Get(ctx context.Context, id uuid.UUID) (*Grant, error)
	Save(ctx context.Context, grant *Grant) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*Grant, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*Grant, error)
	ListByScope(ctx context.Context, scope Scope) ([]*Grant, error)
	ListExpired(ctx context.Context) ([]*Grant, error)
}
