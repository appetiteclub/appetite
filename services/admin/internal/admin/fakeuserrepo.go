package admin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FakeUserRepo provides an in-memory implementation of UserRepo for development
type FakeUserRepo struct {
	users map[uuid.UUID]*User
	mutex sync.RWMutex
}

// NewFakeUserRepo creates a new fake user repository with some seed data
func NewFakeUserRepo() *FakeUserRepo {
	repo := &FakeUserRepo{
		users: make(map[uuid.UUID]*User),
	}

	// Add some seed users
	repo.seedUsers()
	return repo
}

func (r *FakeUserRepo) seedUsers() {
	users := []*User{
		{
			ID:        uuid.New(),
			Email:     "admin@appetite.com",
			Name:      "System Administrator",
			Username:  "system-admin",
			Status:    "active",
			CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
			CreatedBy: "system",
			UpdatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedBy: "system",
		},
		{
			ID:        uuid.New(),
			Email:     "john.doe@appetite.com",
			Name:      "John Doe",
			Username:  "john-doe",
			Status:    "active",
			CreatedAt: time.Now().Add(-7 * 24 * time.Hour),
			CreatedBy: "admin@appetite.com",
			UpdatedAt: time.Now().Add(-2 * 24 * time.Hour),
			UpdatedBy: "admin@appetite.com",
		},
		{
			ID:        uuid.New(),
			Email:     "jane.smith@appetite.com",
			Name:      "Jane Smith",
			Username:  "jane-smith",
			Status:    "pending",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			CreatedBy: "admin@appetite.com",
			UpdatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedBy: "admin@appetite.com",
		},
	}

	for _, user := range users {
		r.users[user.ID] = user
	}
}

func (r *FakeUserRepo) Create(ctx context.Context, req *CreateUserRequest) (*User, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if email/username already exist
	for _, user := range r.users {
		if user.Email == req.Email {
			return nil, fmt.Errorf("user with email %s already exists", req.Email)
		}
		if user.Username == req.Username {
			return nil, fmt.Errorf("user with username %s already exists", req.Username)
		}
	}

	user := &User{
		ID:        uuid.New(),
		Email:     req.Email,
		Name:      req.Name,
		Username:  req.Username,
		Status:    "active",
		CreatedAt: time.Now(),
		CreatedBy: "admin", // TODO: Get from context
		UpdatedAt: time.Now(),
		UpdatedBy: "admin",
	}

	r.users[user.ID] = user
	return user, nil
}

func (r *FakeUserRepo) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, fmt.Errorf("user with id %s not found", id.String())
	}

	// Return a copy to prevent external mutations
	userCopy := *user
	return &userCopy, nil
}

func (r *FakeUserRepo) List(ctx context.Context) ([]*User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	users := make([]*User, 0, len(r.users))
	for _, user := range r.users {
		userCopy := *user
		users = append(users, &userCopy)
	}

	return users, nil
}

func (r *FakeUserRepo) Update(ctx context.Context, id uuid.UUID, req *UpdateUserRequest) (*User, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	user, exists := r.users[id]
	if !exists {
		return nil, fmt.Errorf("user with id %s not found", id.String())
	}

	// Check if new email/username conflicts with existing users
	for _, existingUser := range r.users {
		if existingUser.ID == id {
			continue
		}
		if req.Email != user.Email && existingUser.Email == req.Email {
			return nil, fmt.Errorf("user with email %s already exists", req.Email)
		}
		if req.Username != user.Username && existingUser.Username == req.Username {
			return nil, fmt.Errorf("user with username %s already exists", req.Username)
		}
	}

	// Update fields
	user.Email = req.Email
	user.Name = req.Name
	user.Username = req.Username
	user.Status = req.Status
	user.UpdatedAt = time.Now()
	user.UpdatedBy = "admin" // TODO: Get from context

	userCopy := *user
	return &userCopy, nil
}

func (r *FakeUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, exists := r.users[id]
	if !exists {
		return fmt.Errorf("user with id %s not found", id.String())
	}

	delete(r.users, id)
	return nil
}

func (r *FakeUserRepo) ListByStatus(ctx context.Context, status string) ([]*User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	users := make([]*User, 0)
	for _, user := range r.users {
		if user.Status == status {
			userCopy := *user
			users = append(users, &userCopy)
		}
	}

	return users, nil
}
