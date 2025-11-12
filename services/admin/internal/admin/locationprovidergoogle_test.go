package admin

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func newJSONResponse(statusCode int, body string) *http.Response {
	resp := &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	resp.Header.Set("Content-Type", "application/json")
	return resp
}

func TestGoogleMapsProvideAutocomplete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   string
		setup   func(t *testing.T) *http.Client
		wantErr string
		wantLen int
	}{
		{
			name:  "golden path",
			query: "1600 Amphitheatre",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					if req.URL.Path != "/place/autocomplete/json" {
						t.Fatalf("unexpected path: %s", req.URL.Path)
					}
					if got := req.URL.Query().Get("input"); got == "" {
						t.Fatalf("expected input query parameter")
					}
					return newJSONResponse(http.StatusOK, `{
                        "status":"OK",
                        "predictions":[{
                            "description":"1600 Amphitheatre Parkway, Mountain View, CA, USA",
                            "place_id":"abc123",
                            "structured_formatting":{"main_text":"1600 Amphitheatre Pkwy","secondary_text":"Mountain View, CA"}
                        }]
                    }`), nil
				})
			},
			wantLen: 1,
		},
		{
			name:  "provider returns error status",
			query: "foo",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return newJSONResponse(http.StatusOK, `{"status":"OVER_QUERY_LIMIT","error_message":"quota exceeded"}`), nil
				})
			},
			wantErr: "OVER_QUERY_LIMIT",
		},
		{
			name:  "http failure",
			query: "foo",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return newJSONResponse(http.StatusInternalServerError, ""), nil
				})
			},
			wantErr: "unexpected status",
		},
		{
			name:  "decode failure",
			query: "foo",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("not-json"))}, nil
				})
			},
			wantErr: "decode",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := GoogleMapsOptions{APIKey: "test", Endpoint: "https://example.com"}
			if tt.setup != nil {
				opts.HTTPClient = tt.setup(t)
			}

			provider := NewGoogleMapsProvider(opts)
			suggestions, err := provider.Autocomplete(context.Background(), tt.query)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(suggestions) != tt.wantLen {
				t.Fatalf("expected %d suggestions, got %d", tt.wantLen, len(suggestions))
			}

			if len(suggestions) > 0 {
				got := suggestions[0]
				if got.ProviderRef != "abc123" {
					t.Fatalf("expected provider ref abc123, got %s", got.ProviderRef)
				}
				if !strings.Contains(got.ProviderURL, "abc123") {
					t.Fatalf("provider url missing ref: %s", got.ProviderURL)
				}
				if got.Raw["place_id"] != "abc123" {
					t.Fatalf("raw payload missing place_id: %+v", got.Raw)
				}
			}
		})
	}

	t.Run("empty query", func(t *testing.T) {
		provider := NewGoogleMapsProvider(GoogleMapsOptions{})
		_, err := provider.Autocomplete(context.Background(), " ")
		if err == nil || !strings.Contains(err.Error(), "query") {
			t.Fatalf("expected query error, got %v", err)
		}
	})
}

func TestGoogleMapsProvideResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ref     string
		setup   func(t *testing.T) *http.Client
		wantErr string
	}{
		{
			name: "golden path",
			ref:  "abc123",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					if req.URL.Path != "/place/details/json" {
						t.Fatalf("unexpected path: %s", req.URL.Path)
					}
					if got := req.URL.Query().Get("place_id"); got != "abc123" {
						t.Fatalf("expected place_id param, got %s", got)
					}
					return newJSONResponse(http.StatusOK, `{
                        "status":"OK",
                        "result":{
                            "place_id":"abc123",
                            "formatted_address":"1600 Amphitheatre Pkwy, Mountain View, CA",
                            "geometry":{"location":{"lat":37.422,"lng":-122.084}},
                            "address_components":[
                                {"long_name":"1600","types":["street_number"]},
                                {"long_name":"Amphitheatre Parkway","types":["route"]},
                                {"long_name":"Mountain View","types":["locality"]},
                                {"long_name":"California","types":["administrative_area_level_1"]},
                                {"long_name":"94043","types":["postal_code"]},
                                {"long_name":"United States","types":["country"]}
                            ],
                            "url":"https://maps.google.com/?q=abc123"
                        }
                    }`), nil
				})
			},
		},
		{
			name: "zero results",
			ref:  "missing",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return newJSONResponse(http.StatusOK, `{"status":"ZERO_RESULTS"}`), nil
				})
			},
			wantErr: "zero results",
		},
		{
			name: "provider error",
			ref:  "missing",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return newJSONResponse(http.StatusOK, `{"status":"INVALID_REQUEST","error_message":"oops"}`), nil
				})
			},
			wantErr: "INVALID_REQUEST",
		},
		{
			name: "http failure",
			ref:  "foo",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return newJSONResponse(http.StatusInternalServerError, ""), nil
				})
			},
			wantErr: "unexpected status",
		},
		{
			name: "decode failure",
			ref:  "foo",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("not-json"))}, nil
				})
			},
			wantErr: "decode",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := GoogleMapsOptions{Endpoint: "https://example.com"}
			if tt.setup != nil {
				opts.HTTPClient = tt.setup(t)
			}

			provider := NewGoogleMapsProvider(opts)
			got, err := provider.Resolve(context.Background(), tt.ref)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Provider != ProviderGoogle {
				t.Fatalf("expected provider %s, got %s", ProviderGoogle, got.Provider)
			}
			if got.Address.Street != "Amphitheatre Parkway" {
				t.Fatalf("unexpected street: %+v", got.Address)
			}
			if got.Address.Number != "1600" || got.Address.City != "Mountain View" || got.Address.Country != "United States" {
				t.Fatalf("unexpected address: %+v", got.Address)
			}
			if got.Coordinates.Latitude == 0 || got.Coordinates.Longitude == 0 {
				t.Fatalf("expected coordinates to be set, got %+v", got.Coordinates)
			}
			if got.Raw["place_id"] != "abc123" {
				t.Fatalf("raw payload missing place_id: %+v", got.Raw)
			}
		})
	}

	t.Run("empty reference", func(t *testing.T) {
		provider := NewGoogleMapsProvider(GoogleMapsOptions{})
		_, err := provider.Resolve(context.Background(), "")
		if err == nil || !strings.Contains(err.Error(), "reference") {
			t.Fatalf("expected reference error, got %v", err)
		}
	})
}

func TestMapGoogleAddress(t *testing.T) {
	comps := []googleAddressComponent{
		{LongName: "1", Types: []string{"street_number"}},
		{LongName: "Main St", Types: []string{"route"}},
		{LongName: "Downtown", Types: []string{"sublocality"}},
		{LongName: "Metropolis", Types: []string{"locality"}},
		{LongName: "State", Types: []string{"administrative_area_level_1"}},
		{LongName: "12345", Types: []string{"postal_code"}},
		{LongName: "Country", Types: []string{"country"}},
		{LongName: "Apt 5", Types: []string{"subpremise"}},
	}

	addr := mapGoogleAddress(comps)
	if addr.Street != "Main St" || addr.Number != "1" {
		t.Fatalf("unexpected street/number: %+v", addr)
	}
	if addr.Unit != "Apt 5" {
		t.Fatalf("expected unit to be set, got %+v", addr)
	}
	if addr.City != "Metropolis" {
		t.Fatalf("expected city fallback, got %+v", addr)
	}
	if addr.State != "State" || addr.PostalCode != "12345" || addr.Country != "Country" {
		t.Fatalf("unexpected address fields: %+v", addr)
	}
}
