package dictionary

import (
	"fmt"
	"time"

	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

// Option is the aggregate root for a fake option.
// An option represents a single entry within a set (e.g., "Residential", "House", "Bungalow").
// Options can have hierarchical relationships via ParentID.
type Option struct {
	ID          uuid.UUID  `json:"id" bson:"_id"`
	Set         uuid.UUID  `json:"set_id" bson:"set_id"`                               // Reference to the Set this option belongs to
	ParentID    *uuid.UUID `json:"parent_id,omitempty" bson:"parent_id,omitempty"`     // Optional parent option for hierarchy
	Locale      string     `json:"locale" bson:"locale"`                               // Language/locale code
	ShortCode   string     `json:"short_code" bson:"short_code"`                       // Short code for the option
	Key         string     `json:"key" bson:"key"`                                     // Unique key within the set per locale
	Label       string     `json:"label" bson:"label"`                                 // Human-readable label
	Description string     `json:"description,omitempty" bson:"description,omitempty"` // Optional description
	Value       string     `json:"value" bson:"value"`                                 // The actual value
	Order       int        `json:"order" bson:"order"`                                 // Display order
	Active      bool       `json:"active" bson:"active"`
	CreatedAt   time.Time  `json:"created_at" bson:"created_at"`
	CreatedBy   string     `json:"created_by" bson:"created_by"`
	UpdatedAt   time.Time  `json:"updated_at" bson:"updated_at"`
	UpdatedBy   string     `json:"updated_by" bson:"updated_by"`
}

// GetID returns the ID of the Option (implements Identifiable interface).
func (o *Option) GetID() uuid.UUID {
	return o.ID
}

// ResourceType returns the resource type for URL generation.
func (o *Option) ResourceType() string {
	return "dictionary/option"
}

// SetID sets the ID of the Option.
func (o *Option) SetID(id uuid.UUID) {
	o.ID = id
}

// New creates a new Option with a generated ID.
func NewOption() *Option {
	return &Option{
		ID:     apt.GenerateNewID(),
		Active: true,
		Order:  0,
	}
}

// EnsureID ensures the aggregate root has a valid ID.
func (o *Option) EnsureID() {
	if o.ID == uuid.Nil {
		o.ID = apt.GenerateNewID()
	}
}

// BeforeCreate sets creation timestamps.
func (o *Option) BeforeCreate() {
	o.EnsureID()
	o.CreatedAt = time.Now()
	o.UpdatedAt = time.Now()
	if !o.Active {
		o.Active = true
	}
}

// BeforeUpdate sets update timestamps.
func (o *Option) BeforeUpdate() {
	o.UpdatedAt = time.Now()
}

// MarshalBSON implements custom BSON marshaling for Option.
// It converts UUID fields to strings for MongoDB storage.
func (o *Option) MarshalBSON() ([]byte, error) {
	doc := bson.M{
		"_id":         o.ID.String(),
		"set_id":      o.Set.String(),
		"locale":      o.Locale,
		"short_code":  o.ShortCode,
		"key":         o.Key,
		"label":       o.Label,
		"description": o.Description,
		"value":       o.Value,
		"order":       o.Order,
		"active":      o.Active,
		"created_at":  o.CreatedAt,
		"created_by":  o.CreatedBy,
		"updated_at":  o.UpdatedAt,
		"updated_by":  o.UpdatedBy,
	}

	if o.ParentID != nil {
		doc["parent_id"] = o.ParentID.String()
	}

	return bson.Marshal(doc)
}

// UnmarshalBSON implements custom BSON unmarshaling for Option.
// It converts string IDs from MongoDB back to UUID.
func (o *Option) UnmarshalBSON(data []byte) error {
	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		return err
	}

	// Parse UUIDs from strings
	if idStr, ok := doc["_id"].(string); ok && idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return fmt.Errorf("invalid UUID format for _id: %w", err)
		}
		o.ID = id
	}

	if setIDStr, ok := doc["set_id"].(string); ok && setIDStr != "" {
		setID, err := uuid.Parse(setIDStr)
		if err != nil {
			return fmt.Errorf("invalid UUID format for set_id: %w", err)
		}
		o.Set = setID
	}

	if parentIDStr, ok := doc["parent_id"].(string); ok && parentIDStr != "" {
		parentID, err := uuid.Parse(parentIDStr)
		if err != nil {
			return fmt.Errorf("invalid UUID format for parent_id: %w", err)
		}
		o.ParentID = &parentID
	}

	// Map other fields
	if v, ok := doc["locale"].(string); ok {
		o.Locale = v
	}
	if v, ok := doc["short_code"].(string); ok {
		o.ShortCode = v
	}
	if v, ok := doc["key"].(string); ok {
		o.Key = v
	}
	if v, ok := doc["label"].(string); ok {
		o.Label = v
	}
	if v, ok := doc["description"].(string); ok {
		o.Description = v
	}
	if v, ok := doc["value"].(string); ok {
		o.Value = v
	}
	if v, ok := doc["order"].(int32); ok {
		o.Order = int(v)
	} else if v, ok := doc["order"].(int64); ok {
		o.Order = int(v)
	} else if v, ok := doc["order"].(int); ok {
		o.Order = v
	}
	if v, ok := doc["active"].(bool); ok {
		o.Active = v
	}
	if v, ok := doc["created_at"].(time.Time); ok {
		o.CreatedAt = v
	}
	if v, ok := doc["created_by"].(string); ok {
		o.CreatedBy = v
	}
	if v, ok := doc["updated_at"].(time.Time); ok {
		o.UpdatedAt = v
	}
	if v, ok := doc["updated_by"].(string); ok {
		o.UpdatedBy = v
	}

	return nil
}
