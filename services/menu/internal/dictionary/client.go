package dictionary

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Client interface for Dictionary Service validation
type Client interface {
	EnsureAllergen(ctx context.Context, id uuid.UUID) error
	EnsureAllergens(ctx context.Context, ids []uuid.UUID) error
	EnsureDietary(ctx context.Context, id uuid.UUID) error
	EnsureDietaryOptions(ctx context.Context, ids []uuid.UUID) error
	EnsureCuisineType(ctx context.Context, id uuid.UUID) error
	EnsureCuisineTypes(ctx context.Context, ids []uuid.UUID) error
	EnsureMenuCategory(ctx context.Context, id uuid.UUID) error
	EnsureMenuCategories(ctx context.Context, ids []uuid.UUID) error
}

// HTTPClient implements the Dictionary Client using HTTP
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTP dictionary client
func NewHTTPClient(baseURL string) *HTTPClient {
	if baseURL == "" {
		baseURL = "http://localhost:8085" // Default dictionary service URL
	}
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// optionResponse represents the response from dictionary service
type optionResponse struct {
	ID     string `json:"id"`
	Set    string `json:"set_id"`
	Key    string `json:"key"`
	Label  string `json:"label"`
	Active bool   `json:"active"`
}

// EnsureAllergen validates that the allergen ID exists in dictionary
func (c *HTTPClient) EnsureAllergen(ctx context.Context, id uuid.UUID) error {
	return c.ensureOptionExists(ctx, id, "allergens")
}

// EnsureAllergens validates that all allergen IDs exist in dictionary
func (c *HTTPClient) EnsureAllergens(ctx context.Context, ids []uuid.UUID) error {
	for _, id := range ids {
		if err := c.EnsureAllergen(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// EnsureDietary validates that the dietary option ID exists in dictionary
func (c *HTTPClient) EnsureDietary(ctx context.Context, id uuid.UUID) error {
	return c.ensureOptionExists(ctx, id, "dietary")
}

// EnsureDietaryOptions validates that all dietary option IDs exist in dictionary
func (c *HTTPClient) EnsureDietaryOptions(ctx context.Context, ids []uuid.UUID) error {
	for _, id := range ids {
		if err := c.EnsureDietary(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// EnsureCuisineType validates that the cuisine type ID exists in dictionary
func (c *HTTPClient) EnsureCuisineType(ctx context.Context, id uuid.UUID) error {
	return c.ensureOptionExists(ctx, id, "cuisine_type")
}

// EnsureCuisineTypes validates that all cuisine type IDs exist in dictionary
func (c *HTTPClient) EnsureCuisineTypes(ctx context.Context, ids []uuid.UUID) error {
	for _, id := range ids {
		if err := c.EnsureCuisineType(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// EnsureMenuCategory validates that the menu category ID exists in dictionary
func (c *HTTPClient) EnsureMenuCategory(ctx context.Context, id uuid.UUID) error {
	return c.ensureOptionExists(ctx, id, "menu_categories")
}

// EnsureMenuCategories validates that all menu category IDs exist in dictionary
func (c *HTTPClient) EnsureMenuCategories(ctx context.Context, ids []uuid.UUID) error {
	for _, id := range ids {
		if err := c.EnsureMenuCategory(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// ensureOptionExists checks if an option with the given ID exists in a specific set
func (c *HTTPClient) ensureOptionExists(ctx context.Context, id uuid.UUID, setName string) error {
	url := fmt.Sprintf("%s/dictionary/options/%s", c.baseURL, id.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dictionary service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("option %s not found in dictionary set %s", id.String(), setName)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dictionary service returned status %d", resp.StatusCode)
	}

	var option optionResponse
	if err := json.NewDecoder(resp.Body).Decode(&option); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !option.Active {
		return fmt.Errorf("option %s is not active in dictionary", id.String())
	}

	return nil
}

// NoopClient is a no-op implementation for testing or when dictionary validation is disabled
type NoopClient struct{}

// NewNoopClient creates a new no-op dictionary client
func NewNoopClient() *NoopClient {
	return &NoopClient{}
}

func (c *NoopClient) EnsureAllergen(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (c *NoopClient) EnsureAllergens(ctx context.Context, ids []uuid.UUID) error {
	return nil
}

func (c *NoopClient) EnsureDietary(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (c *NoopClient) EnsureDietaryOptions(ctx context.Context, ids []uuid.UUID) error {
	return nil
}

func (c *NoopClient) EnsureCuisineType(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (c *NoopClient) EnsureCuisineTypes(ctx context.Context, ids []uuid.UUID) error {
	return nil
}

func (c *NoopClient) EnsureMenuCategory(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (c *NoopClient) EnsureMenuCategories(ctx context.Context, ids []uuid.UUID) error {
	return nil
}
