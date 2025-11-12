package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// LocationIQProvider implements LocationProvider backed by LocationIQ APIs.
type LocationIQProvider struct {
	client   *http.Client
	apiKey   string
	endpoint string
}

// LocationIQOptions configures the LocationIQ provider.
type LocationIQOptions struct {
	HTTPClient *http.Client
	APIKey     string
	Endpoint   string
}

const defaultLocationIQEndpoint = "https://api.locationiq.com/v1"

// NewLocationIQProvider creates a LocationIQ provider with sane defaults.
func NewLocationIQProvider(opts LocationIQOptions) *LocationIQProvider {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	endpoint := strings.TrimRight(opts.Endpoint, "/")
	if endpoint == "" {
		endpoint = defaultLocationIQEndpoint
	}

	return &LocationIQProvider{
		client:   client,
		apiKey:   opts.APIKey,
		endpoint: endpoint,
	}
}

func (p *LocationIQProvider) ProviderID() string {
	return ProviderLocationIQ
}

func (p *LocationIQProvider) Autocomplete(ctx context.Context, query string) ([]LocationSuggestion, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query cannot be empty")
	}
	if strings.TrimSpace(p.apiKey) == "" {
		return nil, errors.New("locationiq api key is required")
	}

	values := url.Values{}
	values.Set("key", p.apiKey)
	values.Set("q", query)
	values.Set("limit", "5")
	values.Set("format", "json")
	values.Set("normalizeaddress", "1")

	endpoint := fmt.Sprintf("%s/autocomplete.php?%s", p.endpoint, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("locationiq autocomplete request build: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("locationiq autocomplete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("locationiq autocomplete unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("locationiq autocomplete read: %w", err)
	}

	var results []locationIQAutocompleteResult
	if err := json.Unmarshal(body, &results); err != nil {
		if apiErr := decodeLocationIQError(body); apiErr != nil {
			return nil, apiErr
		}
		return nil, fmt.Errorf("locationiq autocomplete decode: %w", err)
	}

	suggestions := make([]LocationSuggestion, 0, len(results))
	for _, res := range results {
		reference := buildLocationIQReference(res.OSMType, res.OSMID)
		placeID := normalizePlaceID(res.PlaceID)
		osmIDStr := normalizePlaceID(res.OSMID)
		raw := map[string]any{
			"place_id":     placeID,
			"display_name": res.DisplayName,
			"lat":          res.Lat,
			"lon":          res.Lon,
			"osm_id":       res.OSMID,
			"osm_type":     res.OSMType,
			"address":      res.Address,
		}

		suggestions = append(suggestions, LocationSuggestion{
			Text:        res.DisplayName,
			ProviderRef: reference,
			ProviderURL: buildLocationIQURL(res.OSMType, osmIDStr),
			Raw:         raw,
		})
	}

	return suggestions, nil
}

func (p *LocationIQProvider) Resolve(ctx context.Context, reference string) (*ResolvedAddress, error) {
	if strings.TrimSpace(reference) == "" {
		return nil, errors.New("reference cannot be empty")
	}
	if strings.TrimSpace(p.apiKey) == "" {
		return nil, errors.New("locationiq api key is required")
	}

	osmType, osmID, err := parseLocationIQReference(reference)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	values.Set("key", p.apiKey)
	values.Set("osmtype", osmType)
	values.Set("osmid", osmID)
	values.Set("format", "json")

	endpoint := fmt.Sprintf("%s/details.php?%s", p.endpoint, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("locationiq resolve request build: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("locationiq resolve request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("locationiq resolve unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("locationiq resolve read: %w", err)
	}

	if apiErr := decodeLocationIQError(body); apiErr != nil {
		return nil, apiErr
	}

	var payload locationIQDetailsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("locationiq resolve decode: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		raw = map[string]any{}
	}

	addr := mapLocationIQAddress(payload.Address)

	lat := parseFloat(payload.Lat)
	lng := parseFloat(payload.Lon)

	osmIDStr := normalizePlaceID(payload.OSMID)
	providerURL := buildLocationIQURL(payload.OSMType, osmIDStr)

	placeID := normalizePlaceID(payload.PlaceID)

	if raw != nil {
		raw["place_id"] = placeID
	}
	if addr.PostalCode == "" && strings.TrimSpace(payload.CalculatedPostcode) != "" {
		addr.PostalCode = strings.TrimSpace(payload.CalculatedPostcode)
	}

	return &ResolvedAddress{
		Formatted:   payload.DisplayName,
		Address:     addr,
		Coordinates: Coordinates{Latitude: lat, Longitude: lng},
		Provider:    ProviderLocationIQ,
		ProviderRef: buildLocationIQReference(payload.OSMType, payload.OSMID),
		ProviderURL: providerURL,
		Raw:         raw,
	}, nil
}

func decodeLocationIQError(body []byte) error {
	var errPayload locationIQErrorResponse
	if err := json.Unmarshal(body, &errPayload); err == nil {
		if errPayload.Error != "" {
			return fmt.Errorf("locationiq error: %s", errPayload.Error)
		}
		if errPayload.Message != "" {
			return fmt.Errorf("locationiq error: %s", errPayload.Message)
		}
	}
	return nil
}

func mapLocationIQAddress(data any) Address {
	convert := func(m map[string]any) Address {
		get := func(key string) string {
			if v, ok := m[key]; ok {
				if str, ok := v.(string); ok {
					return strings.TrimSpace(str)
				}
			}
			return ""
		}

		out := Address{}
		out.Number = firstNonEmptyValue(get("house_number"), get("addr_house_number"))
		out.Street = firstNonEmptyValue(
			get("road"),
			get("pedestrian"),
			get("footway"),
			get("neighbourhood"),
			get("residential"),
		)
		out.Unit = firstNonEmptyValue(get("unit"), get("suite"), get("level"), get("apartment"))
		out.City = firstNonEmptyValue(get("city"), get("town"), get("village"), get("hamlet"), get("municipality"), get("county"), get("state_district"))
		out.State = firstNonEmptyValue(get("state"), get("region"))
		out.PostalCode = firstNonEmptyValue(get("postcode"), get("postal_code"))
		out.Country = firstNonEmptyValue(get("country"), get("country_code"))

		return out
	}

	parsed := Address{}
	switch v := data.(type) {
	case map[string]any:
		parsed = convert(v)
	case []any:
		combined := map[string]any{}
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name, _ := m["name"].(string)
			if name == "" {
				continue
			}
			typ, _ := m["type"].(string)
			if typ == "" {
				typ = fmt.Sprintf("%v", m["class"])
			}
			typ = strings.ToLower(strings.TrimSpace(typ))
			if typ == "" {
				continue
			}
			combined[typ] = name
		}
		if len(combined) > 0 {
			parsed = convert(combined)
		}
	case nil:
		return Address{}
	}

	return parsed
}

func buildLocationIQURL(osmType, osmID string) string {
	if osmType == "" || osmID == "" || osmID == "0" {
		return ""
	}
	osmType = strings.ToLower(osmType)
	switch osmType {
	case "n", "node":
		return fmt.Sprintf("https://www.openstreetmap.org/node/%s", osmID)
	case "w", "way":
		return fmt.Sprintf("https://www.openstreetmap.org/way/%s", osmID)
	case "r", "relation", "rel":
		return fmt.Sprintf("https://www.openstreetmap.org/relation/%s", osmID)
	}
	return ""
}

func firstNonEmptyValue(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func buildLocationIQReference(osmType string, osmID any) string {
	if osmType == "" || osmID == nil {
		return ""
	}
	osmIDStr := normalizePlaceID(osmID)
	typeRune := ' '
	if len(osmType) > 0 {
		typeRune = rune(osmType[0])
	}
	if typeRune == ' ' {
		return osmIDStr
	}
	return fmt.Sprintf("%c:%s", unicode.ToUpper(typeRune), osmIDStr)
}

func parseLocationIQReference(ref string) (string, string, error) {
	parts := strings.Split(ref, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid locationiq reference: %s", ref)
	}
	typ := strings.ToUpper(strings.TrimSpace(parts[0]))
	id := strings.TrimSpace(parts[1])
	if typ == "" || id == "" {
		return "", "", fmt.Errorf("invalid locationiq reference: %s", ref)
	}
	return typ, id, nil
}

func normalizePlaceID(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case json.Number:
		return v.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func parseFloat(value string) float64 {
	if value == "" {
		return 0
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return f
}

type locationIQAutocompleteResult struct {
	PlaceID     any    `json:"place_id"`
	DisplayName string `json:"display_name"`
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	OSMID       any    `json:"osm_id"`
	OSMType     string `json:"osm_type"`
	Address     any    `json:"address"`
}

type locationIQDetailsResponse struct {
	PlaceID            any    `json:"place_id"`
	DisplayName        string `json:"display_name"`
	Lat                string `json:"lat"`
	Lon                string `json:"lon"`
	OSMID              any    `json:"osm_id"`
	OSMType            string `json:"osm_type"`
	Address            any    `json:"address"`
	CalculatedPostcode string `json:"calculated_postcode"`
}

type locationIQErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
