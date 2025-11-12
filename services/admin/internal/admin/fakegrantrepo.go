package admin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FakeGrantRepo provides an in-memory implementation of GrantRepo for development
type FakeGrantRepo struct {
	grants   map[uuid.UUID]*Grant
	mutex    sync.RWMutex
	userRepo UserRepo
	roleRepo RoleRepo
}

// NewFakeGrantRepo creates a new fake grant repository with some seed data
func NewFakeGrantRepo(userRepo UserRepo, roleRepo RoleRepo) *FakeGrantRepo {
	repo := &FakeGrantRepo{
		grants:   make(map[uuid.UUID]*Grant),
		userRepo: userRepo,
		roleRepo: roleRepo,
	}

	repo.seedGrants()
	return repo
}

func (r *FakeGrantRepo) seedGrants() {
	// Don't seed grants automatically - they should be created through UI
	// This avoids issues with role IDs changing on each startup
}

func (r *FakeGrantRepo) Create(ctx context.Context, req *CreateGrantRequest) (*Grant, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	grant := &Grant{
		ID:        uuid.New(),
		UserID:    req.UserID,
		GrantType: req.GrantType,
		Value:     req.Value,
		Scope:     req.Scope,
		ExpiresAt: req.ExpiresAt,
		Status:    "active",
		CreatedAt: time.Now(),
		CreatedBy: "admin",
		UpdatedAt: time.Now(),
		UpdatedBy: "admin",
	}

	r.grants[grant.ID] = grant
	return grant, nil
}

func (r *FakeGrantRepo) Get(ctx context.Context, id uuid.UUID) (*Grant, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	grant, exists := r.grants[id]
	if !exists {
		return nil, fmt.Errorf("grant with id %s not found", id.String())
	}

	grantCopy := *grant
	return &grantCopy, nil
}

func (r *FakeGrantRepo) List(ctx context.Context) ([]*Grant, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	grants := make([]*Grant, 0, len(r.grants))
	for _, grant := range r.grants {
		grantCopy := *grant
		grants = append(grants, &grantCopy)
	}

	return grants, nil
}

func (r *FakeGrantRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*Grant, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	grants := make([]*Grant, 0)
	for _, grant := range r.grants {
		if grant.UserID == userID && grant.Status == "active" {
			grantCopy := *grant
			grants = append(grants, &grantCopy)
		}
	}

	return grants, nil
}

func (r *FakeGrantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, exists := r.grants[id]
	if !exists {
		return fmt.Errorf("grant with id %s not found", id.String())
	}

	delete(r.grants, id)
	return nil
}
