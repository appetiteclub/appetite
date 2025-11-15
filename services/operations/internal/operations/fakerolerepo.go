package operations

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FakeRoleRepo provides an in-memory implementation of RoleRepo for testing
type FakeRoleRepo struct {
	roles map[uuid.UUID]*Role
	mutex sync.RWMutex
}

// NewFakeRoleRepo creates a new fake role repository with seed data
func NewFakeRoleRepo() *FakeRoleRepo {
	repo := &FakeRoleRepo{
		roles: make(map[uuid.UUID]*Role),
	}

	repo.seedRoles()
	return repo
}

func (r *FakeRoleRepo) seedRoles() {
	roles := []*Role{
		{
			ID:          uuid.New(),
			Name:        "superadmin",
			Description: "Full system access with all permissions",
			Permissions: []string{"*:*"},
			Status:      "active",
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
			CreatedBy:   "system",
			UpdatedAt:   time.Now().Add(-30 * 24 * time.Hour),
			UpdatedBy:   "system",
		},
		{
			ID:          uuid.New(),
			Name:        "conversational-interface-manager",
			Description: "Manages conversational interface and can delegate operations",
			Permissions: []string{
				"chat:access",
				"chat:delegate",
				"operations:read",
				"system:health",
			},
			Status:    "active",
			CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
			CreatedBy: "system",
			UpdatedAt: time.Now().Add(-30 * 24 * time.Hour),
			UpdatedBy: "system",
		},
		{
			ID:          uuid.New(),
			Name:        "admin",
			Description: "Administrative capabilities",
			Permissions: []string{
				"users:*",
				"roles:*",
				"grants:*",
				"tables:*",
				"orders:*",
			},
			Status:    "active",
			CreatedAt: time.Now().Add(-15 * 24 * time.Hour),
			CreatedBy: "system",
			UpdatedAt: time.Now().Add(-15 * 24 * time.Hour),
			UpdatedBy: "system",
		},
		{
			ID:          uuid.New(),
			Name:        "user",
			Description: "Basic user access",
			Permissions: []string{
				"users:read",
				"users:update",
			},
			Status:    "active",
			CreatedAt: time.Now().Add(-10 * 24 * time.Hour),
			CreatedBy: "system",
			UpdatedAt: time.Now().Add(-10 * 24 * time.Hour),
			UpdatedBy: "system",
		},
	}

	for _, role := range roles {
		r.roles[role.ID] = role
	}
}

func (r *FakeRoleRepo) Get(ctx context.Context, id uuid.UUID) (*Role, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	role, exists := r.roles[id]
	if !exists {
		return nil, fmt.Errorf("role with id %s not found", id.String())
	}

	roleCopy := *role
	return &roleCopy, nil
}

func (r *FakeRoleRepo) List(ctx context.Context) ([]*Role, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	roles := make([]*Role, 0, len(r.roles))
	for _, role := range r.roles {
		roleCopy := *role
		roles = append(roles, &roleCopy)
	}

	return roles, nil
}

func (r *FakeRoleRepo) ListByStatus(ctx context.Context, status string) ([]*Role, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	roles := make([]*Role, 0)
	for _, role := range r.roles {
		if role.Status == status {
			roleCopy := *role
			roles = append(roles, &roleCopy)
		}
	}

	return roles, nil
}
