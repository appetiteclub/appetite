package admin

import "testing"

func TestLocationFormModelFromPropertyCleansEncoding(t *testing.T) {
	property := &Property{
		Location: Location{
			DisplayName: "RincÃ³n de Romos",
			Region:      "Aguascalientes",
			Address: Address{
				Street:     "El BajÃ­o",
				City:       "RincÃ³n de Romos",
				Country:    "mx",
				PostalCode: "20300",
				State:      "Aguascalientes",
			},
		},
	}

	model := locationFormModelFromProperty(property)

	if model.City != "Rincón de Romos" {
		t.Fatalf("expected city to be decoded, got %q", model.City)
	}
	if model.Country != "Mexico" {
		t.Fatalf("expected country to be expanded, got %q", model.Country)
	}
	if model.Street != "El Bajío" {
		t.Fatalf("expected street to be decoded, got %q", model.Street)
	}
	if model.SearchValue != "Rincón de Romos" {
		t.Fatalf("unexpected search value: %q", model.SearchValue)
	}
}
