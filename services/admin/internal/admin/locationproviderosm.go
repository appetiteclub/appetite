package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// OpenStreetMapProvider implements LocationProvider backed by Nominatim.
type OpenStreetMapProvider struct {
	client    *http.Client
	endpoint  string
	email     string
	userAgent string
}

// OpenStreetMapOptions configures the OpenStreetMapProvider.
type OpenStreetMapOptions struct {
	HTTPClient *http.Client
	Endpoint   string
	Email      string
}

const (
	defaultOSMEndpoint  = "https://nominatim.openstreetmap.org"
	defaultOSMUserAgent = "appetite-admin-service/1.0"
)

// NewOpenStreetMapProvider creates a provider for OSM's Nominatim service.
func NewOpenStreetMapProvider(opts OpenStreetMapOptions) *OpenStreetMapProvider {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	endpoint := strings.TrimRight(opts.Endpoint, "/")
	if endpoint == "" {
		endpoint = defaultOSMEndpoint
	}

	return &OpenStreetMapProvider{
		client:    client,
		endpoint:  endpoint,
		email:     opts.Email,
		userAgent: defaultOSMUserAgent,
	}
}

func (p *OpenStreetMapProvider) ProviderID() string {
	return ProviderOSM
}

func (p *OpenStreetMapProvider) Autocomplete(ctx context.Context, query string) ([]LocationSuggestion, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query cannot be empty")
	}

	values := url.Values{}
	values.Set("q", query)
	values.Set("format", "jsonv2")
	values.Set("addressdetails", "1")
	values.Set("limit", "5")

	endpoint := fmt.Sprintf("%s/search?%s", p.endpoint, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("osm autocomplete request build: %w", err)
	}
	p.decorateRequest(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("osm autocomplete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osm autocomplete unexpected status: %d", resp.StatusCode)
	}

	var payload []osmSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("osm autocomplete decode: %w", err)
	}

	suggestions := make([]LocationSuggestion, 0, len(payload))
	for _, item := range payload {
		ref := buildOSMReference(item.OSMType, item.OSMID)
		suggestions = append(suggestions, LocationSuggestion{
			Text:        item.DisplayName,
			ProviderRef: ref,
			ProviderURL: fmt.Sprintf("https://www.openstreetmap.org/%s/%s", strings.ToLower(item.OSMType), item.OSMID),
			Raw: map[string]any{
				"osm_id":       item.OSMID,
				"osm_type":     item.OSMType,
				"display_name": item.DisplayName,
			},
		})
	}

	return suggestions, nil
}

func (p *OpenStreetMapProvider) Resolve(ctx context.Context, reference string) (*ResolvedAddress, error) {
	if strings.TrimSpace(reference) == "" {
		return nil, errors.New("reference cannot be empty")
	}

	osmType, osmID, err := parseOSMReference(reference)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	values.Set("format", "jsonv2")
	values.Set("addressdetails", "1")
	values.Set("osm_ids", fmt.Sprintf("%s%s", strings.ToUpper(osmType[:1]), osmID))

	endpoint := fmt.Sprintf("%s/lookup?%s", p.endpoint, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("osm resolve request build: %w", err)
	}
	p.decorateRequest(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("osm resolve request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osm resolve unexpected status: %d", resp.StatusCode)
	}

	var payload []osmSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("osm resolve decode: %w", err)
	}

	if len(payload) == 0 {
		return nil, fmt.Errorf("osm resolve zero results")
	}

	item := payload[0]
	lat, _ := strconv.ParseFloat(item.Lat, 64)
	lng, _ := strconv.ParseFloat(item.Lon, 64)

	addr := mapOSMAddress(item.Address)

	raw := map[string]any{
		"osm_id":       item.OSMID,
		"osm_type":     item.OSMType,
		"display_name": item.DisplayName,
		"address":      item.Address,
	}

	return &ResolvedAddress{
		Formatted:   item.DisplayName,
		Address:     addr,
		Coordinates: Coordinates{Latitude: lat, Longitude: lng},
		Provider:    ProviderOSM,
		ProviderRef: buildOSMReference(item.OSMType, item.OSMID),
		ProviderURL: fmt.Sprintf("https://www.openstreetmap.org/%s/%s", strings.ToLower(item.OSMType), item.OSMID),
		Raw:         raw,
	}, nil
}

func (p *OpenStreetMapProvider) decorateRequest(req *http.Request) {
	req.Header.Set("User-Agent", p.userAgent)
	if p.email != "" {
		req.Header.Set("From", p.email)
	}
}

func buildOSMReference(osmType, osmID string) string {
	if osmType == "" || osmID == "" {
		return ""
	}
	prefix := strings.ToUpper(osmType[:1])
	return prefix + osmID
}

func parseOSMReference(ref string) (string, string, error) {
	if len(ref) < 2 {
		return "", "", fmt.Errorf("invalid osm reference: %s", ref)
	}
	prefix := strings.ToUpper(ref[:1])
	id := ref[1:]
	switch prefix {
	case "N":
		return "node", id, nil
	case "W":
		return "way", id, nil
	case "R":
		return "relation", id, nil
	default:
		return "", "", fmt.Errorf("unknown osm reference prefix: %s", prefix)
	}
}

func mapOSMAddress(addr osmAddress) Address {
	var out Address
	out.Street = firstNonEmpty(addr.Road, addr.Pedestrian, addr.Cycleway)
	out.Number = firstNonEmpty(addr.HouseNumber)
	out.Unit = addr.Unit
	out.City = firstNonEmpty(addr.City, addr.Town, addr.Village, addr.Municipality, addr.Suburb, addr.Neighbourhood)
	out.State = firstNonEmpty(addr.State, addr.County)
	out.PostalCode = addr.Postcode
	out.Country = addr.Country
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

type osmSearchResult struct {
	OSMID       string     `json:"osm_id"`
	OSMType     string     `json:"osm_type"`
	DisplayName string     `json:"display_name"`
	Lat         string     `json:"lat"`
	Lon         string     `json:"lon"`
	Address     osmAddress `json:"address"`
}

type osmAddress struct {
	HouseNumber   string `json:"house_number"`
	Road          string `json:"road"`
	Pedestrian    string `json:"pedestrian"`
	Cycleway      string `json:"cycleway"`
	City          string `json:"city"`
	Town          string `json:"town"`
	Village       string `json:"village"`
	Municipality  string `json:"municipality"`
	County        string `json:"county"`
	State         string `json:"state"`
	Postcode      string `json:"postcode"`
	Country       string `json:"country"`
	Suburb        string `json:"suburb"`
	Neighbourhood string `json:"neighbourhood"`
	Unit          string `json:"unit"`
}
