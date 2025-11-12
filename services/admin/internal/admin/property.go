package admin

import (
	"time"

	"github.com/google/uuid"
)

// Property represents a real table property in the admin service.
// This is a DTO that mirrors the table service's Property aggregate.
type Property struct {
	ID             uuid.UUID      `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Classification Classification `json:"classification"`
	Location       Location       `json:"location"`
	Features       Features       `json:"features"`
	Prices         []Price        `json:"prices"`
	Status         string         `json:"status"`
	OwnerID        string         `json:"owner_id,omitempty"`
	SchemaVersion  int            `json:"schema_version"`
	CreatedAt      time.Time      `json:"created_at"`
	CreatedBy      string         `json:"created_by"`
	UpdatedAt      time.Time      `json:"updated_at"`
	UpdatedBy      string         `json:"updated_by"`
}

const CurrentPropertySchemaVersion = 3

// Classification represents the property taxonomy (references to fake service).
type Classification struct {
	CategoryID uuid.UUID `json:"category_id"`
	TypeID     uuid.UUID `json:"type_id"`
	SubtypeID  uuid.UUID `json:"subtype_id"`
}

// Location represents the physical location of a property.
type Location struct {
	Address     Address        `json:"address"`
	Coordinates Coordinates    `json:"coordinates"`
	Region      string         `json:"region,omitempty"`
	Provider    string         `json:"provider,omitempty"`
	ProviderURL string         `json:"provider_url,omitempty"`
	ProviderRef string         `json:"provider_ref,omitempty"`
	Raw         map[string]any `json:"raw,omitempty"`
	DisplayName string         `json:"display_name,omitempty"`
}

// Address represents a structured physical address.
type Address struct {
	Street     string `json:"street"`
	Number     string `json:"number,omitempty"`
	Unit       string `json:"unit,omitempty"`
	City       string `json:"city"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	Country    string `json:"country"`
}

// Coordinates represents geographic coordinates.
type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Features represents the physical characteristics and amenities.
type Features struct {
	TotalArea       float64  `json:"total_area"`
	CoveredArea     float64  `json:"covered_area"`
	LandArea        float64  `json:"land_area"`
	Bedrooms        int      `json:"bedrooms"`
	Bathrooms       int      `json:"bathrooms"`
	HalfBaths       int      `json:"half_baths"`
	Rooms           int      `json:"rooms"`
	Parking         int      `json:"parking"`
	CoveredParking  int      `json:"covered_parking"`
	Floors          int      `json:"floors"`
	Floor           int      `json:"floor"`
	YearBuilt       int      `json:"year_built"`
	Condition       string   `json:"condition"`
	Pool            bool     `json:"pool"`
	Garden          bool     `json:"garden"`
	Balcony         bool     `json:"balcony"`
	Terrace         bool     `json:"terrace"`
	Elevator        bool     `json:"elevator"`
	AirConditioning bool     `json:"air_conditioning"`
	Heating         bool     `json:"heating"`
	Furnished       bool     `json:"furnished"`
	PetFriendly     bool     `json:"pet_friendly"`
	Storage         bool     `json:"storage"`
	Laundry         bool     `json:"laundry"`
	Fireplace       bool     `json:"fireplace"`
	Amenities       []string `json:"amenities,omitempty"`
}

// Price represents pricing information.
type Price struct {
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Type       string  `json:"type"`
	Negotiable bool    `json:"negotiable"`
}

// CreatePropertyRequest represents a request to create a new property.
type CreatePropertyRequest struct {
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Classification Classification `json:"classification"`
	Location       Location       `json:"location"`
	Features       Features       `json:"features"`
	Prices         []Price        `json:"prices"`
	Status         string         `json:"status,omitempty"`
	OwnerID        string         `json:"owner_id,omitempty"`
	SchemaVersion  int            `json:"schema_version,omitempty"`
}

// UpdatePropertyRequest represents a request to update an existing property.
type UpdatePropertyRequest struct {
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Classification Classification `json:"classification"`
	Location       Location       `json:"location"`
	Features       Features       `json:"features"`
	Prices         []Price        `json:"prices"`
	Status         string         `json:"status"`
	OwnerID        string         `json:"owner_id,omitempty"`
	SchemaVersion  int            `json:"schema_version,omitempty"`
}
