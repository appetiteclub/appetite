package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aquamarinepk/aqm"
	"github.com/google/uuid"
)

// APIRoleRepo implements RoleRepo by calling AuthZ service APIs
type APIRoleRepo struct {
	httpClient *http.Client
	authzURL   string
	logger     aqm.Logger
}

// NewAPIRoleRepo creates a new API-based role repository
func NewAPIRoleRepo(config *aqm.Config, logger aqm.Logger) (*APIRoleRepo, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}

	authzURL, _ := config.GetString("services.authz.url")
	if authzURL == "" {
		return nil, fmt.Errorf("services.authz.url not configured")
	}

	return &APIRoleRepo{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		authzURL: authzURL,
		logger:   logger,
	}, nil
}

func (r *APIRoleRepo) Get(ctx context.Context, id uuid.UUID) (*Role, error) {
	url := fmt.Sprintf("%s/authz/roles/%s", r.authzURL, id.String())

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
		Data *Role `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return wrapper.Data, nil
}

func (r *APIRoleRepo) List(ctx context.Context) ([]*Role, error) {
	url := fmt.Sprintf("%s/authz/roles", r.authzURL)

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
		Data []*Role `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return wrapper.Data, nil
}

func (r *APIRoleRepo) ListByStatus(ctx context.Context, status string) ([]*Role, error) {
	url := fmt.Sprintf("%s/authz/roles?status=%s", r.authzURL, status)

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
		Data []*Role `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return wrapper.Data, nil
}
