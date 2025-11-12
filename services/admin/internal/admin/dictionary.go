package admin

import (
	"time"

	"github.com/google/uuid"
)

// DictionarySet represents a fake set.
type DictionarySet struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Locale      string    `json:"locale"`
	Label       string    `json:"label"`
	Description string    `json:"description,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DictionaryOptionDetail represents a detailed fake option with lookups.
type DictionaryOptionDetail struct {
	ID          uuid.UUID  `json:"id"`
	Set         uuid.UUID  `json:"set_id"`
	SetName     string     `json:"set_name"` // Populated from lookup
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	ParentLabel string     `json:"parent_label"` // Populated from lookup
	Locale      string     `json:"locale"`
	ShortCode   string     `json:"short_code"`
	Key         string     `json:"key"`
	Label       string     `json:"label"`
	Description string     `json:"description,omitempty"`
	Value       string     `json:"value"`
	Order       int        `json:"order"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateSetRequest represents a request to create a set.
type CreateSetRequest struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Active      bool   `json:"active"`
}

// UpdateSetRequest represents a request to update a set.
type UpdateSetRequest struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Active      bool   `json:"active"`
}

// CreateOptionRequest represents a request to create an option.
type CreateOptionRequest struct {
	Set         uuid.UUID  `json:"set_id"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	ShortCode   string     `json:"short_code"`
	Key         string     `json:"key"`
	Label       string     `json:"label"`
	Description string     `json:"description,omitempty"`
	Value       string     `json:"value"`
	Order       int        `json:"order"`
	Active      bool       `json:"active"`
}

// UpdateOptionRequest represents a request to update an option.
type UpdateOptionRequest struct {
	Set         uuid.UUID  `json:"set_id"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	ShortCode   string     `json:"short_code"`
	Key         string     `json:"key"`
	Label       string     `json:"label"`
	Description string     `json:"description,omitempty"`
	Value       string     `json:"value"`
	Order       int        `json:"order"`
	Active      bool       `json:"active"`
}
