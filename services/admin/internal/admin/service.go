package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
)

type Service interface {
	CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
	ListUsers(ctx context.Context) ([]*User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req *UpdateUserRequest) (*User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsersByStatus(ctx context.Context, status string) ([]*User, error)

	CreateRole(ctx context.Context, req *CreateRoleRequest) (*Role, error)
	GetRole(ctx context.Context, id uuid.UUID) (*Role, error)
	ListRoles(ctx context.Context) ([]*Role, error)
	UpdateRole(ctx context.Context, id uuid.UUID, req *UpdateRoleRequest) (*Role, error)
	DeleteRole(ctx context.Context, id uuid.UUID) error

	CreateGrant(ctx context.Context, req *CreateGrantRequest) (*Grant, error)
	GetGrant(ctx context.Context, id uuid.UUID) (*Grant, error)
	ListGrants(ctx context.Context) ([]*Grant, error)
	DeleteGrant(ctx context.Context, id uuid.UUID) error

	CreateProperty(ctx context.Context, req *CreatePropertyRequest) (*Property, error)
	GetProperty(ctx context.Context, id uuid.UUID) (*Property, error)
	ListProperties(ctx context.Context) ([]*Property, error)
	UpdateProperty(ctx context.Context, id uuid.UUID, req *UpdatePropertyRequest) (*Property, error)
	DeleteProperty(ctx context.Context, id uuid.UUID) error
	ListPropertiesByOwner(ctx context.Context, ownerID string) ([]*Property, error)
	ListPropertiesByStatus(ctx context.Context, status string) ([]*Property, error)
	SuggestLocations(ctx context.Context, query string) ([]LocationSuggestion, error)
	ResolveLocation(ctx context.Context, reference string) (*ResolvedAddress, error)
	NormalizeLocation(ctx context.Context, req NormalizeLocationRequest) (*NormalizedLocation, error)

	ListMediaByResource(ctx context.Context, targetType string, targetID uuid.UUID) ([]*Media, error)
	ListPropertyMedia(ctx context.Context, propertyID uuid.UUID) ([]*Media, error)
	CreatePropertyMedia(ctx context.Context, propertyID uuid.UUID, req CreateMediaRequest) (*Media, error)
	ListMedia(ctx context.Context, params MediaListParams) ([]*Media, error)
	GetMedia(ctx context.Context, id uuid.UUID) (*Media, error)
	UpdateMedia(ctx context.Context, id uuid.UUID, req UpdateMediaRequest) (*Media, error)
	DeleteMedia(ctx context.Context, id uuid.UUID) error
	EnableMedia(ctx context.Context, id uuid.UUID) (*Media, error)
	DisableMedia(ctx context.Context, id uuid.UUID) (*Media, error)
}

type defaultService struct {
	repos            Repos
	locationProvider LocationProvider
	logger           apt.Logger
	config           *apt.Config
}

type Repos struct {
	UserRepo     UserRepo
	RoleRepo     RoleRepo
	GrantRepo    GrantRepo
	PropertyRepo PropertyRepo
	MediaRepo    MediaRepo
}

func NewDefaultService(repos Repos, locationProvider LocationProvider, config *apt.Config, logger apt.Logger) *defaultService {
	return &defaultService{
		repos:            repos,
		locationProvider: locationProvider,
		logger:           logger,
		config:           config,
	}
}

var ErrLocationProviderUnavailable = errors.New("location provider not configured")

func (s *defaultService) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {

	return s.repos.UserRepo.Create(ctx, req)
}

func (s *defaultService) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repos.UserRepo.Get(ctx, id)
}

func (s *defaultService) ListUsers(ctx context.Context) ([]*User, error) {
	return s.repos.UserRepo.List(ctx)
}

func (s *defaultService) UpdateUser(ctx context.Context, id uuid.UUID, req *UpdateUserRequest) (*User, error) {
	return s.repos.UserRepo.Update(ctx, id, req)
}

func (s *defaultService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.repos.UserRepo.Delete(ctx, id)
}

func (s *defaultService) ListUsersByStatus(ctx context.Context, status string) ([]*User, error) {
	return s.repos.UserRepo.ListByStatus(ctx, status)
}

func (s *defaultService) CreateRole(ctx context.Context, req *CreateRoleRequest) (*Role, error) {
	return s.repos.RoleRepo.Create(ctx, req)
}

func (s *defaultService) GetRole(ctx context.Context, id uuid.UUID) (*Role, error) {
	return s.repos.RoleRepo.Get(ctx, id)
}

func (s *defaultService) ListRoles(ctx context.Context) ([]*Role, error) {
	return s.repos.RoleRepo.List(ctx)
}

func (s *defaultService) UpdateRole(ctx context.Context, id uuid.UUID, req *UpdateRoleRequest) (*Role, error) {
	return s.repos.RoleRepo.Update(ctx, id, req)
}

func (s *defaultService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return s.repos.RoleRepo.Delete(ctx, id)
}

func (s *defaultService) CreateGrant(ctx context.Context, req *CreateGrantRequest) (*Grant, error) {
	// Validar usuario
	user, err := s.repos.UserRepo.Get(ctx, req.UserID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}
	// Validar rol
	if strings.EqualFold(req.GrantType, "role") {
		roleID, err := uuid.Parse(req.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid role id")
		}

		role, err := s.repos.RoleRepo.Get(ctx, roleID)
		if err != nil || role == nil {
			return nil, fmt.Errorf("role not found")
		}
	}
	return s.repos.GrantRepo.Create(ctx, req)
}

func (s *defaultService) GetGrant(ctx context.Context, id uuid.UUID) (*Grant, error) {
	return s.repos.GrantRepo.Get(ctx, id)
}

func (s *defaultService) ListGrants(ctx context.Context) ([]*Grant, error) {
	return s.repos.GrantRepo.List(ctx)
}

func (s *defaultService) DeleteGrant(ctx context.Context, id uuid.UUID) error {
	return s.repos.GrantRepo.Delete(ctx, id)
}

// Property methods

func (s *defaultService) CreateProperty(ctx context.Context, req *CreatePropertyRequest) (*Property, error) {
	return s.repos.PropertyRepo.Create(ctx, req)
}

func (s *defaultService) GetProperty(ctx context.Context, id uuid.UUID) (*Property, error) {
	return s.repos.PropertyRepo.Get(ctx, id)
}

func (s *defaultService) ListProperties(ctx context.Context) ([]*Property, error) {
	return s.repos.PropertyRepo.List(ctx)
}

func (s *defaultService) UpdateProperty(ctx context.Context, id uuid.UUID, req *UpdatePropertyRequest) (*Property, error) {
	return s.repos.PropertyRepo.Update(ctx, id, req)
}

func (s *defaultService) DeleteProperty(ctx context.Context, id uuid.UUID) error {
	return s.repos.PropertyRepo.Delete(ctx, id)
}

func (s *defaultService) ListPropertiesByOwner(ctx context.Context, ownerID string) ([]*Property, error) {
	return s.repos.PropertyRepo.ListByOwner(ctx, ownerID)
}

func (s *defaultService) ListPropertiesByStatus(ctx context.Context, status string) ([]*Property, error) {
	return s.repos.PropertyRepo.ListByStatus(ctx, status)
}

func (s *defaultService) SuggestLocations(ctx context.Context, query string) ([]LocationSuggestion, error) {
	if s.locationProvider == nil {
		return nil, ErrLocationProviderUnavailable
	}
	return s.locationProvider.Autocomplete(ctx, query)
}

func (s *defaultService) ResolveLocation(ctx context.Context, reference string) (*ResolvedAddress, error) {
	if s.locationProvider == nil {
		return nil, ErrLocationProviderUnavailable
	}
	return s.locationProvider.Resolve(ctx, reference)
}

func (s *defaultService) NormalizeLocation(ctx context.Context, req NormalizeLocationRequest) (*NormalizedLocation, error) {
	if s.locationProvider == nil {
		return nil, ErrLocationProviderUnavailable
	}
	ref := strings.TrimSpace(req.ProviderRef)
	if ref == "" {
		return nil, fmt.Errorf("reference cannot be empty")
	}
	resolved, err := s.locationProvider.Resolve(ctx, ref)
	if err != nil {
		return nil, err
	}
	result := buildNormalizedLocation(resolved, req.SelectedText)
	return &result, nil
}

func (s *defaultService) ListMediaByResource(ctx context.Context, targetType string, targetID uuid.UUID) ([]*Media, error) {
	if s.repos.MediaRepo == nil {
		return []*Media{}, nil
	}
	return s.repos.MediaRepo.ListByTarget(ctx, targetType, targetID)
}

func (s *defaultService) ListPropertyMedia(ctx context.Context, propertyID uuid.UUID) ([]*Media, error) {
	return s.ListMediaByResource(ctx, "property", propertyID)
}

func (s *defaultService) CreatePropertyMedia(ctx context.Context, propertyID uuid.UUID, req CreateMediaRequest) (*Media, error) {
	if s.repos.MediaRepo == nil {
		return nil, fmt.Errorf("media repository not configured")
	}
	if req.ResourceType == "" {
		req.ResourceType = "property"
	}
	req.ResourceID = propertyID
	return s.repos.MediaRepo.Create(ctx, req)
}

func (s *defaultService) ListMedia(ctx context.Context, params MediaListParams) ([]*Media, error) {
	if s.repos.MediaRepo == nil {
		return []*Media{}, nil
	}
	return s.repos.MediaRepo.List(ctx, params)
}

func (s *defaultService) GetMedia(ctx context.Context, id uuid.UUID) (*Media, error) {
	if s.repos.MediaRepo == nil {
		return nil, fmt.Errorf("media repository not configured")
	}
	return s.repos.MediaRepo.Get(ctx, id)
}

func (s *defaultService) UpdateMedia(ctx context.Context, id uuid.UUID, req UpdateMediaRequest) (*Media, error) {
	if s.repos.MediaRepo == nil {
		return nil, fmt.Errorf("media repository not configured")
	}
	return s.repos.MediaRepo.Update(ctx, id, req)
}

func (s *defaultService) DeleteMedia(ctx context.Context, id uuid.UUID) error {
	if s.repos.MediaRepo == nil {
		return fmt.Errorf("media repository not configured")
	}
	return s.repos.MediaRepo.Delete(ctx, id)
}

func (s *defaultService) EnableMedia(ctx context.Context, id uuid.UUID) (*Media, error) {
	if s.repos.MediaRepo == nil {
		return nil, fmt.Errorf("media repository not configured")
	}
	return s.repos.MediaRepo.Enable(ctx, id)
}

func (s *defaultService) DisableMedia(ctx context.Context, id uuid.UUID) (*Media, error) {
	if s.repos.MediaRepo == nil {
		return nil, fmt.Errorf("media repository not configured")
	}
	return s.repos.MediaRepo.Disable(ctx, id)
}

func (s *defaultService) log() apt.Logger {
	return s.logger
}
