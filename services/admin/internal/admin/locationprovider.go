package admin

import (
	"context"
	"strings"

	"github.com/aquamarinepk/aqm"
)

// LocationProvider defines the behavior required to integrate external geocoding providers.
type LocationProvider interface {
	// ProviderID returns the identifier (e.g. "google", "osm") for the provider.
	ProviderID() string
	// Autocomplete returns address suggestions for the given free-form query.
	Autocomplete(ctx context.Context, query string) ([]LocationSuggestion, error)
	// Resolve resolves a provider reference (e.g. place_id) into a structured address.
	Resolve(ctx context.Context, reference string) (*ResolvedAddress, error)
}

// LocationSuggestion represents a single autocomplete suggestion returned by a provider.
type LocationSuggestion struct {
	Text        string         `json:"text"`
	ProviderRef string         `json:"provider_ref"`
	ProviderURL string         `json:"provider_url,omitempty"`
	Raw         map[string]any `json:"raw,omitempty"`
}

// ResolvedAddress wraps the provider-normalized address payload.
type ResolvedAddress struct {
	Formatted   string         `json:"formatted"`
	Address     Address        `json:"address"`
	Coordinates Coordinates    `json:"coordinates"`
	Provider    string         `json:"provider"`
	ProviderRef string         `json:"provider_ref"`
	ProviderURL string         `json:"provider_url,omitempty"`
	Raw         map[string]any `json:"raw,omitempty"`
}

const (
	ProviderGoogle     = "google"
	ProviderOSM        = "osm"
	ProviderLocationIQ = "locationiq"
)

// NewLocationProvider creates a LocationProvider from configuration properties.
// Returns nil if the provider is disabled or configuration is missing.
func NewLocationProvider(config *aqm.Config) LocationProvider {
	if config == nil {
		return nil
	}

	providerName, _ := config.GetString("geocode.provider")
	provider := strings.ToLower(strings.TrimSpace(providerName))

	switch provider {
	case "", ProviderLocationIQ:
		keyVal, _ := config.GetString("geocode.locationiq.key")
		key := strings.TrimSpace(keyVal)
		if key == "" {
			return nil
		}
		endpoint, _ := config.GetString("geocode.locationiq.endpoint")
		return NewLocationIQProvider(LocationIQOptions{
			APIKey:   key,
			Endpoint: endpoint,
		})

	case ProviderGoogle:
		apiKey, _ := config.GetString("geocode.google.api.key")
		endpoint, _ := config.GetString("geocode.google.endpoint")
		return NewGoogleMapsProvider(GoogleMapsOptions{
			APIKey:   strings.TrimSpace(apiKey),
			Endpoint: endpoint,
		})

	case ProviderOSM:
		endpoint, _ := config.GetString("geocode.osm.endpoint")
		email, _ := config.GetString("geocode.osm.email")
		return NewOpenStreetMapProvider(OpenStreetMapOptions{
			Endpoint: endpoint,
			Email:    email,
		})

	default:
		return nil
	}
}
