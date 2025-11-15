package menu

import (
	"context"
	"fmt"
	"strings"

	"github.com/appetiteclub/appetite/services/menu/internal/dictionary"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidateCreateMenuItem validates a menu item before creation
func ValidateCreateMenuItem(ctx context.Context, item *MenuItem, dictClient dictionary.Client) []ValidationError {
	var errors []ValidationError

	// Validate short code
	if item.ShortCode == "" {
		errors = append(errors, ValidationError{
			Field:   "short_code",
			Message: "short_code is required",
		})
	}

	// Validate name (at least one language)
	if len(item.Name) == 0 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "at least one name translation is required",
		})
	} else {
		for lang, name := range item.Name {
			if strings.TrimSpace(name) == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("name.%s", lang),
					Message: "name cannot be empty",
				})
			}
		}
	}

	// Validate prices
	if len(item.Prices) == 0 {
		errors = append(errors, ValidationError{
			Field:   "prices",
			Message: "at least one price is required",
		})
	} else {
		for i, price := range item.Prices {
			if price.Amount < 0 {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("prices[%d].amount", i),
					Message: "price amount cannot be negative",
				})
			}
			if price.CurrencyCode == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("prices[%d].currency_code", i),
					Message: "currency code is required",
				})
			} else if len(price.CurrencyCode) != 3 {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("prices[%d].currency_code", i),
					Message: "currency code must be 3 characters (ISO 4217)",
				})
			}
		}
	}

	// Validate portions
	for i, portion := range item.Portions {
		if len(portion.Name) == 0 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("portions[%d].name", i),
				Message: "portion name is required",
			})
		}
		if portion.PrepTime < 0 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("portions[%d].prep_time", i),
				Message: "preparation time cannot be negative",
			})
		}
		// Validate portion price overrides
		for j, price := range portion.PriceOverride {
			if price.Amount < 0 {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("portions[%d].price_override[%d].amount", i, j),
					Message: "price amount cannot be negative",
				})
			}
			if price.CurrencyCode == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("portions[%d].price_override[%d].currency_code", i, j),
					Message: "currency code is required",
				})
			} else if len(price.CurrencyCode) != 3 {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("portions[%d].price_override[%d].currency_code", i, j),
					Message: "currency code must be 3 characters (ISO 4217)",
				})
			}
		}
	}

	// Validate ingredients
	for i, ing := range item.Ingredients {
		if strings.TrimSpace(ing.Name) == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("ingredients[%d].name", i),
				Message: "ingredient name is required",
			})
		}
	}

	// Validate dictionary references if client is provided
	if dictClient != nil {
		// Validate allergens
		if len(item.Allergens) > 0 {
			if err := dictClient.EnsureAllergens(ctx, item.Allergens); err != nil {
				errors = append(errors, ValidationError{
					Field:   "allergens",
					Message: fmt.Sprintf("invalid allergen reference: %v", err),
				})
			}
		}

		// Validate dietary options
		if len(item.DietaryOptions) > 0 {
			if err := dictClient.EnsureDietaryOptions(ctx, item.DietaryOptions); err != nil {
				errors = append(errors, ValidationError{
					Field:   "dietary_options",
					Message: fmt.Sprintf("invalid dietary option reference: %v", err),
				})
			}
		}

		// Validate cuisine types
		if len(item.CuisineTypes) > 0 {
			if err := dictClient.EnsureCuisineTypes(ctx, item.CuisineTypes); err != nil {
				errors = append(errors, ValidationError{
					Field:   "cuisine_types",
					Message: fmt.Sprintf("invalid cuisine type reference: %v", err),
				})
			}
		}

		// Validate categories
		if len(item.Categories) > 0 {
			if err := dictClient.EnsureMenuCategories(ctx, item.Categories); err != nil {
				errors = append(errors, ValidationError{
					Field:   "categories",
					Message: fmt.Sprintf("invalid category reference: %v", err),
				})
			}
		}
	}

	return errors
}

// ValidateUpdateMenuItem validates a menu item before update
func ValidateUpdateMenuItem(ctx context.Context, item *MenuItem, dictClient dictionary.Client) []ValidationError {
	// Same validation as create, plus ensure ID exists
	errors := ValidateCreateMenuItem(ctx, item, dictClient)

	if item.ID.String() == "" {
		errors = append(errors, ValidationError{
			Field:   "id",
			Message: "id is required for update",
		})
	}

	return errors
}

// ValidateCreateMenu validates a menu before creation
func ValidateCreateMenu(ctx context.Context, m *Menu, dictClient dictionary.Client) []ValidationError {
	var errors []ValidationError

	// Validate name (at least one language)
	if len(m.Name) == 0 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "at least one name translation is required",
		})
	} else {
		for lang, name := range m.Name {
			if strings.TrimSpace(name) == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("name.%s", lang),
					Message: "name cannot be empty",
				})
			}
		}
	}

	// Validate version state
	if m.VersionState != "" {
		if m.VersionState != MenuVersionDraft &&
			m.VersionState != MenuVersionPublished &&
			m.VersionState != MenuVersionArchived {
			errors = append(errors, ValidationError{
				Field:   "version_state",
				Message: "version_state must be one of: draft, published, archived",
			})
		}
	}

	// Validate sections
	for i, section := range m.Sections {
		// Validate category reference
		if dictClient != nil && section.CategoryID.String() != "" {
			if err := dictClient.EnsureMenuCategory(ctx, section.CategoryID); err != nil {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("sections[%d].category_id", i),
					Message: fmt.Sprintf("invalid category reference: %v", err),
				})
			}
		}
	}

	return errors
}

// ValidateUpdateMenu validates a menu before update
func ValidateUpdateMenu(ctx context.Context, m *Menu, dictClient dictionary.Client) []ValidationError {
	// Same validation as create, plus ensure ID exists
	errors := ValidateCreateMenu(ctx, m, dictClient)

	if m.ID.String() == "" {
		errors = append(errors, ValidationError{
			Field:   "id",
			Message: "id is required for update",
		})
	}

	return errors
}
