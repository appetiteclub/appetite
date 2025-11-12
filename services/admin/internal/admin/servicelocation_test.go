package admin

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/appetiteclub/appetite/services/admin/internal/config"
	"github.com/aquamarinepk/aqm"
)

type stubLocationProvider struct {
	id              string
	suggestions     []LocationSuggestion
	resolved        *ResolvedAddress
	autocompleteErr error
	resolveErr      error
}

var (
	errAutocompleteBoom = errors.New("boom")
	errResolveFail      = errors.New("fail")
)

func (s *stubLocationProvider) ProviderID() string {
	if s.id != "" {
		return s.id
	}
	return "stub"
}

func (s *stubLocationProvider) Autocomplete(ctx context.Context, query string) ([]LocationSuggestion, error) {
	return s.suggestions, s.autocompleteErr
}

func (s *stubLocationProvider) Resolve(ctx context.Context, reference string) (*ResolvedAddress, error) {
	return s.resolved, s.resolveErr
}

func TestDefaultServiceSuggestLocations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name       string
		provider   LocationProvider
		wantErr    error
		wantResult []LocationSuggestion
	}{
		{
			name:     "provider not configured",
			provider: nil,
			wantErr:  ErrLocationProviderUnavailable,
		},
		{
			name: "provider returns suggestions",
			provider: &stubLocationProvider{
				suggestions: []LocationSuggestion{{Text: "Foo", ProviderRef: "ref"}},
			},
			wantResult: []LocationSuggestion{{Text: "Foo", ProviderRef: "ref"}},
		},
		{
			name: "provider returns error",
			provider: &stubLocationProvider{
				autocompleteErr: errAutocompleteBoom,
			},
			wantErr: errAutocompleteBoom,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewDefaultService(Repos{}, tt.provider, configMock())
			got, err := svc.SuggestLocations(ctx, "foo")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.wantResult) {
				t.Fatalf("expected %d suggestions, got %d", len(tt.wantResult), len(got))
			}
			if len(got) > 0 && got[0].ProviderRef != tt.wantResult[0].ProviderRef {
				t.Fatalf("unexpected suggestion result: %+v", got[0])
			}
		})
	}
}

func TestDefaultServiceResolveLocation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	resolved := &ResolvedAddress{Provider: "stub", ProviderRef: "abc"}

	tests := []struct {
		name     string
		provider LocationProvider
		wantErr  error
		wantRef  string
	}{
		{
			name:     "provider not configured",
			provider: nil,
			wantErr:  ErrLocationProviderUnavailable,
		},
		{
			name:     "provider returns payload",
			provider: &stubLocationProvider{resolved: resolved},
			wantRef:  "abc",
		},
		{
			name:     "provider returns error",
			provider: &stubLocationProvider{resolveErr: errResolveFail},
			wantErr:  errResolveFail,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewDefaultService(Repos{}, tt.provider, configMock())
			got, err := svc.ResolveLocation(ctx, "ref")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil || got.ProviderRef != tt.wantRef {
				t.Fatalf("unexpected resolved address: %+v", got)
			}
		})
	}
}

func TestDefaultServiceNormalizeLocation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	resolved := &ResolvedAddress{
		Formatted: "123 Main St, Springfield",
		Address: Address{
			Street:     "123 Main St",
			Number:     "123",
			City:       "Springfield",
			State:      "IL",
			PostalCode: "62704",
			Country:    "USA",
		},
		Coordinates: Coordinates{
			Latitude:  12.34,
			Longitude: 56.78,
		},
		Provider:    ProviderLocationIQ,
		ProviderRef: "ref-123",
		ProviderURL: "https://provider.example/ref-123",
		Raw: map[string]any{
			"display_name": "123 Main St, Springfield, USA",
		},
	}

	tests := []struct {
		name      string
		provider  LocationProvider
		req       NormalizeLocationRequest
		wantErr   error
		checkFunc func(t *testing.T, result *NormalizedLocation)
	}{
		{
			name:     "provider not configured",
			provider: nil,
			req: NormalizeLocationRequest{
				ProviderRef: "ref-123",
			},
			wantErr: ErrLocationProviderUnavailable,
		},
		{
			name: "resolve error",
			provider: &stubLocationProvider{
				resolveErr: errResolveFail,
			},
			req:     NormalizeLocationRequest{ProviderRef: "ref-123"},
			wantErr: errResolveFail,
		},
		{
			name: "normalization success",
			provider: &stubLocationProvider{
				resolved: resolved,
			},
			req: NormalizeLocationRequest{ProviderRef: "ref-123", SelectedText: "custom"},
			checkFunc: func(t *testing.T, result *NormalizedLocation) {
				t.Helper()
				if result.ProviderRef != "ref-123" {
					t.Fatalf("expected provider ref ref-123, got %s", result.ProviderRef)
				}
				if result.Street == "" || !strings.Contains(result.Street, "123") {
					t.Fatalf("expected street to include number, got %s", result.Street)
				}
				if result.SearchValue == "" {
					t.Fatalf("expected search value to be populated")
				}
				if result.RawJSON == "" {
					t.Fatalf("expected raw json to be populated")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewDefaultService(Repos{}, tt.provider, configMock())
			result, err := svc.NormalizeLocation(ctx, tt.req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func configMock() config.XParams {
	return config.NewXParams(aqm.NewNoopLogger(), &config.Config{})
}
