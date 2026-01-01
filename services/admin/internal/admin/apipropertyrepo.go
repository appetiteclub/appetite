package admin

import (
	"context"
	"fmt"

	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
)

// APIPropertyRepo implements PropertyRepo using ServiceClient to call table service.
type APIPropertyRepo struct {
	client *apt.ServiceClient
	logger apt.Logger
}

// NewAPIPropertyRepo creates a new API-based property repository.
// Returns an error if table service URL is not configured.
func NewAPIPropertyRepo(config *apt.Config, logger apt.Logger) (*APIPropertyRepo, error) {
	tableURL, _ := config.GetString("services.table.url")
	if tableURL == "" {
		return nil, fmt.Errorf("services.table.url is required")
	}

	client := apt.NewServiceClient(tableURL)
	if client == nil {
		return nil, fmt.Errorf("failed to create table service client")
	}

	return &APIPropertyRepo{
		client: client,
		logger: logger,
	}, nil
}

// List retrieves all properties from table service.
func (r *APIPropertyRepo) List(ctx context.Context) ([]*Property, error) {
	resp, err := r.client.List(ctx, "tables")
	if err != nil {
		return nil, fmt.Errorf("failed to list properties: %w", err)
	}

	// Handle null/empty data
	if resp.Data == nil {
		return []*Property{}, nil
	}

	propertiesData, ok := resp.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	properties := make([]*Property, 0, len(propertiesData))
	for _, item := range propertiesData {
		propertyData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		property, err := parsePropertyFromMap(propertyData)
		if err != nil {
			continue
		}

		properties = append(properties, property)
	}

	return properties, nil
}

// Get retrieves a property by ID from table service.
func (r *APIPropertyRepo) Get(ctx context.Context, id uuid.UUID) (*Property, error) {
	resp, err := r.client.Get(ctx, "tables", id.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get property: %w", err)
	}

	propertyData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return parsePropertyFromMap(propertyData)
}

// Create creates a new property via table service.
func (r *APIPropertyRepo) Create(ctx context.Context, req *CreatePropertyRequest) (*Property, error) {
	resp, err := r.client.Create(ctx, "tables", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create property: %w", err)
	}

	propertyData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return parsePropertyFromMap(propertyData)
}

// Update updates an existing property via table service.
func (r *APIPropertyRepo) Update(ctx context.Context, id uuid.UUID, req *UpdatePropertyRequest) (*Property, error) {
	resp, err := r.client.Update(ctx, "tables", id.String(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to update property: %w", err)
	}

	propertyData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return parsePropertyFromMap(propertyData)
}

// Delete removes a property via table service.
func (r *APIPropertyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.client.Delete(ctx, "tables", id.String()); err != nil {
		return fmt.Errorf("failed to delete property: %w", err)
	}

	return nil
}

// ListByOwner retrieves properties filtered by owner from table service.
func (r *APIPropertyRepo) ListByOwner(ctx context.Context, ownerID string) ([]*Property, error) {
	// Use the List method and filter client-side for now
	// TODO: Add query parameter support to ServiceClient
	allProperties, err := r.List(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*Property
	for _, prop := range allProperties {
		if prop.OwnerID == ownerID {
			filtered = append(filtered, prop)
		}
	}

	return filtered, nil
}

// ListByStatus retrieves properties filtered by status from table service.
func (r *APIPropertyRepo) ListByStatus(ctx context.Context, status string) ([]*Property, error) {
	// Use the List method and filter client-side for now
	// TODO: Add query parameter support to ServiceClient
	allProperties, err := r.List(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*Property
	for _, prop := range allProperties {
		if prop.Status == status {
			filtered = append(filtered, prop)
		}
	}

	return filtered, nil
}

// Helper functions

func parsePropertyFromMap(data map[string]interface{}) (*Property, error) {
	idStr := stringField(data, "id")
	if idStr == "" {
		return nil, fmt.Errorf("missing property id in response")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid property id: %w", err)
	}

	property := &Property{
		ID:            id,
		Name:          stringField(data, "name"),
		Description:   stringField(data, "description"),
		Status:        stringField(data, "status"),
		OwnerID:       stringField(data, "owner_id"),
		SchemaVersion: intField(data, "schema_version"),
	}

	// Parse classification
	if classData, ok := data["classification"].(map[string]interface{}); ok {
		categoryIDStr := stringField(classData, "category_id")
		typeIDStr := stringField(classData, "type_id")
		subtypeIDStr := stringField(classData, "subtype_id")

		if categoryIDStr != "" {
			property.Classification.CategoryID, _ = uuid.Parse(categoryIDStr)
		}
		if typeIDStr != "" {
			property.Classification.TypeID, _ = uuid.Parse(typeIDStr)
		}
		if subtypeIDStr != "" {
			property.Classification.SubtypeID, _ = uuid.Parse(subtypeIDStr)
		}
	}

	// Parse location
	if locData, ok := data["location"].(map[string]interface{}); ok {
		if addrData, ok := locData["address"].(map[string]interface{}); ok {
			property.Location.Address = Address{
				Street:     stringField(addrData, "street"),
				Number:     stringField(addrData, "number"),
				Unit:       stringField(addrData, "unit"),
				City:       stringField(addrData, "city"),
				State:      stringField(addrData, "state"),
				PostalCode: stringField(addrData, "postal_code"),
				Country:    stringField(addrData, "country"),
			}
		}

		if coordsData, ok := locData["coordinates"].(map[string]interface{}); ok {
			property.Location.Coordinates = Coordinates{
				Latitude:  floatField(coordsData, "latitude"),
				Longitude: floatField(coordsData, "longitude"),
			}
		}

		property.Location.Region = stringField(locData, "region")
		property.Location.Provider = stringField(locData, "provider")
		property.Location.ProviderURL = stringField(locData, "provider_url")
		property.Location.ProviderRef = stringField(locData, "provider_ref")
		property.Location.DisplayName = stringField(locData, "display_name")
		if raw, ok := locData["raw"].(map[string]interface{}); ok {
			property.Location.Raw = raw
		}
	}

	// Parse features
	if featData, ok := data["features"].(map[string]interface{}); ok {
		property.Features = Features{
			TotalArea:       floatField(featData, "total_area"),
			CoveredArea:     floatField(featData, "covered_area"),
			LandArea:        floatField(featData, "land_area"),
			Bedrooms:        intField(featData, "bedrooms"),
			Bathrooms:       intField(featData, "bathrooms"),
			HalfBaths:       intField(featData, "half_baths"),
			Rooms:           intField(featData, "rooms"),
			Parking:         intField(featData, "parking"),
			CoveredParking:  intField(featData, "covered_parking"),
			Floors:          intField(featData, "floors"),
			Floor:           intField(featData, "floor"),
			YearBuilt:       intField(featData, "year_built"),
			Condition:       stringField(featData, "condition"),
			Pool:            boolField(featData, "pool"),
			Garden:          boolField(featData, "garden"),
			Balcony:         boolField(featData, "balcony"),
			Terrace:         boolField(featData, "terrace"),
			Elevator:        boolField(featData, "elevator"),
			AirConditioning: boolField(featData, "air_conditioning"),
			Heating:         boolField(featData, "heating"),
			Furnished:       boolField(featData, "furnished"),
			PetFriendly:     boolField(featData, "pet_friendly"),
			Storage:         boolField(featData, "storage"),
			Laundry:         boolField(featData, "laundry"),
			Fireplace:       boolField(featData, "fireplace"),
		}
	}

	// Parse prices
	if pricesData, ok := data["prices"].([]interface{}); ok {
		property.Prices = make([]Price, 0, len(pricesData))
		for _, item := range pricesData {
			priceData, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			property.Prices = append(property.Prices, Price{
				Amount:     floatField(priceData, "amount"),
				Currency:   stringField(priceData, "currency"),
				Type:       stringField(priceData, "type"),
				Negotiable: boolField(priceData, "negotiable"),
			})
		}
	}

	return property, nil
}

func intField(data map[string]interface{}, key string) int {
	if v, ok := data[key].(float64); ok {
		return int(v)
	}
	return 0
}

func floatField(data map[string]interface{}, key string) float64 {
	if v, ok := data[key].(float64); ok {
		return v
	}
	return 0.0
}

func boolField(data map[string]interface{}, key string) bool {
	if v, ok := data[key].(bool); ok {
		return v
	}
	return false
}
