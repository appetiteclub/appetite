package media

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("media: not found")

type Repository interface {
	Create(ctx context.Context, media *Media) error
	Update(ctx context.Context, media *Media) error
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*Media, error)
	ListByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*Media, error)
	ListAll(ctx context.Context) ([]*Media, error)
}

type InMemoryRepository struct {
	mu    sync.RWMutex
	byID  map[uuid.UUID]*Media
	byRef map[string]map[uuid.UUID][]uuid.UUID
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		byID:  make(map[uuid.UUID]*Media),
		byRef: make(map[string]map[uuid.UUID][]uuid.UUID),
	}
}

func (r *InMemoryRepository) Create(_ context.Context, media *Media) error {
	if media == nil {
		return errors.New("media: cannot create nil entity")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byID[media.ID]; exists {
		return errors.New("media: id already exists")
	}

	clone := media.Clone()
	r.byID[clone.ID] = clone

	if _, ok := r.byRef[clone.TargetType]; !ok {
		r.byRef[clone.TargetType] = make(map[uuid.UUID][]uuid.UUID)
	}
	ids := append(r.byRef[clone.TargetType][clone.TargetID], clone.ID)
	r.byRef[clone.TargetType][clone.TargetID] = ids
	return nil
}

func (r *InMemoryRepository) Update(_ context.Context, media *Media) error {
	if media == nil {
		return errors.New("media: cannot update nil entity")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.byID[media.ID]
	if !ok {
		return ErrNotFound
	}

	// if resource changed, adjust index
	if existing.TargetType != media.TargetType || existing.TargetID != media.TargetID {
		ids := r.byRef[existing.TargetType][existing.TargetID]
		filtered := make([]uuid.UUID, 0, len(ids))
		for _, id := range ids {
			if id != media.ID {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) == 0 {
			delete(r.byRef[existing.TargetType], existing.TargetID)
		} else {
			r.byRef[existing.TargetType][existing.TargetID] = filtered
		}
		if _, ok := r.byRef[media.TargetType]; !ok {
			r.byRef[media.TargetType] = make(map[uuid.UUID][]uuid.UUID)
		}
		r.byRef[media.TargetType][media.TargetID] = append(r.byRef[media.TargetType][media.TargetID], media.ID)
	}

	r.byID[media.ID] = media.Clone()
	return nil
}

func (r *InMemoryRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.byID[id]
	if !ok {
		return ErrNotFound
	}
	delete(r.byID, id)

	ids := r.byRef[existing.TargetType][existing.TargetID]
	filtered := make([]uuid.UUID, 0, len(ids))
	for _, v := range ids {
		if v != id {
			filtered = append(filtered, v)
		}
	}
	if len(filtered) == 0 {
		delete(r.byRef[existing.TargetType], existing.TargetID)
	} else {
		r.byRef[existing.TargetType][existing.TargetID] = filtered
	}
	return nil
}

func (r *InMemoryRepository) Get(_ context.Context, id uuid.UUID) (*Media, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	media, ok := r.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return media.Clone(), nil
}

func (r *InMemoryRepository) ListByResource(_ context.Context, resourceType string, resourceID uuid.UUID) ([]*Media, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resMap, ok := r.byRef[resourceType]
	if !ok {
		return []*Media{}, nil
	}
	ids := resMap[resourceID]
	if len(ids) == 0 {
		return []*Media{}, nil
	}

	items := make([]*Media, 0, len(ids))
	for _, id := range ids {
		if media, ok := r.byID[id]; ok {
			items = append(items, media.Clone())
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	return items, nil
}

func (r *InMemoryRepository) ListAll(_ context.Context) ([]*Media, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]*Media, 0, len(r.byID))
	for _, media := range r.byID {
		items = append(items, media.Clone())
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	return items, nil
}
