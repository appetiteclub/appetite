package admin

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestLocationIQProvideAutocomplete(t *testing.T) {
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
			query: "Warsaw",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					if !strings.HasSuffix(req.URL.Path, "/autocomplete.php") {
						t.Fatalf("unexpected path: %s", req.URL.Path)
					}
					if req.URL.Query().Get("key") != "token" {
						t.Fatalf("expected api key query parameter")
					}
					return newJSONResponse(http.StatusOK, `[
                        {
                            "place_id":"123",
                            "display_name":"Warsaw, Poland",
                            "lat":"52.23",
                            "lon":"21.01",
                            "osm_type":"relation",
                            "osm_id":"2828",
                            "address":{
                                "city":"Warsaw",
                                "state":"Mazowieckie",
                                "country":"Poland"
                            }
                        }
                    ]`), nil
				})
			},
			wantLen: 1,
		},
		{
			name:  "http failure",
			query: "Warsaw",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return newJSONResponse(http.StatusBadGateway, ""), nil
				})
			},
			wantErr: "unexpected status",
		},
		{
			name:  "provider error payload",
			query: "Warsaw",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return newJSONResponse(http.StatusOK, `{"error":"Invalid key"}`), nil
				})
			},
			wantErr: "Invalid key",
		},
		{
			name:  "decode failure",
			query: "Warsaw",
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

			opts := LocationIQOptions{APIKey: "token", Endpoint: "https://api.locationiq.com/v1"}
			if tt.setup != nil {
				opts.HTTPClient = tt.setup(t)
			}

			provider := NewLocationIQProvider(opts)
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
				if got.ProviderRef != "R:2828" {
					t.Fatalf("unexpected provider ref: %+v", got)
				}
				if got.ProviderURL == "" || !strings.Contains(got.ProviderURL, "relation") {
					t.Fatalf("expected provider url to contain relation, got %s", got.ProviderURL)
				}
				if got.Raw["place_id"] != "123" {
					t.Fatalf("raw payload missing place_id: %+v", got.Raw)
				}
			}
		})
	}

	t.Run("missing api key", func(t *testing.T) {
		provider := NewLocationIQProvider(LocationIQOptions{})
		_, err := provider.Autocomplete(context.Background(), "foo")
		if err == nil || !strings.Contains(err.Error(), "api key") {
			t.Fatalf("expected api key error, got %v", err)
		}
	})
}

func TestLocationIQProvideResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ref     string
		setup   func(t *testing.T) *http.Client
		wantErr string
	}{
		{
			name: "golden path",
			ref:  "R:2828",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					if !strings.HasSuffix(req.URL.Path, "/details.php") {
						t.Fatalf("unexpected path: %s", req.URL.Path)
					}
					if req.URL.Query().Get("osmtype") != "R" || req.URL.Query().Get("osmid") != "2828" {
						t.Fatalf("expected osmtype/osmid query params, got %s %s", req.URL.Query().Get("osmtype"), req.URL.Query().Get("osmid"))
					}
					return newJSONResponse(http.StatusOK, `{
                        "place_id":"123",
                        "display_name":"Warsaw, Poland",
                        "lat":"52.23",
                        "lon":"21.01",
                        "osm_type":"relation",
                        "osm_id":"2828",
                        "address":{
                            "road":"Marszalkowska",
                            "house_number":"1",
                            "city":"Warsaw",
                            "state":"Mazowieckie",
                            "postcode":"00-001",
                            "country":"Poland"
                        }
                    }`), nil
				})
			},
		},
		{
			name: "provider error payload",
			ref:  "R:missing",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return newJSONResponse(http.StatusOK, `{"message":"Not found"}`), nil
				})
			},
			wantErr: "Not found",
		},
		{
			name: "http failure",
			ref:  "W:999",
			setup: func(t *testing.T) *http.Client {
				return newTestClient(func(req *http.Request) (*http.Response, error) {
					return newJSONResponse(http.StatusBadGateway, ""), nil
				})
			},
			wantErr: "unexpected status",
		},
		{
			name: "decode failure",
			ref:  "W:999",
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
			opts := LocationIQOptions{APIKey: "token", Endpoint: "https://api.locationiq.com/v1"}
			if tt.setup != nil {
				opts.HTTPClient = tt.setup(t)
			}

			provider := NewLocationIQProvider(opts)
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

			if got.Provider != ProviderLocationIQ {
				t.Fatalf("expected provider %s, got %s", ProviderLocationIQ, got.Provider)
			}
			if got.Address.Street != "Marszalkowska" || got.Address.Number != "1" {
				t.Fatalf("unexpected address mapping: %+v", got.Address)
			}
			if got.Coordinates.Latitude == 0 || got.Coordinates.Longitude == 0 {
				t.Fatalf("expected coordinates to be set, got %+v", got.Coordinates)
			}
			if got.Raw["place_id"] != "123" {
				t.Fatalf("raw payload missing place_id: %+v", got.Raw)
			}
			if got.ProviderRef != "R:2828" {
				t.Fatalf("unexpected resolved provider ref: %s", got.ProviderRef)
			}
		})
	}

	t.Run("missing api key", func(t *testing.T) {
		provider := NewLocationIQProvider(LocationIQOptions{})
		_, err := provider.Resolve(context.Background(), "ref")
		if err == nil || !strings.Contains(err.Error(), "api key") {
			t.Fatalf("expected api key error, got %v", err)
		}
	})
}
