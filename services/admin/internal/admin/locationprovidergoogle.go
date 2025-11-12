package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GoogleMapsProvider implements LocationProvider for Google Maps APIs.
type GoogleMapsProvider struct {
	client   *http.Client
	apiKey   string
	endpoint string
}

// GoogleMapsOptions configures the GoogleMapsProvider.
type GoogleMapsOptions struct {
	HTTPClient *http.Client
	APIKey     string
	Endpoint   string
}

const defaultGoogleEndpoint = "https://maps.googleapis.com/maps/api"

// NewGoogleMapsProvider creates a GoogleMapsProvider with sane defaults.
func NewGoogleMapsProvider(opts GoogleMapsOptions) *GoogleMapsProvider {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	endpoint := strings.TrimRight(opts.Endpoint, "/")
	if endpoint == "" {
		endpoint = defaultGoogleEndpoint
	}

	return &GoogleMapsProvider{
		client:   client,
		apiKey:   opts.APIKey,
		endpoint: endpoint,
	}
}

func (p *GoogleMapsProvider) ProviderID() string {
	return ProviderGoogle
}

func (p *GoogleMapsProvider) Autocomplete(ctx context.Context, query string) ([]LocationSuggestion, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query cannot be empty")
	}

	values := url.Values{}
	values.Set("input", query)
	values.Set("types", "address")
	if p.apiKey != "" {
		values.Set("key", p.apiKey)
	}

	endpoint := fmt.Sprintf("%s/place/autocomplete/json?%s", p.endpoint, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("google autocomplete request build: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google autocomplete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google autocomplete unexpected status: %d", resp.StatusCode)
	}

	var payload googleAutocompleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("google autocomplete decode: %w", err)
	}

	switch payload.Status {
	case "OK":
		suggestions := make([]LocationSuggestion, 0, len(payload.Predictions))
		for _, pred := range payload.Predictions {
			raw := map[string]any{
				"description":           pred.Description,
				"place_id":              pred.PlaceID,
				"structured_formatting": pred.StructuredFormatting,
			}
			suggestions = append(suggestions, LocationSuggestion{
				Text:        pred.Description,
				ProviderRef: pred.PlaceID,
				ProviderURL: fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=place_id:%s", url.QueryEscape(pred.PlaceID)),
				Raw:         raw,
			})
		}
		return suggestions, nil
	case "ZERO_RESULTS":
		return []LocationSuggestion{}, nil
	default:
		if payload.ErrorMessage != "" {
			return nil, fmt.Errorf("google autocomplete %s: %s", payload.Status, payload.ErrorMessage)
		}
		return nil, fmt.Errorf("google autocomplete status %s", payload.Status)
	}
}

func (p *GoogleMapsProvider) Resolve(ctx context.Context, reference string) (*ResolvedAddress, error) {
	if strings.TrimSpace(reference) == "" {
		return nil, errors.New("reference cannot be empty")
	}

	values := url.Values{}
	values.Set("place_id", reference)
	values.Set("fields", "formatted_address,address_component,geometry,url")
	if p.apiKey != "" {
		values.Set("key", p.apiKey)
	}

	endpoint := fmt.Sprintf("%s/place/details/json?%s", p.endpoint, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("google resolve request build: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google resolve request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google resolve unexpected status: %d", resp.StatusCode)
	}

	var payload googleDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("google resolve decode: %w", err)
	}

	switch payload.Status {
	case "OK":
		addr := mapGoogleAddress(payload.Result.AddressComponents)
		coords := Coordinates{
			Latitude:  payload.Result.Geometry.Location.Lat,
			Longitude: payload.Result.Geometry.Location.Lng,
		}
		providerURL := payload.Result.URL
		if providerURL == "" {
			providerURL = fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=place_id:%s", url.QueryEscape(reference))
		}

		raw := map[string]any{
			"place_id":           payload.Result.PlaceID,
			"formatted_address":  payload.Result.FormattedAddress,
			"address_components": payload.Result.AddressComponents,
			"geometry":           payload.Result.Geometry,
		}

		return &ResolvedAddress{
			Formatted:   payload.Result.FormattedAddress,
			Address:     addr,
			Coordinates: coords,
			Provider:    ProviderGoogle,
			ProviderRef: payload.Result.PlaceID,
			ProviderURL: providerURL,
			Raw:         raw,
		}, nil
	case "ZERO_RESULTS":
		return nil, fmt.Errorf("google resolve zero results")
	default:
		if payload.ErrorMessage != "" {
			return nil, fmt.Errorf("google resolve %s: %s", payload.Status, payload.ErrorMessage)
		}
		return nil, fmt.Errorf("google resolve status %s", payload.Status)
	}
}

func mapGoogleAddress(components []googleAddressComponent) Address {
	var addr Address
	var street, number, city, state, postal, country, unit, secondaryCity, adminArea2 string

	for _, comp := range components {
		for _, t := range comp.Types {
			switch t {
			case "street_number":
				number = comp.LongName
			case "route":
				street = comp.LongName
			case "sublocality", "sublocality_level_1":
				if secondaryCity == "" {
					secondaryCity = comp.LongName
				}
			case "locality":
				city = comp.LongName
			case "postal_town":
				if city == "" {
					city = comp.LongName
				}
			case "administrative_area_level_1":
				state = comp.LongName
			case "administrative_area_level_2":
				adminArea2 = comp.LongName
			case "country":
				country = comp.LongName
			case "postal_code":
				postal = comp.LongName
			case "subpremise":
				if unit == "" {
					unit = comp.LongName
				}
			}
		}
	}

	if city == "" {
		city = secondaryCity
	}
	if city == "" {
		city = adminArea2
	}

	addr.Street = street
	addr.Number = number
	addr.City = city
	addr.State = state
	addr.PostalCode = postal
	addr.Country = country
	addr.Unit = unit

	return addr
}

type googleAutocompleteResponse struct {
	Status       string                         `json:"status"`
	Predictions  []googleAutocompletePrediction `json:"predictions"`
	ErrorMessage string                         `json:"error_message"`
}

type googleAutocompletePrediction struct {
	Description          string                     `json:"description"`
	PlaceID              string                     `json:"place_id"`
	StructuredFormatting googleStructuredFormatting `json:"structured_formatting"`
}

type googleStructuredFormatting struct {
	MainText      string `json:"main_text"`
	SecondaryText string `json:"secondary_text"`
}

type googleDetailsResponse struct {
	Status       string             `json:"status"`
	Result       googlePlaceDetails `json:"result"`
	ErrorMessage string             `json:"error_message"`
}

type googlePlaceDetails struct {
	PlaceID           string                   `json:"place_id"`
	FormattedAddress  string                   `json:"formatted_address"`
	AddressComponents []googleAddressComponent `json:"address_components"`
	Geometry          googleGeometry           `json:"geometry"`
	URL               string                   `json:"url"`
}

type googleGeometry struct {
	Location googleLocation `json:"location"`
}

type googleLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type googleAddressComponent struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}
