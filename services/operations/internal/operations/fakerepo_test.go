package operations

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestNewFakeGrantRepo(t *testing.T) {
	repo := NewFakeGrantRepo()
	if repo == nil {
		t.Fatal("NewFakeGrantRepo() returned nil")
	}
	if repo.grants == nil {
		t.Error("grants map is nil")
	}
}

func TestFakeGrantRepoCreate(t *testing.T) {
	repo := NewFakeGrantRepo()

	grant := &Grant{
		UserID:    uuid.New(),
		GrantType: "role",
		Value:     uuid.New().String(),
	}

	err := repo.Create(grant)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if grant.ID == uuid.Nil {
		t.Error("ID was not set")
	}
	if grant.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if grant.Status != "active" {
		t.Errorf("Status = %q, want %q", grant.Status, "active")
	}
}

func TestFakeGrantRepoCreateWithExistingID(t *testing.T) {
	repo := NewFakeGrantRepo()

	existingID := uuid.New()
	grant := &Grant{
		ID:     existingID,
		UserID: uuid.New(),
		GrantType: "role", Value: uuid.New().String(),
		Status: "pending",
	}

	err := repo.Create(grant)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if grant.ID != existingID {
		t.Errorf("ID was changed, got %v, want %v", grant.ID, existingID)
	}
	if grant.Status != "pending" {
		t.Errorf("Status was changed, got %q, want %q", grant.Status, "pending")
	}
}

func TestFakeGrantRepoGet(t *testing.T) {
	repo := NewFakeGrantRepo()
	ctx := context.Background()

	grant := &Grant{
		UserID: uuid.New(),
		GrantType: "role", Value: uuid.New().String(),
	}
	repo.Create(grant)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existingGrant",
			id:      grant.ID,
			wantErr: false,
		},
		{
			name:    "nonExistentGrant",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Get(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("Get() returned nil for existing grant")
			}
		})
	}
}

func TestFakeGrantRepoList(t *testing.T) {
	repo := NewFakeGrantRepo()
	ctx := context.Background()

	// Initially empty
	grants, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(grants) != 0 {
		t.Errorf("List() on empty repo = %d, want 0", len(grants))
	}

	// Add grants
	for i := 0; i < 3; i++ {
		repo.Create(&Grant{
			UserID: uuid.New(),
			GrantType: "role", Value: uuid.New().String(),
		})
	}

	grants, err = repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(grants) != 3 {
		t.Errorf("List() = %d, want 3", len(grants))
	}
}

func TestFakeGrantRepoListByUser(t *testing.T) {
	repo := NewFakeGrantRepo()
	ctx := context.Background()

	userID := uuid.New()
	otherUserID := uuid.New()

	// Create grants for different users
	repo.Create(&Grant{UserID: userID, GrantType: "role", Value: uuid.New().String()})
	repo.Create(&Grant{UserID: userID, GrantType: "role", Value: uuid.New().String()})
	repo.Create(&Grant{UserID: otherUserID, GrantType: "role", Value: uuid.New().String()})

	grants, err := repo.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(grants) != 2 {
		t.Errorf("ListByUser() = %d, want 2", len(grants))
	}
}

func TestFakeGrantRepoListByUserInactive(t *testing.T) {
	repo := NewFakeGrantRepo()
	ctx := context.Background()

	userID := uuid.New()

	// Create active grant
	repo.Create(&Grant{UserID: userID, GrantType: "role", Value: uuid.New().String()})

	// Create inactive grant
	inactiveGrant := &Grant{UserID: userID, GrantType: "role", Value: uuid.New().String(), Status: "inactive"}
	repo.Create(inactiveGrant)

	grants, err := repo.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	// Only active grants should be returned
	if len(grants) != 1 {
		t.Errorf("ListByUser() = %d, want 1 (only active)", len(grants))
	}
}

func TestNewFakeRoleRepo(t *testing.T) {
	repo := NewFakeRoleRepo()
	if repo == nil {
		t.Fatal("NewFakeRoleRepo() returned nil")
	}
	if repo.roles == nil {
		t.Error("roles map is nil")
	}

	// Should have seeded roles
	ctx := context.Background()
	roles, _ := repo.List(ctx)
	if len(roles) == 0 {
		t.Error("repo should have seeded roles")
	}
}

func TestFakeRoleRepoGet(t *testing.T) {
	repo := NewFakeRoleRepo()
	ctx := context.Background()

	// Get an existing seeded role
	roles, _ := repo.List(ctx)
	if len(roles) == 0 {
		t.Fatal("no seeded roles")
	}

	existingID := roles[0].ID

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existingRole",
			id:      existingID,
			wantErr: false,
		},
		{
			name:    "nonExistentRole",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Get(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("Get() returned nil for existing role")
			}
		})
	}
}

func TestFakeRoleRepoList(t *testing.T) {
	repo := NewFakeRoleRepo()
	ctx := context.Background()

	roles, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	// Should have seeded roles (superadmin, admin, user, etc.)
	if len(roles) < 3 {
		t.Errorf("List() = %d, want at least 3 seeded roles", len(roles))
	}
}

func TestFakeRoleRepoListByStatus(t *testing.T) {
	repo := NewFakeRoleRepo()
	ctx := context.Background()

	tests := []struct {
		name      string
		status    string
		wantCount bool // true if we expect at least one result
	}{
		{
			name:      "activeRoles",
			status:    "active",
			wantCount: true,
		},
		{
			name:      "inactiveRoles",
			status:    "inactive",
			wantCount: false, // seeded roles are all active
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles, err := repo.ListByStatus(ctx, tt.status)
			if err != nil {
				t.Fatalf("ListByStatus() error = %v", err)
			}
			if tt.wantCount && len(roles) == 0 {
				t.Errorf("ListByStatus(%q) returned no roles, expected some", tt.status)
			}
			if !tt.wantCount && len(roles) > 0 {
				t.Errorf("ListByStatus(%q) returned %d roles, expected 0", tt.status, len(roles))
			}
		})
	}
}

func TestRoleFields(t *testing.T) {
	repo := NewFakeRoleRepo()
	ctx := context.Background()

	roles, _ := repo.List(ctx)

	// Find the superadmin role
	var superadmin *Role
	for _, role := range roles {
		if role.Name == "superadmin" {
			superadmin = role
			break
		}
	}

	if superadmin == nil {
		t.Fatal("superadmin role not found")
	}

	if superadmin.Name != "superadmin" {
		t.Errorf("Name = %q, want %q", superadmin.Name, "superadmin")
	}
	if superadmin.Status != "active" {
		t.Errorf("Status = %q, want %q", superadmin.Status, "active")
	}
	if len(superadmin.Permissions) == 0 {
		t.Error("Permissions should not be empty")
	}
}

func TestGrantFields(t *testing.T) {
	grant := &Grant{
		ID:     uuid.New(),
		UserID: uuid.New(),
		GrantType: "role", Value: uuid.New().String(),
		Status: "active",
	}

	if grant.ID == uuid.Nil {
		t.Error("ID should not be nil")
	}
	if grant.UserID == uuid.Nil {
		t.Error("UserID should not be nil")
	}
	if grant.Status != "active" {
		t.Errorf("Status = %q, want %q", grant.Status, "active")
	}
}
