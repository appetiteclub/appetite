package authz

import (
	"testing"
	"time"

	"github.com/google/uuid"

	authpkg "github.com/appetiteclub/apt/auth"
)

func TestNewRole(t *testing.T) {
	role := NewRole()

	if role == nil {
		t.Error("NewRole() returned nil")
	}
	if role.Status != authpkg.UserStatusActive {
		t.Errorf("Expected status %v, got %v", authpkg.UserStatusActive, role.Status)
	}
}

func TestRoleEnsureID(t *testing.T) {
	tests := []struct {
		name  string
		role  *Role
		hasID bool
	}{
		{
			name:  "generates ID when nil",
			role:  &Role{ID: uuid.Nil},
			hasID: false,
		},
		{
			name:  "keeps existing ID",
			role:  &Role{ID: uuid.New()},
			hasID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.role.ID
			tt.role.EnsureID()

			if tt.hasID {
				if tt.role.ID != originalID {
					t.Error("should keep existing ID")
				}
			} else {
				if tt.role.ID == uuid.Nil {
					t.Error("should generate new ID")
				}
			}
		})
	}
}

func TestRoleBeforeCreate(t *testing.T) {
	role := &Role{
		ID:     uuid.Nil,
		Status: "",
	}

	beforeTime := time.Now()
	role.BeforeCreate()
	afterTime := time.Now()

	if role.ID == uuid.Nil {
		t.Error("Expected ID to be generated")
	}
	if role.Status != authpkg.UserStatusActive {
		t.Errorf("Expected status %v, got %v", authpkg.UserStatusActive, role.Status)
	}
	if role.CreatedAt.Before(beforeTime) || role.CreatedAt.After(afterTime) {
		t.Error("CreatedAt should be between before and after time")
	}
	if role.CreatedAt != role.UpdatedAt {
		t.Error("CreatedAt and UpdatedAt should be equal")
	}
}

func TestRoleBeforeUpdate(t *testing.T) {
	role := &Role{
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	beforeTime := time.Now()
	role.BeforeUpdate()
	afterTime := time.Now()

	if role.UpdatedAt.Before(beforeTime) || role.UpdatedAt.After(afterTime) {
		t.Error("UpdatedAt should be between before and after time")
	}
}

func TestRoleIsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   authpkg.UserStatus
		expected bool
	}{
		{
			name:     "active status",
			status:   authpkg.UserStatusActive,
			expected: true,
		},
		{
			name:     "deleted status",
			status:   authpkg.UserStatusDeleted,
			expected: false,
		},
		{
			name:     "suspended status",
			status:   authpkg.UserStatusSuspended,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := &Role{Status: tt.status}
			result := role.IsActive()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRoleHasPermission(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		permission  string
		expected    bool
	}{
		{
			name:        "has permission",
			permissions: []string{"read", "write", "delete"},
			permission:  "write",
			expected:    true,
		},
		{
			name:        "does not have permission",
			permissions: []string{"read", "write"},
			permission:  "delete",
			expected:    false,
		},
		{
			name:        "empty permissions",
			permissions: []string{},
			permission:  "read",
			expected:    false,
		},
		{
			name:        "case sensitive",
			permissions: []string{"READ", "WRITE"},
			permission:  "read",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := &Role{Permissions: tt.permissions}
			result := role.HasPermission(tt.permission)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRoleCompleteScenario(t *testing.T) {
	// Test a complete role creation and update scenario
	role := NewRole()
	role.Name = "admin"
	role.Permissions = []string{"users:read", "users:write", "roles:read"}

	// Test BeforeCreate
	role.BeforeCreate()
	if role.ID == uuid.Nil {
		t.Error("Expected ID to be generated")
	}
	if role.Status != authpkg.UserStatusActive {
		t.Errorf("Expected status %v, got %v", authpkg.UserStatusActive, role.Status)
	}
	if role.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	originalCreatedAt := role.CreatedAt
	originalID := role.ID

	// Test BeforeUpdate
	time.Sleep(1 * time.Millisecond) // Ensure time difference
	role.Permissions = append(role.Permissions, "roles:write")
	role.BeforeUpdate()

	if role.ID != originalID {
		t.Error("ID should not change on update")
	}
	if role.CreatedAt != originalCreatedAt {
		t.Error("CreatedAt should not change on update")
	}
	if !role.UpdatedAt.After(role.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}

	// Test permissions
	if !role.HasPermission("users:read") {
		t.Error("Role should have users:read permission")
	}
	if !role.HasPermission("roles:write") {
		t.Error("Role should have roles:write permission")
	}
	if role.HasPermission("admin:delete") {
		t.Error("Role should not have admin:delete permission")
	}

	// Test status
	if !role.IsActive() {
		t.Error("Role should be active")
	}

	role.Status = authpkg.UserStatusSuspended
	if role.IsActive() {
		t.Error("Role should not be active when status is suspended")
	}
}
