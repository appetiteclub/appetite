package media

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	KindReal        Kind = "real"
	KindDrawing     Kind = "drawing"
	KindAIGenerated Kind = "ai_generated"
)

type Resolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Variant struct {
	Path     string `json:"path"`
	MimeType string `json:"mime_type"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Filesize int64  `json:"filesize"`
}

type Media struct {
	ID                 uuid.UUID          `json:"id"`
	TargetType         string             `json:"resource_type"`
	TargetID           uuid.UUID          `json:"resource_id"`
	StoragePath        string             `json:"storage_path"`
	MimeType           string             `json:"mime_type"`
	Resolution         Resolution         `json:"resolution"`
	Filesize           int64              `json:"filesize"`
	Enabled            bool               `json:"enabled"`
	Kind               Kind               `json:"kind"`
	CategoryID         uuid.UUID          `json:"category_id"`
	Tags               []uuid.UUID        `json:"tags"`
	Metadata           map[string]any     `json:"metadata"`
	CroppingApplied    bool               `json:"cropping_applied"`
	CompressionApplied bool               `json:"compression_applied"`
	Variants           map[string]Variant `json:"variants"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

func (m *Media) GetID() uuid.UUID {
	return m.ID
}

func (m *Media) ResourceType() string {
	return "media"
}

func (m *Media) Clone() *Media {
	if m == nil {
		return nil
	}
	clone := *m
	if len(m.Tags) > 0 {
		clone.Tags = append([]uuid.UUID(nil), m.Tags...)
	}
	if len(m.Metadata) > 0 {
		clone.Metadata = make(map[string]any, len(m.Metadata))
		for k, v := range m.Metadata {
			clone.Metadata[k] = v
		}
	}
	if len(m.Variants) > 0 {
		clone.Variants = make(map[string]Variant, len(m.Variants))
		for k, v := range m.Variants {
			clone.Variants[k] = v
		}
	}
	return &clone
}

func (m *Media) Validate() []string {
	var errs []string
	if m.TargetType == "" {
		errs = append(errs, "resource_type is required")
	}
	if m.TargetID == uuid.Nil {
		errs = append(errs, "resource_id is required")
	}
	if strings.TrimSpace(m.StoragePath) == "" {
		errs = append(errs, "storage_path is required")
	}
	if strings.TrimSpace(m.MimeType) == "" {
		errs = append(errs, "mime_type is required")
	}
	if m.Resolution.Width <= 0 || m.Resolution.Height <= 0 {
		errs = append(errs, "resolution width and height must be positive")
	}
	if m.Filesize < 0 {
		errs = append(errs, "filesize cannot be negative")
	}
	if !m.Kind.Valid() {
		errs = append(errs, fmt.Sprintf("kind must be one of: %s", strings.Join(AllKinds(), ", ")))
	}
	if m.CategoryID == uuid.Nil {
		errs = append(errs, "category_id is required")
	}
	return errs
}

func (k Kind) Valid() bool {
	switch k {
	case KindReal, KindDrawing, KindAIGenerated:
		return true
	default:
		return false
	}
}

func AllKinds() []string {
	return []string{string(KindReal), string(KindDrawing), string(KindAIGenerated)}
}
