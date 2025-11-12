package authz

import (
	"testing"
	"time"

	"github.com/google/uuid"

	authpkg "github.com/aquamarinepk/aqm/auth"
)

func TestNewGrant(t *testing.T) {
	grant := NewGrant()

	if grant == nil {
		t.Error("NewGrant() returned nil")
	}
	if grant.Status != authpkg.UserStatusActive {
		t.Errorf("Expected status %v, got %v", authpkg.UserStatusActive, grant.Status)
	}
	if grant.Scope.Type != "global" {
		t.Errorf("Expected scope type 'global', got '%s'", grant.Scope.Type)
	}
}

func TestGrantEnsureID(t *testing.T) {
	tests := []struct {
		name  string
		grant *Grant
		hasID bool
	}{
		{
			name:  "generates ID when nil",
			grant: &Grant{ID: uuid.Nil},
			hasID: false,
		},
		{
			name:  "keeps existing ID",
			grant: &Grant{ID: uuid.New()},
			hasID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.grant.ID
			tt.grant.EnsureID()

			if tt.hasID {
				if tt.grant.ID != originalID {
					t.Error("should keep existing ID")
				}
			} else {
				if tt.grant.ID == uuid.Nil {
					t.Error("should generate new ID")
				}
			}
		})
	}
}

func TestGrantBeforeCreate(t *testing.T) {
	grant := &Grant{
		ID:     uuid.Nil,
		Status: "",
	}

	beforeTime := time.Now()
	grant.BeforeCreate()
	afterTime := time.Now()

	if grant.ID == uuid.Nil {
		t.Error("Expected ID to be generated")
	}
	if grant.Status != authpkg.UserStatusActive {
		t.Errorf("Expected status %v, got %v", authpkg.UserStatusActive, grant.Status)
	}
	if grant.CreatedAt.Before(beforeTime) || grant.CreatedAt.After(afterTime) {
		t.Error("CreatedAt should be between before and after time")
	}
	if grant.CreatedAt != grant.UpdatedAt {
		t.Error("CreatedAt and UpdatedAt should be equal")
	}
}

func TestGrantBeforeUpdate(t *testing.T) {
	grant := &Grant{
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	beforeTime := time.Now()
	grant.BeforeUpdate()
	afterTime := time.Now()

	if grant.UpdatedAt.Before(beforeTime) || grant.UpdatedAt.After(afterTime) {
		t.Error("UpdatedAt should be between before and after time")
	}
}

func TestGrantIsActive(t *testing.T) {
	tests := []struct {
		name     string
		grant    *Grant
		expected bool
	}{
		{
			name: "active status and no expiration",
			grant: &Grant{
				Status:    authpkg.UserStatusActive,
				ExpiresAt: nil,
			},
			expected: true,
		},
		{
			name: "suspended status",
			grant: &Grant{
				Status:    authpkg.UserStatusSuspended,
				ExpiresAt: nil,
			},
			expected: false,
		},
		{
			name: "active status but expired",
			grant: &Grant{
				Status:    authpkg.UserStatusActive,
				ExpiresAt: timePtr(time.Now().Add(-time.Hour)),
			},
			expected: false,
		},
		{
			name: "active status and not expired",
			grant: &Grant{
				Status:    authpkg.UserStatusActive,
				ExpiresAt: timePtr(time.Now().Add(time.Hour)),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.grant.IsActive()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGrantIsExpired(t *testing.T) {
	tests := []struct {
		name     string
		grant    *Grant
		expected bool
	}{
		{
			name: "no expiration",
			grant: &Grant{
				ExpiresAt: nil,
			},
			expected: false,
		},
		{
			name: "expired",
			grant: &Grant{
				ExpiresAt: timePtr(time.Now().Add(-time.Hour)),
			},
			expected: true,
		},
		{
			name: "not expired",
			grant: &Grant{
				ExpiresAt: timePtr(time.Now().Add(time.Hour)),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.grant.IsExpired()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGrantMatchesScope(t *testing.T) {
	tests := []struct {
		name           string
		grantScope     Scope
		requestedScope Scope
		expected       bool
	}{
		{
			name:           "global scope matches everything",
			grantScope:     Scope{Type: "global", ID: ""},
			requestedScope: Scope{Type: "team", ID: "123"},
			expected:       true,
		},
		{
			name:           "exact scope match",
			grantScope:     Scope{Type: "team", ID: "123"},
			requestedScope: Scope{Type: "team", ID: "123"},
			expected:       true,
		},
		{
			name:           "scope type mismatch",
			grantScope:     Scope{Type: "team", ID: "123"},
			requestedScope: Scope{Type: "organization", ID: "123"},
			expected:       false,
		},
		{
			name:           "scope ID mismatch",
			grantScope:     Scope{Type: "team", ID: "123"},
			requestedScope: Scope{Type: "team", ID: "456"},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grant := &Grant{Scope: tt.grantScope}
			result := grant.MatchesScope(tt.requestedScope)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGrantTypeConstants(t *testing.T) {
	if GrantTypeRole != GrantType("role") {
		t.Errorf("Expected GrantTypeRole to be 'role', got %v", GrantTypeRole)
	}
	if GrantTypePermission != GrantType("permission") {
		t.Errorf("Expected GrantTypePermission to be 'permission', got %v", GrantTypePermission)
	}
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}
