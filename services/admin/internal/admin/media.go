package admin

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Media represents a media asset retrieved from the media service.
type Media struct {
	ID                 uuid.UUID               `json:"id"`
	TargetType         string                  `json:"resource_type"`
	TargetID           uuid.UUID               `json:"resource_id"`
	StoragePath        string                  `json:"storage_path"`
	MimeType           string                  `json:"mime_type"`
	Resolution         MediaResolution         `json:"resolution"`
	Filesize           int64                   `json:"filesize"`
	Enabled            bool                    `json:"enabled"`
	Kind               string                  `json:"kind"`
	CategoryID         uuid.UUID               `json:"category_id"`
	Tags               []uuid.UUID             `json:"tags"`
	Metadata           map[string]any          `json:"metadata"`
	CroppingApplied    bool                    `json:"cropping_applied"`
	CompressionApplied bool                    `json:"compression_applied"`
	Variants           map[string]MediaVariant `json:"variants"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
}

// MediaResolution describes media dimensions.
type MediaResolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// MediaVariant represents a generated variant of the media asset.
type MediaVariant struct {
	Path     string `json:"path"`
	MimeType string `json:"mime_type"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Filesize int64  `json:"filesize"`
}

// MediaRepo defines the contract for interacting with the media service.
type MediaRepo interface {
	List(ctx context.Context, params MediaListParams) ([]*Media, error)
	ListByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]*Media, error)
	Get(ctx context.Context, id uuid.UUID) (*Media, error)
	Create(ctx context.Context, req CreateMediaRequest) (*Media, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateMediaRequest) (*Media, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Enable(ctx context.Context, id uuid.UUID) (*Media, error)
	Disable(ctx context.Context, id uuid.UUID) (*Media, error)
}

// CreateMediaRequest captures metadata and binary payload required to create a media asset.
type CreateMediaRequest struct {
	ResourceType string
	ResourceID   uuid.UUID
	FileName     string
	MimeType     string
	Data         []byte
	Filesize     int64
	CategoryID   uuid.UUID
	Kind         string
	Tags         []uuid.UUID
	Enabled      bool
	Resolution   MediaResolution
	Metadata     map[string]string
}

// MediaListParams captures filters supported by the media service list endpoint.
type MediaListParams struct {
	ResourceType string
	ResourceID   *uuid.UUID
}

// UpdateMediaRequest captures the mutable fields for a media asset.
type UpdateMediaRequest struct {
	MimeType    *string
	Resolution  *MediaResolution
	Filesize    *int64
	Kind        *string
	CategoryID  *uuid.UUID
	Tags        *[]uuid.UUID
	Metadata    map[string]string
	Enabled     *bool
	StoragePath *string
}

// MediaTagSet describes a dictionary-backed tag grouping for media assets.
type MediaTagSet struct {
	Name  string
	Label string
}

var defaultMediaTagSets = []MediaTagSet{
	{Name: "media_usage", Label: "Usage Tags"},
	{Name: "media_style", Label: "Style Tags"},
	{Name: "media_mood", Label: "Mood Tags"},
	{Name: "media_capture_conditions", Label: "Capture Conditions"},
	{Name: "media_feature_tags", Label: "Feature Tags"},
}

var defaultMediaTagSetNames = func() []string {
	names := make([]string, len(defaultMediaTagSets))
	for i, set := range defaultMediaTagSets {
		names[i] = set.Name
	}
	return names
}()

// DefaultMediaTagSets returns a copy of the configured media tag sets.
func DefaultMediaTagSets() []MediaTagSet {
	sets := make([]MediaTagSet, len(defaultMediaTagSets))
	copy(sets, defaultMediaTagSets)
	return sets
}

// DefaultMediaTagSetNames returns the available media tag set identifiers.
func DefaultMediaTagSetNames() []string {
	names := make([]string, len(defaultMediaTagSetNames))
	copy(names, defaultMediaTagSetNames)
	return names
}
