package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
)

// APIGrantRepo implements GrantRepo by calling AuthZ service APIs
type APIGrantRepo struct {
	httpClient *http.Client
	authzURL   string
	logger     apt.Logger
}

// NewAPIGrantRepo creates a new API-based grant repository
func NewAPIGrantRepo(config *apt.Config, logger apt.Logger) (*APIGrantRepo, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		logger = apt.NewNoopLogger()
	}

	authzURL, _ := config.GetString("services.authz.url")
	if authzURL == "" {
		return nil, fmt.Errorf("services.authz.url not configured")
	}

	return &APIGrantRepo{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		authzURL: authzURL,
		logger:   logger,
	}, nil
}

func (r *APIGrantRepo) Get(ctx context.Context, id uuid.UUID) (*Grant, error) {
	url := fmt.Sprintf("%s/authz/grants/%s", r.authzURL, id.String())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var wrapper struct {
		Data *Grant `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return wrapper.Data, nil
}

func (r *APIGrantRepo) List(ctx context.Context) ([]*Grant, error) {
	url := fmt.Sprintf("%s/authz/grants", r.authzURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var wrapper struct {
		Data []*Grant `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return wrapper.Data, nil
}

func (r *APIGrantRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*Grant, error) {
	url := fmt.Sprintf("%s/authz/grants/users/%s", r.authzURL, userID.String())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var wrapper struct {
		Data []*Grant `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return wrapper.Data, nil
}
