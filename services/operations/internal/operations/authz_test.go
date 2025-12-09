package operations

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRoleStructFields(t *testing.T) {
	now := time.Now()
	roleID := uuid.New()

	role := Role{
		ID:          roleID,
		Name:        "admin",
		Description: "Administrator role",
		Permissions: []string{"read", "write", "delete"},
		Status:      "active",
		CreatedAt:   now,
		CreatedBy:   "system",
		UpdatedAt:   now,
		UpdatedBy:   "system",
	}

	if role.ID != roleID {
		t.Errorf("ID = %v, want %v", role.ID, roleID)
	}
	if role.Name != "admin" {
		t.Errorf("Name = %q, want %q", role.Name, "admin")
	}
	if role.Description != "Administrator role" {
		t.Errorf("Description = %q, want %q", role.Description, "Administrator role")
	}
	if len(role.Permissions) != 3 {
		t.Errorf("Permissions length = %d, want %d", len(role.Permissions), 3)
	}
	if role.Status != "active" {
		t.Errorf("Status = %q, want %q", role.Status, "active")
	}
	if role.CreatedBy != "system" {
		t.Errorf("CreatedBy = %q, want %q", role.CreatedBy, "system")
	}
	if role.UpdatedBy != "system" {
		t.Errorf("UpdatedBy = %q, want %q", role.UpdatedBy, "system")
	}
}

func TestGrantStructFields(t *testing.T) {
	now := time.Now()
	grantID := uuid.New()
	userID := uuid.New()
	expiresAt := now.Add(24 * time.Hour)

	grant := Grant{
		ID:        grantID,
		UserID:    userID,
		GrantType: "role",
		Value:     "admin-role-id",
		Scope: Scope{
			Type: "global",
			ID:   "",
		},
		ExpiresAt: &expiresAt,
		Status:    "active",
		CreatedAt: now,
		CreatedBy: "system",
		UpdatedAt: now,
		UpdatedBy: "system",
	}

	if grant.ID != grantID {
		t.Errorf("ID = %v, want %v", grant.ID, grantID)
	}
	if grant.UserID != userID {
		t.Errorf("UserID = %v, want %v", grant.UserID, userID)
	}
	if grant.GrantType != "role" {
		t.Errorf("GrantType = %q, want %q", grant.GrantType, "role")
	}
	if grant.Value != "admin-role-id" {
		t.Errorf("Value = %q, want %q", grant.Value, "admin-role-id")
	}
	if grant.Scope.Type != "global" {
		t.Errorf("Scope.Type = %q, want %q", grant.Scope.Type, "global")
	}
	if grant.ExpiresAt == nil {
		t.Error("ExpiresAt should not be nil")
	}
	if grant.Status != "active" {
		t.Errorf("Status = %q, want %q", grant.Status, "active")
	}
}

func TestGrantWithNilExpiration(t *testing.T) {
	grant := Grant{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		GrantType: "permission",
		Value:     "orders:write",
		Scope: Scope{
			Type: "property",
			ID:   "prop-123",
		},
		ExpiresAt: nil,
		Status:    "active",
	}

	if grant.ExpiresAt != nil {
		t.Error("ExpiresAt should be nil for permanent grants")
	}
	if grant.GrantType != "permission" {
		t.Errorf("GrantType = %q, want %q", grant.GrantType, "permission")
	}
	if grant.Scope.ID != "prop-123" {
		t.Errorf("Scope.ID = %q, want %q", grant.Scope.ID, "prop-123")
	}
}

func TestScopeFields(t *testing.T) {
	tests := []struct {
		name     string
		scope    Scope
		wantType string
		wantID   string
	}{
		{
			name:     "globalScope",
			scope:    Scope{Type: "global", ID: ""},
			wantType: "global",
			wantID:   "",
		},
		{
			name:     "propertyScope",
			scope:    Scope{Type: "property", ID: "prop-123"},
			wantType: "property",
			wantID:   "prop-123",
		},
		{
			name:     "venueScope",
			scope:    Scope{Type: "venue", ID: "venue-456"},
			wantType: "venue",
			wantID:   "venue-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.scope.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", tt.scope.Type, tt.wantType)
			}
			if tt.scope.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", tt.scope.ID, tt.wantID)
			}
		})
	}
}

func TestRoleWithEmptyPermissions(t *testing.T) {
	role := Role{
		ID:          uuid.New(),
		Name:        "guest",
		Permissions: []string{},
		Status:      "active",
	}

	if len(role.Permissions) != 0 {
		t.Errorf("Permissions length = %d, want 0", len(role.Permissions))
	}
}

func TestRoleWithNilPermissions(t *testing.T) {
	role := Role{
		ID:          uuid.New(),
		Name:        "guest",
		Permissions: nil,
		Status:      "active",
	}

	if role.Permissions != nil {
		t.Error("Permissions should be nil")
	}
}
