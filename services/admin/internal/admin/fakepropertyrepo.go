package admin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FakePropertyRepo provides an in-memory implementation of PropertyRepo for development.
type FakePropertyRepo struct {
	properties map[uuid.UUID]*Property
	mutex      sync.RWMutex
}

// NewFakePropertyRepo creates a new fake property repository with seed data.
func NewFakePropertyRepo() *FakePropertyRepo {
	repo := &FakePropertyRepo{
		properties: make(map[uuid.UUID]*Property),
	}
	repo.seedProperties()
	return repo
}

func (r *FakePropertyRepo) seedProperties() {
	// Hardcoded fake IDs from table fake fake
	residentialCategoryID := uuid.MustParse("00000000-0000-0000-0001-000000000001")
	commercialCategoryID := uuid.MustParse("00000000-0000-0000-0001-000000000002")
	houseTypeID := uuid.MustParse("00000000-0000-0000-0002-000000000001")
	apartmentTypeID := uuid.MustParse("00000000-0000-0000-0002-000000000002")
	officeTypeID := uuid.MustParse("00000000-0000-0000-0002-000000000003")
	bungalowSubtypeID := uuid.MustParse("00000000-0000-0000-0003-000000000001")
	loftSubtypeID := uuid.MustParse("00000000-0000-0000-0003-000000000002")

	properties := []*Property{
		{
			ID:          uuid.New(),
			Name:        "Modern Villa in Palermo",
			Description: "Beautiful modern villa with pool and garden in exclusive Palermo neighborhood",
			Classification: Classification{
				CategoryID: residentialCategoryID,
				TypeID:     houseTypeID,
				SubtypeID:  bungalowSubtypeID,
			},
			Location: Location{
				Address: Address{
					Street:     "Av. Santa Fe",
					Number:     "3500",
					City:       "Buenos Aires",
					State:      "CABA",
					PostalCode: "C1425",
					Country:    "Argentina",
				},
				Coordinates: Coordinates{
					Latitude:  -34.588,
					Longitude: -58.415,
				},
				Region: "EUROPE",
			},
			Features: Features{
				TotalArea:       450.0,
				CoveredArea:     350.0,
				LandArea:        500.0,
				Bedrooms:        4,
				Bathrooms:       3,
				HalfBaths:       1,
				Rooms:           8,
				Parking:         2,
				CoveredParking:  2,
				Floors:          2,
				YearBuilt:       2020,
				Condition:       "excellent",
				Pool:            true,
				Garden:          true,
				Balcony:         false,
				Terrace:         true,
				Elevator:        false,
				AirConditioning: true,
				Heating:         true,
				Furnished:       false,
				PetFriendly:     true,
				Storage:         true,
				Laundry:         true,
				Fireplace:       true,
				Amenities:       []string{"security", "grill"},
			},
			Prices: []Price{
				{
					Amount:     650000.0,
					Currency:   "USD",
					Type:       "sale",
					Negotiable: true,
				},
				{
					Amount:   4200.0,
					Currency: "USD",
					Type:     "rent_monthly",
				},
			},
			Status:    "available",
			OwnerID:   "agent-001",
			CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
			CreatedBy: "agent@appetite.com",
			UpdatedAt: time.Now().Add(-5 * 24 * time.Hour),
			UpdatedBy: "agent@appetite.com",
		},
		{
			ID:          uuid.New(),
			Name:        "Downtown Loft Apartment",
			Description: "Spacious loft in the heart of downtown with amazing city views",
			Classification: Classification{
				CategoryID: residentialCategoryID,
				TypeID:     apartmentTypeID,
				SubtypeID:  loftSubtypeID,
			},
			Location: Location{
				Address: Address{
					Street:     "Av. Corrientes",
					Number:     "1234",
					Unit:       "5B",
					City:       "Buenos Aires",
					State:      "CABA",
					PostalCode: "C1043",
					Country:    "Argentina",
				},
				Coordinates: Coordinates{
					Latitude:  -34.604,
					Longitude: -58.382,
				},
				Region: "EUROPE",
			},
			Features: Features{
				TotalArea:       120.0,
				CoveredArea:     120.0,
				LandArea:        0.0,
				Bedrooms:        2,
				Bathrooms:       2,
				HalfBaths:       0,
				Rooms:           4,
				Parking:         1,
				CoveredParking:  1,
				Floors:          1,
				Floor:           5,
				YearBuilt:       2018,
				Condition:       "good",
				Pool:            false,
				Garden:          false,
				Balcony:         true,
				Terrace:         false,
				Elevator:        true,
				AirConditioning: true,
				Heating:         true,
				Furnished:       true,
				PetFriendly:     false,
				Storage:         false,
				Laundry:         true,
				Fireplace:       false,
				Amenities:       []string{"gym", "concierge"},
			},
			Prices: []Price{
				{
					Amount:   1500.0,
					Currency: "USD",
					Type:     "rent_monthly",
				},
				{
					Amount:   120.0,
					Currency: "USD",
					Type:     "rent_daily",
				},
			},
			Status:    "available",
			OwnerID:   "agent-002",
			CreatedAt: time.Now().Add(-15 * 24 * time.Hour),
			CreatedBy: "agent2@appetite.com",
			UpdatedAt: time.Now().Add(-2 * 24 * time.Hour),
			UpdatedBy: "agent2@appetite.com",
		},
		{
			ID:          uuid.New(),
			Name:        "Premium Office Space - Microcentro",
			Description: "Modern office space in prime location, perfect for startups and small businesses",
			Classification: Classification{
				CategoryID: commercialCategoryID,
				TypeID:     officeTypeID,
				SubtypeID:  uuid.Nil,
			},
			Location: Location{
				Address: Address{
					Street:     "Av. 9 de Julio",
					Number:     "500",
					Unit:       "Piso 12",
					City:       "Buenos Aires",
					State:      "CABA",
					PostalCode: "C1065",
					Country:    "Argentina",
				},
				Coordinates: Coordinates{
					Latitude:  -34.608,
					Longitude: -58.384,
				},
				Region: "EUROPE",
			},
			Features: Features{
				TotalArea:       200.0,
				CoveredArea:     200.0,
				LandArea:        0.0,
				Bedrooms:        0,
				Bathrooms:       2,
				HalfBaths:       0,
				Rooms:           6,
				Parking:         3,
				CoveredParking:  3,
				Floors:          1,
				Floor:           12,
				YearBuilt:       2021,
				Condition:       "new",
				Pool:            false,
				Garden:          false,
				Balcony:         false,
				Terrace:         false,
				Elevator:        true,
				AirConditioning: true,
				Heating:         true,
				Furnished:       false,
				PetFriendly:     false,
				Storage:         true,
				Laundry:         false,
				Fireplace:       false,
				Amenities:       []string{"security", "reception", "meeting_rooms"},
			},
			Prices: []Price{
				{
					Amount:   650000.0,
					Currency: "USD",
					Type:     "sale",
				},
				{
					Amount:     3500.0,
					Currency:   "USD",
					Type:       "rent_monthly",
					Negotiable: true,
				},
			},
			Status:    "available",
			OwnerID:   "agent-001",
			CreatedAt: time.Now().Add(-7 * 24 * time.Hour),
			CreatedBy: "agent@appetite.com",
			UpdatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedBy: "agent@appetite.com",
		},
	}

	for _, prop := range properties {
		r.properties[prop.ID] = prop
	}
}

func (r *FakePropertyRepo) Create(ctx context.Context, req *CreatePropertyRequest) (*Property, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	property := &Property{
		ID:             uuid.New(),
		Name:           req.Name,
		Description:    req.Description,
		Classification: req.Classification,
		Location:       req.Location,
		Features:       req.Features,
		Prices:         req.Prices,
		Status:         req.Status,
		OwnerID:        req.OwnerID,
		CreatedAt:      time.Now(),
		CreatedBy:      "admin", // TODO: Get from context
		UpdatedAt:      time.Now(),
		UpdatedBy:      "admin",
	}

	if property.Status == "" {
		property.Status = "available"
	}

	r.properties[property.ID] = property
	return property, nil
}

func (r *FakePropertyRepo) Get(ctx context.Context, id uuid.UUID) (*Property, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	property, exists := r.properties[id]
	if !exists {
		return nil, fmt.Errorf("property with id %s not found", id.String())
	}

	return property, nil
}

func (r *FakePropertyRepo) List(ctx context.Context) ([]*Property, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	properties := make([]*Property, 0, len(r.properties))
	for _, prop := range r.properties {
		properties = append(properties, prop)
	}

	return properties, nil
}

func (r *FakePropertyRepo) Update(ctx context.Context, id uuid.UUID, req *UpdatePropertyRequest) (*Property, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	property, exists := r.properties[id]
	if !exists {
		return nil, fmt.Errorf("property with id %s not found", id.String())
	}

	property.Name = req.Name
	property.Description = req.Description
	property.Classification = req.Classification
	property.Location = req.Location
	property.Features = req.Features
	property.Prices = req.Prices
	property.Status = req.Status
	property.OwnerID = req.OwnerID
	property.UpdatedAt = time.Now()
	property.UpdatedBy = "admin" // TODO: Get from context

	return property, nil
}

func (r *FakePropertyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.properties[id]; !exists {
		return fmt.Errorf("property with id %s not found", id.String())
	}

	delete(r.properties, id)
	return nil
}

func (r *FakePropertyRepo) ListByOwner(ctx context.Context, ownerID string) ([]*Property, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	properties := make([]*Property, 0)
	for _, prop := range r.properties {
		if prop.OwnerID == ownerID {
			properties = append(properties, prop)
		}
	}

	return properties, nil
}

func (r *FakePropertyRepo) ListByStatus(ctx context.Context, status string) ([]*Property, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	properties := make([]*Property, 0)
	for _, prop := range r.properties {
		if prop.Status == status {
			properties = append(properties, prop)
		}
	}

	return properties, nil
}
