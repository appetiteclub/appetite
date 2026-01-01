package authz

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	authpkg "github.com/appetiteclub/apt/auth"
)


type roleSeedDocument struct {
	Roles []roleSeed `json:"roles"`
}

type roleSeed struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

func defaultRoleSeeds() []roleSeed {
	return []roleSeed{
		{
			Name: "superadmin",
			Permissions: []string{
				"*:*",
			},
		},
		{
			Name: "admin",
			Permissions: []string{
				"users:create",
				"users:read",
				"users:update",
				"users:delete",
				"users:list",
				"roles:create",
				"roles:read",
				"roles:update",
				"roles:delete",
				"roles:list",
				"grants:create",
				"grants:read",
				"grants:delete",
				"grants:list",
			},
		},
		{
			Name: "user",
			Permissions: []string{
				"users:read",
				"users:update",
			},
		},
	}
}

func loadRoleSeeds(seedFS embed.FS) ([]roleSeed, error) {
	seedBytes, err := seedFS.ReadFile("seed.json")
	if err != nil {
		return nil, fmt.Errorf("read seed.json: %w", err)
	}

	if len(seedBytes) == 0 {
		return nil, errors.New("bootstrap role seed file is empty")
	}

	var doc roleSeedDocument
	if err := json.Unmarshal(seedBytes, &doc); err != nil {
		return nil, fmt.Errorf("decode bootstrap role seeds: %w", err)
	}

	return doc.Roles, nil
}

func (s *BootstrapService) ensureRoleSeed(ctx context.Context, seed roleSeed) error {
	name := strings.TrimSpace(seed.Name)
	if name == "" {
		return nil
	}

	existing, err := s.roleRepo.GetByName(ctx, name)
	if err != nil {
		return fmt.Errorf("get role %s: %w", name, err)
	}

	if existing != nil {
		s.log().Info("Role already exists, skipping", "name", name)
		return nil
	}

	role := &Role{
		Name:        name,
		Permissions: seed.Permissions,
		Status:      authpkg.UserStatusActive,
		CreatedBy:   "seed:bootstrap",
		UpdatedBy:   "seed:bootstrap",
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return fmt.Errorf("create role %s: %w", name, err)
	}

	s.log().Info("Role created successfully", "name", name, "id", role.ID)
	return nil
}
