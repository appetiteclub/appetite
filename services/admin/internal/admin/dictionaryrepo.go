package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/aquamarinepk/aqm"
	"github.com/google/uuid"
)

// DictionaryRepo provides access to dictionary service for taxonomy data.
type DictionaryRepo interface {
	// Property classification helpers
	ListCategories(ctx context.Context) ([]DictionaryOption, error)
	ListTypesByCategory(ctx context.Context, categoryID uuid.UUID) ([]DictionaryOption, error)
	ListSubtypesByType(ctx context.Context, typeID uuid.UUID) ([]DictionaryOption, error)
	ListStatuses(ctx context.Context) ([]DictionaryOption, error)
	ListPriceTypes(ctx context.Context) ([]DictionaryOption, error)
	ListConditions(ctx context.Context) ([]DictionaryOption, error)
	GetOptionsBySetName(ctx context.Context, setName, locale string, parentID *uuid.UUID) ([]DictionaryOption, error)

	// Set CRUD operations
	ListSets(ctx context.Context) ([]DictionarySet, error)
	GetSet(ctx context.Context, id uuid.UUID) (*DictionarySet, error)
	CreateSet(ctx context.Context, req *CreateSetRequest) (*DictionarySet, error)
	UpdateSet(ctx context.Context, id uuid.UUID, req *UpdateSetRequest) (*DictionarySet, error)
	DeleteSet(ctx context.Context, id uuid.UUID) error

	// Option CRUD operations
	ListOptions(ctx context.Context, setID *uuid.UUID) ([]DictionaryOptionDetail, error)
	GetOption(ctx context.Context, id uuid.UUID) (*DictionaryOptionDetail, error)
	CreateOption(ctx context.Context, req *CreateOptionRequest) (*DictionaryOptionDetail, error)
	UpdateOption(ctx context.Context, id uuid.UUID, req *UpdateOptionRequest) (*DictionaryOptionDetail, error)
	DeleteOption(ctx context.Context, id uuid.UUID) error
}

// DictionaryOption represents a fake option.
type DictionaryOption struct {
	ID   uuid.UUID
	Name string
	Key  string
}

// FakeDictionaryRepo provides hardcoded fake data for development.
type FakeDictionaryRepo struct{}

// NewFakeDictionaryRepo creates a new fake dictionary repo.
func NewFakeDictionaryRepo() *FakeDictionaryRepo {
	return &FakeDictionaryRepo{}
}

// ListCategories returns all categories.
func (c *FakeDictionaryRepo) ListCategories(ctx context.Context) ([]DictionaryOption, error) {
	return []DictionaryOption{
		{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Name: "Residential"},
		{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), Name: "Commercial"},
		{ID: uuid.MustParse("33333333-3333-3333-3333-333333333333"), Name: "Industrial"},
		{ID: uuid.MustParse("44444444-4444-4444-4444-444444444444"), Name: "Land"},
		{ID: uuid.MustParse("55555555-5555-5555-5555-555555555555"), Name: "Special Purpose"},
		{ID: uuid.MustParse("66666666-6666-6666-6666-666666666666"), Name: "Mixed Use"},
	}, nil
}

// ListTypesByCategory returns types for a given category.
func (c *FakeDictionaryRepo) ListTypesByCategory(ctx context.Context, categoryID uuid.UUID) ([]DictionaryOption, error) {
	// Simplified: return all types for now
	// In real implementation, filter by category_id
	return []DictionaryOption{
		{ID: uuid.MustParse("a1111111-1111-1111-1111-111111111111"), Name: "House"},
		{ID: uuid.MustParse("a2222222-2222-2222-2222-222222222222"), Name: "Apartment"},
		{ID: uuid.MustParse("a3333333-3333-3333-3333-333333333333"), Name: "Office"},
		{ID: uuid.MustParse("a4444444-4444-4444-4444-444444444444"), Name: "Retail"},
	}, nil
}

// ListSubtypesByType returns subtypes for a given type.
func (c *FakeDictionaryRepo) ListSubtypesByType(ctx context.Context, typeID uuid.UUID) ([]DictionaryOption, error) {
	// Simplified: return all subtypes for now
	// In real implementation, filter by type_id
	return []DictionaryOption{
		{ID: uuid.MustParse("b1111111-1111-1111-1111-111111111111"), Name: "Bungalow"},
		{ID: uuid.MustParse("b2222222-2222-2222-2222-222222222222"), Name: "Studio"},
		{ID: uuid.MustParse("b3333333-3333-3333-3333-333333333333"), Name: "Loft"},
	}, nil
}

// ListStatuses returns all table status options.
func (c *FakeDictionaryRepo) ListStatuses(ctx context.Context) ([]DictionaryOption, error) {
	return []DictionaryOption{
		{ID: uuid.MustParse("c1111111-1111-1111-1111-111111111111"), Name: "Available"},
		{ID: uuid.MustParse("c2222222-2222-2222-2222-222222222222"), Name: "Reserved"},
		{ID: uuid.MustParse("c3333333-3333-3333-3333-333333333333"), Name: "Sold"},
	}, nil
}

// ListPriceTypes returns all price type options.
func (c *FakeDictionaryRepo) ListPriceTypes(ctx context.Context) ([]DictionaryOption, error) {
	return []DictionaryOption{
		{ID: uuid.MustParse("d1111111-1111-1111-1111-111111111111"), Name: "Sale", Key: "sale"},
		{ID: uuid.MustParse("d2222222-2222-2222-2222-222222222222"), Name: "Rent (Monthly)", Key: "rent_monthly"},
		{ID: uuid.MustParse("d3333333-3333-3333-3333-333333333333"), Name: "Rent (Weekly)", Key: "rent_weekly"},
		{ID: uuid.MustParse("d4444444-4444-4444-4444-444444444444"), Name: "Rent (Daily)", Key: "rent_daily"},
		{ID: uuid.MustParse("d5555555-5555-5555-5555-555555555555"), Name: "Rent (Yearly)", Key: "rent_yearly"},
	}, nil
}

// ListConditions returns all condition options.
func (c *FakeDictionaryRepo) ListConditions(ctx context.Context) ([]DictionaryOption, error) {
	return []DictionaryOption{
		{ID: uuid.MustParse("e1111111-1111-1111-1111-111111111111"), Name: "New"},
		{ID: uuid.MustParse("e2222222-2222-2222-2222-222222222222"), Name: "Excellent"},
		{ID: uuid.MustParse("e3333333-3333-3333-3333-333333333333"), Name: "Good"},
		{ID: uuid.MustParse("e4444444-4444-4444-4444-444444444444"), Name: "Fair"},
		{ID: uuid.MustParse("e5555555-5555-5555-5555-555555555555"), Name: "Poor"},
	}, nil
}

func (c *FakeDictionaryRepo) GetOptionsBySetName(ctx context.Context, setName, locale string, parentID *uuid.UUID) ([]DictionaryOption, error) {
	return []DictionaryOption{}, nil
}

// Set CRUD stub implementations for FakeDictionaryRepo
func (c *FakeDictionaryRepo) ListSets(ctx context.Context) ([]DictionarySet, error) {
	return []DictionarySet{}, nil
}

func (c *FakeDictionaryRepo) GetSet(ctx context.Context, id uuid.UUID) (*DictionarySet, error) {
	return nil, nil
}

func (c *FakeDictionaryRepo) CreateSet(ctx context.Context, req *CreateSetRequest) (*DictionarySet, error) {
	return nil, nil
}

func (c *FakeDictionaryRepo) UpdateSet(ctx context.Context, id uuid.UUID, req *UpdateSetRequest) (*DictionarySet, error) {
	return nil, nil
}

func (c *FakeDictionaryRepo) DeleteSet(ctx context.Context, id uuid.UUID) error {
	return nil
}

// Option CRUD stub implementations for FakeDictionaryRepo
func (c *FakeDictionaryRepo) ListOptions(ctx context.Context, setID *uuid.UUID) ([]DictionaryOptionDetail, error) {
	return []DictionaryOptionDetail{}, nil
}

func (c *FakeDictionaryRepo) GetOption(ctx context.Context, id uuid.UUID) (*DictionaryOptionDetail, error) {
	return nil, nil
}

func (c *FakeDictionaryRepo) CreateOption(ctx context.Context, req *CreateOptionRequest) (*DictionaryOptionDetail, error) {
	return nil, nil
}

func (c *FakeDictionaryRepo) UpdateOption(ctx context.Context, id uuid.UUID, req *UpdateOptionRequest) (*DictionaryOptionDetail, error) {
	return nil, nil
}

func (c *FakeDictionaryRepo) DeleteOption(ctx context.Context, id uuid.UUID) error {
	return nil
}

// APIDictionaryRepo calls the real dictionary service via API.
type APIDictionaryRepo struct {
	client *aqm.ServiceClient
	logger aqm.Logger
}

// NewAPIDictionaryRepo creates a new API-based dictionary repo.
// Returns an error if dictionary service URL is not configured.
func NewAPIDictionaryRepo(config *aqm.Config, logger aqm.Logger) (*APIDictionaryRepo, error) {
	dictURL, _ := config.GetString("services.dictionary.url")
	if dictURL == "" {
		return nil, fmt.Errorf("services.dictionary.url is required")
	}

	client := aqm.NewServiceClient(dictURL)
	if client == nil {
		return nil, fmt.Errorf("failed to create dictionary service client")
	}

	return &APIDictionaryRepo{
		client: client,
		logger: logger,
	}, nil
}

// GetOptionsBySetName retrieves all options for a given set name and locale.
// This is a helper method to load dictionary options from the dictionary service.
func (c *APIDictionaryRepo) GetOptionsBySetName(ctx context.Context, setName, locale string, parentID *uuid.UUID) ([]DictionaryOption, error) {
	// First, find the set by name and locale
	sets, err := c.ListSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sets: %w", err)
	}

	var targetSetID *uuid.UUID
	for _, set := range sets {
		if set.Name == setName && set.Locale == locale {
			targetSetID = &set.ID
			break
		}
	}

	if targetSetID == nil {
		return []DictionaryOption{}, nil // Set not found, return empty
	}

	// Get all options for this set
	allOptions, err := c.ListOptions(ctx, targetSetID)
	if err != nil {
		return nil, fmt.Errorf("failed to list options: %w", err)
	}

	// Filter by parentID if specified
	var filteredOptions []DictionaryOption
	for _, opt := range allOptions {
		// Check parent filter
		if parentID != nil {
			if opt.ParentID == nil || *opt.ParentID != *parentID {
				continue
			}
		} else {
			// If no parent filter specified, only include root options (no parent)
			if opt.ParentID != nil {
				continue
			}
		}

		filteredOptions = append(filteredOptions, DictionaryOption{
			ID:   opt.ID,
			Name: opt.Label,
			Key:  opt.Key,
		})
	}

	return filteredOptions, nil
}

// ListCategories returns all categories from dictionary service.
func (c *APIDictionaryRepo) ListCategories(ctx context.Context) ([]DictionaryOption, error) {
	return c.GetOptionsBySetName(ctx, "table_category", "en", nil)
}

// ListTypesByCategory returns types for a category from dictionary service.
func (c *APIDictionaryRepo) ListTypesByCategory(ctx context.Context, categoryID uuid.UUID) ([]DictionaryOption, error) {
	if categoryID == uuid.Nil {
		// Return all types (root level)
		return c.GetOptionsBySetName(ctx, "table_type", "en", nil)
	}
	// Return types filtered by category
	return c.GetOptionsBySetName(ctx, "table_type", "en", &categoryID)
}

// ListSubtypesByType returns subtypes for a type from dictionary service.
func (c *APIDictionaryRepo) ListSubtypesByType(ctx context.Context, typeID uuid.UUID) ([]DictionaryOption, error) {
	if typeID == uuid.Nil {
		// Return all subtypes (root level)
		return c.GetOptionsBySetName(ctx, "table_subtype", "en", nil)
	}
	// Return subtypes filtered by type
	return c.GetOptionsBySetName(ctx, "table_subtype", "en", &typeID)
}

// ListStatuses returns all table status options from dictionary service.
func (c *APIDictionaryRepo) ListStatuses(ctx context.Context) ([]DictionaryOption, error) {
	return c.GetOptionsBySetName(ctx, "table_status", "en", nil)
}

// ListPriceTypes returns all price type options from dictionary service.
func (c *APIDictionaryRepo) ListPriceTypes(ctx context.Context) ([]DictionaryOption, error) {
	return c.GetOptionsBySetName(ctx, "price_type", "en", nil)
}

// ListConditions returns all condition options from dictionary service.
func (c *APIDictionaryRepo) ListConditions(ctx context.Context) ([]DictionaryOption, error) {
	return c.GetOptionsBySetName(ctx, "condition", "en", nil)
}

// Set CRUD implementations for APIDictionaryRepo
func (c *APIDictionaryRepo) ListSets(ctx context.Context) ([]DictionarySet, error) {
	resp, err := c.client.List(ctx, "dictionary/sets")
	if err != nil {
		return nil, err
	}

	// Parse response data
	setsData, ok := resp.Data.([]interface{})
	if !ok {
		return []DictionarySet{}, nil
	}

	sets := make([]DictionarySet, 0, len(setsData))
	for _, item := range setsData {
		setData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		idStr := stringField(setData, "id")
		if idStr == "" {
			continue
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}

		set := DictionarySet{
			ID:          id,
			Name:        stringField(setData, "name"),
			Locale:      stringField(setData, "locale"),
			Label:       stringField(setData, "label"),
			Description: stringField(setData, "description"),
			Active:      boolField(setData, "active"),
			CreatedAt:   timeField(setData, "created_at"),
			UpdatedAt:   timeField(setData, "updated_at"),
		}
		sets = append(sets, set)
	}

	return sets, nil
}

func (c *APIDictionaryRepo) GetSet(ctx context.Context, id uuid.UUID) (*DictionarySet, error) {
	resp, err := c.client.Get(ctx, "dictionary/sets", id.String())
	if err != nil {
		return nil, err
	}

	setData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return parseSetFromMap(setData)
}

func (c *APIDictionaryRepo) CreateSet(ctx context.Context, req *CreateSetRequest) (*DictionarySet, error) {
	resp, err := c.client.Create(ctx, "dictionary/sets", req)
	if err != nil {
		return nil, err
	}

	setData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return parseSetFromMap(setData)
}

func (c *APIDictionaryRepo) UpdateSet(ctx context.Context, id uuid.UUID, req *UpdateSetRequest) (*DictionarySet, error) {
	resp, err := c.client.Update(ctx, "dictionary/sets", id.String(), req)
	if err != nil {
		return nil, err
	}

	setData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return parseSetFromMap(setData)
}

func (c *APIDictionaryRepo) DeleteSet(ctx context.Context, id uuid.UUID) error {
	return c.client.Delete(ctx, "dictionary/sets", id.String())
}

// Option CRUD implementations for APIDictionaryRepo
func (c *APIDictionaryRepo) ListOptions(ctx context.Context, setID *uuid.UUID) ([]DictionaryOptionDetail, error) {
	resp, err := c.client.List(ctx, "dictionary/options")
	if err != nil {
		return nil, err
	}

	// Parse response data
	optionsData, ok := resp.Data.([]interface{})
	if !ok {
		return []DictionaryOptionDetail{}, nil
	}

	options := make([]DictionaryOptionDetail, 0, len(optionsData))
	for _, item := range optionsData {
		optData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		idStr := stringField(optData, "id")
		if idStr == "" {
			continue
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}

		setIDStr := stringField(optData, "set_id")
		if setIDStr == "" {
			continue
		}

		optSetID, err := uuid.Parse(setIDStr)
		if err != nil {
			continue
		}

		// Filter by setID if provided
		if setID != nil && optSetID != *setID {
			continue
		}

		var parentID *uuid.UUID
		parentIDStr := stringField(optData, "parent_id")
		if parentIDStr != "" {
			pid, err := uuid.Parse(parentIDStr)
			if err == nil {
				parentID = &pid
			}
		}

		option := DictionaryOptionDetail{
			ID:          id,
			Set:         optSetID,
			SetName:     stringField(optData, "set_name"),
			ParentID:    parentID,
			ParentLabel: stringField(optData, "parent_label"),
			Locale:      stringField(optData, "locale"),
			ShortCode:   stringField(optData, "short_code"),
			Key:         stringField(optData, "key"),
			Label:       stringField(optData, "label"),
			Description: stringField(optData, "description"),
			Value:       stringField(optData, "value"),
			Order:       intField(optData, "order"),
			Active:      boolField(optData, "active"),
			CreatedAt:   timeField(optData, "created_at"),
			UpdatedAt:   timeField(optData, "updated_at"),
		}
		options = append(options, option)
	}

	return options, nil
}

func (c *APIDictionaryRepo) GetOption(ctx context.Context, id uuid.UUID) (*DictionaryOptionDetail, error) {
	resp, err := c.client.Get(ctx, "dictionary/options", id.String())
	if err != nil {
		return nil, err
	}

	optionData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return parseOptionFromMap(optionData)
}

func (c *APIDictionaryRepo) CreateOption(ctx context.Context, req *CreateOptionRequest) (*DictionaryOptionDetail, error) {
	resp, err := c.client.Create(ctx, "dictionary/options", req)
	if err != nil {
		return nil, err
	}

	optionData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return parseOptionFromMap(optionData)
}

func (c *APIDictionaryRepo) UpdateOption(ctx context.Context, id uuid.UUID, req *UpdateOptionRequest) (*DictionaryOptionDetail, error) {
	resp, err := c.client.Update(ctx, "dictionary/options", id.String(), req)
	if err != nil {
		return nil, err
	}

	optionData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return parseOptionFromMap(optionData)
}

func (c *APIDictionaryRepo) DeleteOption(ctx context.Context, id uuid.UUID) error {
	return c.client.Delete(ctx, "dictionary/options", id.String())
}

// Helper to convert options to map for templates
func DictionaryOptionsToMap(options []DictionaryOption) []map[string]string {
	result := make([]map[string]string, len(options))
	for i, opt := range options {
		result[i] = map[string]string{
			"id":   opt.ID.String(),
			"name": opt.Name,
		}
		if opt.Key != "" {
			result[i]["key"] = opt.Key
		}
	}
	return result
}

// timeField parses a time field from API response data
func timeField(data map[string]interface{}, key string) time.Time {
	value, ok := data[key]
	if !ok || value == nil {
		return time.Time{}
	}

	// Try to parse as string (ISO 8601 format)
	if str, ok := value.(string); ok {
		t, err := time.Parse(time.RFC3339, str)
		if err == nil {
			return t
		}
	}

	return time.Time{}
}

// parseSetFromMap parses a DictionarySet from a map
func parseSetFromMap(data map[string]interface{}) (*DictionarySet, error) {
	idStr := stringField(data, "id")
	if idStr == "" {
		return nil, fmt.Errorf("missing set id")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid set id: %w", err)
	}

	set := &DictionarySet{
		ID:          id,
		Name:        stringField(data, "name"),
		Locale:      stringField(data, "locale"),
		Label:       stringField(data, "label"),
		Description: stringField(data, "description"),
		Active:      boolField(data, "active"),
		CreatedAt:   timeField(data, "created_at"),
		UpdatedAt:   timeField(data, "updated_at"),
	}

	return set, nil
}

// parseOptionFromMap parses a DictionaryOptionDetail from a map
func parseOptionFromMap(data map[string]interface{}) (*DictionaryOptionDetail, error) {
	idStr := stringField(data, "id")
	if idStr == "" {
		return nil, fmt.Errorf("missing option id")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid option id: %w", err)
	}

	setIDStr := stringField(data, "set_id")
	if setIDStr == "" {
		return nil, fmt.Errorf("missing set id")
	}

	setID, err := uuid.Parse(setIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid set id: %w", err)
	}

	var parentID *uuid.UUID
	parentIDStr := stringField(data, "parent_id")
	if parentIDStr != "" {
		pid, err := uuid.Parse(parentIDStr)
		if err == nil {
			parentID = &pid
		}
	}

	option := &DictionaryOptionDetail{
		ID:          id,
		Set:         setID,
		SetName:     stringField(data, "set_name"),
		ParentID:    parentID,
		ParentLabel: stringField(data, "parent_label"),
		Locale:      stringField(data, "locale"),
		ShortCode:   stringField(data, "short_code"),
		Key:         stringField(data, "key"),
		Label:       stringField(data, "label"),
		Description: stringField(data, "description"),
		Value:       stringField(data, "value"),
		Order:       intField(data, "order"),
		Active:      boolField(data, "active"),
		CreatedAt:   timeField(data, "created_at"),
		UpdatedAt:   timeField(data, "updated_at"),
	}

	return option, nil
}
