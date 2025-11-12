package admin

import (
	"encoding/json"
	"net/http"
	"strings"
)

type LocationFormModel struct {
	SearchValue  string
	SelectedText string
	Street       string
	Number       string
	Unit         string
	City         string
	State        string
	PostalCode   string
	Country      string
	Region       string
	Latitude     string
	Longitude    string
	Provider     string
	ProviderRef  string
	ProviderURL  string
	RawJSON      string
	Error        string
}

func newLocationFormModel() LocationFormModel {
	return LocationFormModel{}
}

func locationFormModelFromNormalized(normalized *NormalizedLocation) LocationFormModel {
	if normalized == nil {
		return LocationFormModel{}
	}
	model := LocationFormModel{
		SearchValue:  cleanString(normalized.SearchValue),
		SelectedText: cleanString(normalized.SelectedText),
		Street:       cleanString(normalized.Street),
		Number:       cleanString(normalized.Number),
		Unit:         cleanString(normalized.Unit),
		City:         cleanString(normalized.City),
		State:        cleanString(normalized.State),
		PostalCode:   cleanString(normalized.PostalCode),
		Country:      cleanString(normalized.Country),
		Region:       cleanString(normalized.Region),
		Latitude:     cleanString(normalized.Latitude),
		Longitude:    cleanString(normalized.Longitude),
		Provider:     normalized.Provider,
		ProviderRef:  normalized.ProviderRef,
		ProviderURL:  normalized.ProviderURL,
		RawJSON:      normalized.RawJSON,
	}
	if normalized.SelectedText != "" {
		model.SearchValue = cleanString(normalized.SelectedText)
	}
	return model
}

func locationFormModelFromProperty(property *Property) LocationFormModel {
	if property == nil {
		return LocationFormModel{}
	}
	country := expandCountry(cleanString(property.Location.Address.Country))
	model := LocationFormModel{
		SearchValue:  cleanString(composeSearchValue(property.Location.Address)),
		SelectedText: cleanString(property.Location.DisplayName),
		Street:       cleanString(property.Location.Address.Street),
		Number:       cleanString(property.Location.Address.Number),
		Unit:         cleanString(property.Location.Address.Unit),
		City:         cleanString(property.Location.Address.City),
		State:        cleanString(property.Location.Address.State),
		PostalCode:   cleanString(property.Location.Address.PostalCode),
		Country:      country,
		Region:       cleanString(property.Location.Region),
		Provider:     property.Location.Provider,
		ProviderRef:  property.Location.ProviderRef,
		ProviderURL:  property.Location.ProviderURL,
	}
	if property.Location.DisplayName != "" {
		model.SearchValue = cleanString(property.Location.DisplayName)
	}
	if property.Location.Coordinates.Latitude != 0 || property.Location.Coordinates.Longitude != 0 {
		model.Latitude = formatCoordinate(property.Location.Coordinates.Latitude)
		model.Longitude = formatCoordinate(property.Location.Coordinates.Longitude)
	}
	if property.Location.Raw != nil {
		if encoded, err := encodeRawLocation(property.Location.Raw); err == nil {
			model.RawJSON = encoded
		}
	}
	return model
}

func locationFormModelFromRequest(r *http.Request) LocationFormModel {
	return LocationFormModel{
		SearchValue:  cleanString(strings.TrimSpace(r.FormValue("location_search"))),
		SelectedText: cleanString(strings.TrimSpace(r.FormValue("location_display_name"))),
		Street:       cleanString(strings.TrimSpace(r.FormValue("street"))),
		Number:       cleanString(strings.TrimSpace(r.FormValue("number"))),
		Unit:         cleanString(strings.TrimSpace(r.FormValue("unit"))),
		City:         cleanString(strings.TrimSpace(r.FormValue("city"))),
		State:        cleanString(strings.TrimSpace(r.FormValue("state"))),
		PostalCode:   cleanString(strings.TrimSpace(r.FormValue("postal_code"))),
		Country:      cleanString(strings.TrimSpace(r.FormValue("country"))),
		Region:       cleanString(strings.TrimSpace(r.FormValue("region"))),
		Latitude:     cleanString(strings.TrimSpace(r.FormValue("location_latitude"))),
		Longitude:    cleanString(strings.TrimSpace(r.FormValue("location_longitude"))),
		Provider:     strings.TrimSpace(r.FormValue("location_provider")),
		ProviderRef:  strings.TrimSpace(r.FormValue("location_provider_ref")),
		ProviderURL:  strings.TrimSpace(r.FormValue("location_provider_url")),
		RawJSON:      strings.TrimSpace(r.FormValue("location_raw")),
	}
}

func encodeRawLocation(raw map[string]any) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func composeSearchValue(addr Address) string {
	parts := []string{}
	if street := strings.TrimSpace(addr.Street); street != "" {
		if number := strings.TrimSpace(addr.Number); number != "" {
			parts = append(parts, street+" "+number)
		} else {
			parts = append(parts, street)
		}
	}
	if unit := strings.TrimSpace(addr.Unit); unit != "" {
		parts = append(parts, unit)
	}
	cityParts := []string{}
	if city := strings.TrimSpace(addr.City); city != "" {
		cityParts = append(cityParts, city)
	}
	if state := strings.TrimSpace(addr.State); state != "" {
		cityParts = append(cityParts, state)
	}
	if len(cityParts) > 0 {
		parts = append(parts, strings.Join(cityParts, ", "))
	}
	if postal := strings.TrimSpace(addr.PostalCode); postal != "" {
		parts = append(parts, postal)
	}
	if country := strings.TrimSpace(addr.Country); country != "" {
		parts = append(parts, expandCountry(cleanString(country)))
	}
	return strings.Join(parts, ", ")
}
