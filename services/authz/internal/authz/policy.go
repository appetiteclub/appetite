package authz

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// PolicyEngine evaluates permissions based on grants and roles
type PolicyEngine struct {
	roleRepo  RoleRepo
	grantRepo GrantRepo
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine(roleRepo RoleRepo, grantRepo GrantRepo) *PolicyEngine {
	return &PolicyEngine{
		roleRepo:  roleRepo,
		grantRepo: grantRepo,
	}
}

// Has evaluates if a user has a specific permission in the given scope
func (p *PolicyEngine) Has(ctx context.Context, userID uuid.UUID, permission string, scope Scope) (bool, error) {
	// Get all active grants for the user
	grants, err := p.grantRepo.ListByUserID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("could not get user grants: %w", err)
	}

	// Filter active and non-expired grants
	activeGrants := p.filterActiveGrants(grants)

	// Check direct permission grants
	for _, grant := range activeGrants {
		if grant.GrantType == GrantTypePermission {
			if grant.Value == permission && grant.MatchesScope(scope) {
				return true, nil
			}
		}
	}

	// Check role-based permissions
	for _, grant := range activeGrants {
		if grant.GrantType == GrantTypeRole {
			if grant.MatchesScope(scope) {
				hasPermission, err := p.roleHasPermission(ctx, grant.Value, permission)
				if err != nil {
					return false, fmt.Errorf("error check role permission: %w", err)
				}
				if hasPermission {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// GetUserPermissions returns all permissions for a user in the given scope
func (p *PolicyEngine) GetUserPermissions(ctx context.Context, userID uuid.UUID, scope Scope) ([]string, error) {
	grants, err := p.grantRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("could not get user grants: %w", err)
	}

	activeGrants := p.filterActiveGrants(grants)
	permissions := make(map[string]bool) // Use map to avoid duplicates

	// Add direct permissions
	for _, grant := range activeGrants {
		if grant.GrantType == GrantTypePermission && grant.MatchesScope(scope) {
			permissions[grant.Value] = true
		}
	}

	// Add role-based permissions
	for _, grant := range activeGrants {
		if grant.GrantType == GrantTypeRole && grant.MatchesScope(scope) {
			rolePerms, err := p.getRolePermissions(ctx, grant.Value)
			if err != nil {
				return nil, fmt.Errorf("could not get role permissions: %w", err)
			}
			for _, perm := range rolePerms {
				permissions[perm] = true
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(permissions))
	for perm := range permissions {
		result = append(result, perm)
	}

	return result, nil
}

// filterActiveGrants filters grants that are active and not expired
func (p *PolicyEngine) filterActiveGrants(grants []*Grant) []*Grant {
	var active []*Grant
	for _, grant := range grants {
		if grant.IsActive() {
			active = append(active, grant)
		}
	}
	return active
}

// roleHasPermission checks if a role has a specific permission
func (p *PolicyEngine) roleHasPermission(ctx context.Context, roleID string, permission string) (bool, error) {
	id, err := uuid.Parse(roleID)
	if err != nil {
		return false, fmt.Errorf("invalid role ID: %w", err)
	}

	role, err := p.roleRepo.Get(ctx, id)
	if err != nil {
		return false, fmt.Errorf("could not get role: %w", err)
	}
	if role == nil {
		return false, nil
	}

	return role.HasPermission(permission), nil
}

// getRolePermissions gets all permissions for a role
func (p *PolicyEngine) getRolePermissions(ctx context.Context, roleID string) ([]string, error) {
	id, err := uuid.Parse(roleID)
	if err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}

	role, err := p.roleRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get role: %w", err)
	}
	if role == nil {
		return nil, nil
	}

	return role.Permissions, nil
}
