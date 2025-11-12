package admin

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOpenStreetMapProvideAutocomplete(t *testing.T) {
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
			query: "obelisco",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					if req.URL.Path != "/search" {
						t.Fatalf("unexpected path: %s", req.URL.Path)
					}
					if got := req.URL.Query().Get("q"); got == "" {
						t.Fatalf("expected q parameter")
					}
					return newJSONResponse(http.StatusOK, `[{"osm_id":"123","osm_type":"way","display_name":"Obelisco, Buenos Aires","lat":"-34.6037","lon":"-58.3816"}]`), nil
				})
			},
			wantLen: 1,
		},
		{
			name:  "http failure",
			query: "obelisco",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return newJSONResponse(http.StatusBadGateway, ""), nil
				})
			},
			wantErr: "unexpected status",
		},
		{
			name:  "decode failure",
			query: "obelisco",
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

			opts := OpenStreetMapOptions{Endpoint: "https://nominatim.test"}
			if tt.setup != nil {
				opts.HTTPClient = tt.setup(t)
			}

			provider := NewOpenStreetMapProvider(opts)
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
				if got.ProviderRef != "W123" {
					t.Fatalf("unexpected provider ref: %s", got.ProviderRef)
				}
				if got.Raw["osm_id"] != "123" {
					t.Fatalf("raw payload missing osm_id: %+v", got.Raw)
				}
			}
		})
	}

	t.Run("empty query", func(t *testing.T) {
		provider := NewOpenStreetMapProvider(OpenStreetMapOptions{})
		_, err := provider.Autocomplete(context.Background(), "")
		if err == nil || !strings.Contains(err.Error(), "query") {
			t.Fatalf("expected query error, got %v", err)
		}
	})
}

func TestOpenStreetMapProvideResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ref     string
		setup   func(t *testing.T) *http.Client
		wantErr string
	}{
		{
			name: "golden path",
			ref:  "W123",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					if req.URL.Path != "/lookup" {
						t.Fatalf("unexpected path: %s", req.URL.Path)
					}
					return newJSONResponse(http.StatusOK, `[{"osm_id":"123","osm_type":"way","display_name":"Obelisco, Buenos Aires","lat":"-34.6037","lon":"-58.3816","address":{"road":"Av. 9 de Julio","house_number":"1","city":"Buenos Aires","state":"Buenos Aires","postcode":"C1043","country":"Argentina"}}]`), nil
				})
			},
		},
		{
			name: "zero results",
			ref:  "W999",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return newJSONResponse(http.StatusOK, `[]`), nil
				})
			},
			wantErr: "zero results",
		},
		{
			name: "http failure",
			ref:  "W999",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return newJSONResponse(http.StatusBadGateway, ""), nil
				})
			},
			wantErr: "unexpected status",
		},
		{
			name: "decode failure",
			ref:  "W999",
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

			opts := OpenStreetMapOptions{Endpoint: "https://nominatim.test"}
			if tt.setup != nil {
				opts.HTTPClient = tt.setup(t)
			}

			provider := NewOpenStreetMapProvider(opts)
			result, err := provider.Resolve(context.Background(), tt.ref)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Provider != ProviderOSM {
				t.Fatalf("expected provider %s, got %s", ProviderOSM, result.Provider)
			}
			if result.Address.Street != "Av. 9 de Julio" || result.Address.Number != "1" {
				t.Fatalf("unexpected address: %+v", result.Address)
			}
			if result.Coordinates.Latitude == 0 || result.Coordinates.Longitude == 0 {
				t.Fatalf("expected coordinates to be set: %+v", result.Coordinates)
			}
		})
	}

	t.Run("invalid reference", func(t *testing.T) {
		provider := NewOpenStreetMapProvider(OpenStreetMapOptions{})
		_, err := provider.Resolve(context.Background(), "X1")
		if err == nil || !strings.Contains(err.Error(), "unknown") {
			t.Fatalf("expected unknown prefix error, got %v", err)
		}
	})

	t.Run("empty reference", func(t *testing.T) {
		provider := NewOpenStreetMapProvider(OpenStreetMapOptions{})
		_, err := provider.Resolve(context.Background(), "")
		if err == nil || !strings.Contains(err.Error(), "reference") {
			t.Fatalf("expected reference error, got %v", err)
		}
	})
}

func TestParseOSMReference(t *testing.T) {
	tests := []struct {
		ref     string
		wantTyp string
		wantID  string
		wantErr bool
	}{
		{ref: "N1", wantTyp: "node", wantID: "1"},
		{ref: "W2", wantTyp: "way", wantID: "2"},
		{ref: "R3", wantTyp: "relation", wantID: "3"},
		{ref: "X4", wantErr: true},
		{ref: "Z", wantErr: true},
	}

	for _, tt := range tests {
		gotType, gotID, err := parseOSMReference(tt.ref)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("expected error for ref %s", tt.ref)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for ref %s: %v", tt.ref, err)
		}
		if gotType != tt.wantTyp || gotID != tt.wantID {
			t.Fatalf("unexpected result for %s: %s %s", tt.ref, gotType, gotID)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "foo", "bar"); got != "foo" {
		t.Fatalf("expected foo, got %s", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Fatalf("expected empty string when all values empty")
	}
}
