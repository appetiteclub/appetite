package operations

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FakeGrantRepo provides an in-memory implementation of GrantRepo for testing
type FakeGrantRepo struct {
	grants map[uuid.UUID]*Grant
	mutex  sync.RWMutex
}

// NewFakeGrantRepo creates a new fake grant repository
func NewFakeGrantRepo() *FakeGrantRepo {
	return &FakeGrantRepo{
		grants: make(map[uuid.UUID]*Grant),
	}
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

// Create is a helper method for testing (not part of the interface)
func (r *FakeGrantRepo) Create(grant *Grant) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if grant.ID == uuid.Nil {
		grant.ID = uuid.New()
	}
	if grant.CreatedAt.IsZero() {
		grant.CreatedAt = time.Now()
	}
	if grant.UpdatedAt.IsZero() {
		grant.UpdatedAt = time.Now()
	}
	if grant.Status == "" {
		grant.Status = "active"
	}

	r.grants[grant.ID] = grant
	return nil
}
