package admin

import "testing"

func TestBuildNormalizedLocationCleansEncoding(t *testing.T) {
	resolved := &ResolvedAddress{
		Formatted: "Rinc\u00C3\u00B3n de Romos, Aguascalientes",
		Address: Address{
			City:    "RincÃ³n de Romos",
			Street:  "El BajÃ­o",
			Country: "mx",
		},
	}

	loc := buildNormalizedLocation(resolved, "RincÃ³n de Romos")

	if loc.City != "Rincón de Romos" {
		t.Fatalf("expected city to be decoded, got %q", loc.City)
	}

	if loc.SelectedText != "Rincón de Romos" {
		t.Fatalf("expected selected text to be decoded, got %q", loc.SelectedText)
	}

	if loc.SearchValue != "Rincón de Romos, Aguascalientes" {
		t.Fatalf("expected search value to be decoded, got %q", loc.SearchValue)
	}

	if loc.Country != "Mexico" {
		t.Fatalf("expected country to be expanded, got %q", loc.Country)
	}

	if loc.Street != "El Bajío" {
		t.Fatalf("expected street to be decoded, got %q", loc.Street)
	}
}

func TestStripDiacritics(t *testing.T) {
	input := "Łódź"
	want := "Lodz"
	got := stripDiacritics(input)
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
