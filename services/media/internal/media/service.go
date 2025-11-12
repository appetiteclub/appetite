package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/appetiteclub/appetite/services/media/internal/dictionary"
	"github.com/appetiteclub/appetite/services/media/internal/storage"
)

type ServiceOptions struct {
	EnableCropping    bool
	EnableCompression bool
	Variants          []VariantDefinition
}

type VariantDefinition struct {
	Name   string
	Width  int
	Height int
}

type Service struct {
	repo  Repository
	store storage.MediaStorage
	dict  dictionary.Client
	opts  ServiceOptions
}

type CreateInput struct {
	ResourceType string
	ResourceID   uuid.UUID
	MimeType     string
	Resolution   Resolution
	Filesize     int64
	Kind         Kind
	CategoryID   uuid.UUID
	Tags         []uuid.UUID
	Metadata     map[string]any
	Enabled      bool
	StoragePath  string
	Variants     map[string]Variant
	Data         io.Reader
}

type UpdateInput struct {
	ID          uuid.UUID
	MimeType    *string
	Resolution  *Resolution
	Filesize    *int64
	Kind        *Kind
	CategoryID  *uuid.UUID
	Tags        *[]uuid.UUID
	Metadata    *map[string]any
	Enabled     *bool
	StoragePath *string
	Variants    *map[string]Variant
}

type ListFilter struct {
	ResourceType string
	ResourceID   uuid.UUID
}

func NewService(repo Repository, mediaStorage storage.MediaStorage, dict dictionary.Client, opts ServiceOptions) *Service {
	if dict == nil {
		dict = dictionary.NewNoopClient()
	}
	if mediaStorage == nil {
		mediaStorage = storage.NewNoopBackend()
	}
	return &Service{repo: repo, store: mediaStorage, dict: dict, opts: opts}
}

func (s *Service) CreateMedia(ctx context.Context, input CreateInput) (*Media, error) {
	if input.ResourceType == "" {
		return nil, errors.New("media: resource_type is required")
	}
	if input.ResourceID == uuid.Nil {
		return nil, errors.New("media: resource_id is required")
	}
	if input.CategoryID == uuid.Nil {
		return nil, errors.New("media: category_id is required")
	}
	if !input.Kind.Valid() {
		return nil, fmt.Errorf("media: invalid kind %q", input.Kind)
	}
	if err := s.dict.EnsureCategory(ctx, input.CategoryID); err != nil {
		return nil, fmt.Errorf("media: category validation failed: %w", err)
	}
	if len(input.Tags) > 0 {
		if err := s.dict.EnsureTags(ctx, input.Tags); err != nil {
			return nil, fmt.Errorf("media: tags validation failed: %w", err)
		}
	}

	storagePath := input.StoragePath
	if storagePath == "" {
		id := uuid.NewString()
		path, err := s.store.Save(ctx, id, input.Data, input.MimeType)
		if err != nil {
			return nil, fmt.Errorf("media: save storage failed: %w", err)
		}
		storagePath = path
	}

	now := time.Now().UTC()
	media := &Media{
		ID:                 uuid.New(),
		TargetType:         input.ResourceType,
		TargetID:           input.ResourceID,
		StoragePath:        storagePath,
		MimeType:           input.MimeType,
		Resolution:         input.Resolution,
		Filesize:           input.Filesize,
		Enabled:            input.Enabled,
		Kind:               input.Kind,
		CategoryID:         input.CategoryID,
		Tags:               append([]uuid.UUID(nil), input.Tags...),
		Metadata:           cloneMap(input.Metadata),
		CroppingApplied:    s.opts.EnableCropping,
		CompressionApplied: s.opts.EnableCompression,
		Variants:           cloneVariants(input.Variants),
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if errs := media.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("media: validation failed: %v", errs)
	}

	if err := s.repo.Create(ctx, media); err != nil {
		return nil, err
	}
	return media.Clone(), nil
}

func (s *Service) GetMedia(ctx context.Context, id uuid.UUID) (*Media, error) {
	media, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return media, nil
}

func (s *Service) ListMedia(ctx context.Context, filter ListFilter) ([]*Media, error) {
	if filter.ResourceType == "" && filter.ResourceID == uuid.Nil {
		return s.repo.ListAll(ctx)
	}
	if filter.ResourceType == "" || filter.ResourceID == uuid.Nil {
		return []*Media{}, nil
	}
	return s.repo.ListByResource(ctx, filter.ResourceType, filter.ResourceID)
}

func (s *Service) UpdateMedia(ctx context.Context, input UpdateInput) (*Media, error) {
	media, err := s.repo.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.MimeType != nil {
		media.MimeType = *input.MimeType
	}
	if input.Resolution != nil {
		media.Resolution = *input.Resolution
	}
	if input.Filesize != nil {
		media.Filesize = *input.Filesize
	}
	if input.Kind != nil {
		if !input.Kind.Valid() {
			return nil, fmt.Errorf("media: invalid kind %q", *input.Kind)
		}
		media.Kind = *input.Kind
	}
	if input.CategoryID != nil {
		if err := s.dict.EnsureCategory(ctx, *input.CategoryID); err != nil {
			return nil, fmt.Errorf("media: category validation failed: %w", err)
		}
		media.CategoryID = *input.CategoryID
	}
	if input.Tags != nil {
		if err := s.dict.EnsureTags(ctx, *input.Tags); err != nil {
			return nil, fmt.Errorf("media: tags validation failed: %w", err)
		}
		media.Tags = append([]uuid.UUID(nil), (*input.Tags)...)
	}
	if input.Metadata != nil {
		media.Metadata = cloneMap(*input.Metadata)
	}
	if input.Enabled != nil {
		media.Enabled = *input.Enabled
	}
	if input.StoragePath != nil && *input.StoragePath != "" {
		media.StoragePath = *input.StoragePath
	}
	if input.Variants != nil {
		media.Variants = cloneVariants(*input.Variants)
	}

	media.UpdatedAt = time.Now().UTC()

	if errs := media.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("media: validation failed: %v", errs)
	}

	if err := s.repo.Update(ctx, media); err != nil {
		return nil, err
	}
	return media.Clone(), nil
}

func (s *Service) DeleteMedia(ctx context.Context, id uuid.UUID) error {
	media, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if media.StoragePath != "" {
		if err := s.store.Delete(ctx, media.StoragePath); err != nil {
			return fmt.Errorf("media: storage delete failed: %w", err)
		}
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) SetMediaEnabled(ctx context.Context, id uuid.UUID, enabled bool) (*Media, error) {
	media, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	media.Enabled = enabled
	media.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, media); err != nil {
		return nil, err
	}
	return media.Clone(), nil
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneVariants(in map[string]Variant) map[string]Variant {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]Variant, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
