package dictionary

import (
	"fmt"
	"time"

	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

// Set is the aggregate root for a fake set.
// A set is a container for related options (e.g., "table_category", "table_type").
type Set struct {
	ID          uuid.UUID `json:"id" bson:"_id"`
	Name        string    `json:"name" bson:"name"`     // Unique name per locale
	Locale      string    `json:"locale" bson:"locale"` // Language/locale code
	Label       string    `json:"label" bson:"label"`   // Human-readable label
	Description string    `json:"description,omitempty" bson:"description,omitempty"`
	Active      bool      `json:"active" bson:"active"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	CreatedBy   string    `json:"created_by" bson:"created_by"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
	UpdatedBy   string    `json:"updated_by" bson:"updated_by"`
}

// GetID returns the ID of the Set (implements Identifiable interface).
func (s *Set) GetID() uuid.UUID {
	return s.ID
}

// ResourceType returns the resource type for URL generation.
func (s *Set) ResourceType() string {
	return "dictionary/set"
}

// SetID sets the ID of the Set.
func (s *Set) SetID(id uuid.UUID) {
	s.ID = id
}

// New creates a new Set with a generated ID.
func NewSet() *Set {
	return &Set{
		ID:     apt.GenerateNewID(),
		Active: true,
	}
}

// EnsureID ensures the aggregate root has a valid ID.
func (s *Set) EnsureID() {
	if s.ID == uuid.Nil {
		s.ID = apt.GenerateNewID()
	}
}

// BeforeCreate sets creation timestamps.
func (s *Set) BeforeCreate() {
	s.EnsureID()
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	if !s.Active {
		s.Active = true
	}
}

// BeforeUpdate sets update timestamps.
func (s *Set) BeforeUpdate() {
	s.UpdatedAt = time.Now()
}

// MarshalBSON implements custom BSON marshaling for Set.
// It converts UUID fields to strings for MongoDB storage.
func (s *Set) MarshalBSON() ([]byte, error) {
	return bson.Marshal(bson.M{
		"_id":         s.ID.String(),
		"name":        s.Name,
		"locale":      s.Locale,
		"label":       s.Label,
		"description": s.Description,
		"active":      s.Active,
		"created_at":  s.CreatedAt,
		"created_by":  s.CreatedBy,
		"updated_at":  s.UpdatedAt,
		"updated_by":  s.UpdatedBy,
	})
}

// UnmarshalBSON implements custom BSON unmarshaling for Set.
// It converts string IDs from MongoDB back to UUID.
func (s *Set) UnmarshalBSON(data []byte) error {
	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		return err
	}

	// Parse UUID from string
	if idStr, ok := doc["_id"].(string); ok && idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return fmt.Errorf("invalid UUID format for _id: %w", err)
		}
		s.ID = id
	}

	// Map other fields
	if v, ok := doc["name"].(string); ok {
		s.Name = v
	}
	if v, ok := doc["locale"].(string); ok {
		s.Locale = v
	}
	if v, ok := doc["label"].(string); ok {
		s.Label = v
	}
	if v, ok := doc["description"].(string); ok {
		s.Description = v
	}
	if v, ok := doc["active"].(bool); ok {
		s.Active = v
	}
	if v, ok := doc["created_at"].(time.Time); ok {
		s.CreatedAt = v
	}
	if v, ok := doc["created_by"].(string); ok {
		s.CreatedBy = v
	}
	if v, ok := doc["updated_at"].(time.Time); ok {
		s.UpdatedAt = v
	}
	if v, ok := doc["updated_by"].(string); ok {
		s.UpdatedBy = v
	}

	return nil
}
