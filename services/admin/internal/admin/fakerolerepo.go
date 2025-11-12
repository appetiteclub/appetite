package admin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FakeRoleRepo provides an in-memory implementation of RoleRepo for development
type FakeRoleRepo struct {
	roles map[uuid.UUID]*Role
	mutex sync.RWMutex
}

// NewFakeRoleRepo creates a new fake role repository with some seed data
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
			Name:        "SuperAdmin",
			Description: "Full system access with all permissions",
			Permissions: []string{"system:admin"},
			Status:      "active",
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
			CreatedBy:   "system",
			UpdatedAt:   time.Now().Add(-30 * 24 * time.Hour),
			UpdatedBy:   "system",
		},
		{
			ID:          uuid.New(),
			Name:        "TeamAdmin",
			Description: "Team administration capabilities",
			Permissions: []string{
				"users:read",
				"users:write",
				"tables:read",
				"tables:write",
				"tables:manage",
				"grants:write",
			},
			Status:    "active",
			CreatedAt: time.Now().Add(-15 * 24 * time.Hour),
			CreatedBy: "system",
			UpdatedAt: time.Now().Add(-15 * 24 * time.Hour),
			UpdatedBy: "system",
		},
		{
			ID:          uuid.New(),
			Name:        "Viewer",
			Description: "Read-only access to resources",
			Permissions: []string{
				"users:read",
				"tables:read",
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

func (r *FakeRoleRepo) Create(ctx context.Context, req *CreateRoleRequest) (*Role, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, role := range r.roles {
		if role.Name == req.Name {
			return nil, fmt.Errorf("role with name %s already exists", req.Name)
		}
	}

	role := &Role{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
		Status:      "active",
		CreatedAt:   time.Now(),
		CreatedBy:   "admin",
		UpdatedAt:   time.Now(),
		UpdatedBy:   "admin",
	}

	r.roles[role.ID] = role
	return role, nil
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

func (r *FakeRoleRepo) Update(ctx context.Context, id uuid.UUID, req *UpdateRoleRequest) (*Role, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	role, exists := r.roles[id]
	if !exists {
		return nil, fmt.Errorf("role with id %s not found", id.String())
	}

	if req.Name != role.Name {
		for _, existingRole := range r.roles {
			if existingRole.ID != id && existingRole.Name == req.Name {
				return nil, fmt.Errorf("role with name %s already exists", req.Name)
			}
		}
	}

	role.Name = req.Name
	role.Description = req.Description
	role.Permissions = req.Permissions
	role.Status = req.Status
	role.UpdatedAt = time.Now()
	role.UpdatedBy = "admin"

	roleCopy := *role
	return &roleCopy, nil
}

func (r *FakeRoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, exists := r.roles[id]
	if !exists {
		return fmt.Errorf("role with id %s not found", id.String())
	}

	delete(r.roles, id)
	return nil
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
