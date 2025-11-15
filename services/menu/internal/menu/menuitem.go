package menu

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

const CurrentMenuItemSchemaVersion = 1

// MenuItem represents a dish, drink or any offerable product
type MenuItem struct {
	ID              uuid.UUID         `json:"id" bson:"_id"`
	ShortCode       string            `json:"short_code" bson:"short_code"`               // Unique within menu
	Name            map[string]string `json:"name" bson:"name"`                           // Localized names
	Description     map[string]string `json:"description" bson:"description"`             // Localized descriptions
	Prices          []Price           `json:"prices" bson:"prices"`                       // Multi-currency support
	Active          bool              `json:"active" bson:"active"`                       // Available for ordering
	Portions        []Portion         `json:"portions" bson:"portions"`                   // Multiple portion options
	Allergens       []uuid.UUID       `json:"allergens" bson:"allergens"`                 // Ref: Dictionary allergens
	DietaryOptions  []uuid.UUID       `json:"dietary_options" bson:"dietary_options"`     // Ref: Dictionary dietary
	CuisineTypes    []uuid.UUID       `json:"cuisine_types" bson:"cuisine_types"`         // Ref: Dictionary cuisine_type
	Categories      []uuid.UUID       `json:"categories" bson:"categories"`               // Ref: Dictionary menu_categories
	Tags            []string          `json:"tags" bson:"tags"`                           // Free-form tags
	Ingredients     []Ingredient      `json:"ingredients" bson:"ingredients"`             // Main ingredients
	Images          []MediaReference  `json:"images" bson:"images"`                       // Media Service references
	VisibilityRules VisibilityRules   `json:"visibility_rules" bson:"visibility_rules"`   // Time-based visibility
	DisplayOrder    int               `json:"display_order" bson:"display_order"`         // Ordering within menu
	SchemaVersion   int               `json:"schema_version" bson:"schema_version"`       // Model versioning
	CreatedAt       time.Time         `json:"created_at" bson:"created_at"`
	CreatedBy       string            `json:"created_by" bson:"created_by"`
	UpdatedAt       time.Time         `json:"updated_at" bson:"updated_at"`
	UpdatedBy       string            `json:"updated_by" bson:"updated_by"`
}

// Price represents a price in a specific currency
type Price struct {
	Amount       float64 `json:"amount" bson:"amount"`
	CurrencyCode string  `json:"currency_code" bson:"currency_code"` // ISO 4217
}

// Portion represents a serving size option for a menu item
type Portion struct {
	ID                  uuid.UUID         `json:"id" bson:"id"`
	Name                map[string]string `json:"name" bson:"name"` // Localized portion name
	SizeInfo            string            `json:"size_info,omitempty" bson:"size_info,omitempty"`
	Unit                string            `json:"unit,omitempty" bson:"unit,omitempty"`
	PriceOverride       []Price           `json:"price_override,omitempty" bson:"price_override,omitempty"` // Override base item price
	PrepTime            int               `json:"prep_time" bson:"prep_time"`                                // Estimated prep time in minutes
	Active              bool              `json:"active" bson:"active"`
	SchemaVersion       int               `json:"schema_version" bson:"schema_version"`
}

// Ingredient represents a simple ingredient definition
type Ingredient struct {
	Name     string `json:"name" bson:"name"`
	Quantity string `json:"quantity,omitempty" bson:"quantity,omitempty"`
	Unit     string `json:"unit,omitempty" bson:"unit,omitempty"`
	Notes    string `json:"notes,omitempty" bson:"notes,omitempty"`
}

// MediaReference references images in the Media Service
type MediaReference struct {
	MediaID      uuid.UUID         `json:"media_id" bson:"media_id"`
	AltText      map[string]string `json:"alt_text" bson:"alt_text"` // Localized alt text
	DisplayOrder int               `json:"display_order" bson:"display_order"`
}

// VisibilityRules defines when a menu item is visible
type VisibilityRules struct {
	TimeOfDay   []TimeWindow `json:"time_of_day,omitempty" bson:"time_of_day,omitempty"`     // e.g., "08:00-11:00" for breakfast
	DaysOfWeek  []int        `json:"days_of_week,omitempty" bson:"days_of_week,omitempty"`   // 0=Sunday, 6=Saturday
	DateRanges  []DateRange  `json:"date_ranges,omitempty" bson:"date_ranges,omitempty"`     // Special seasonal items
}

// TimeWindow represents a time range during the day
type TimeWindow struct {
	Start string `json:"start" bson:"start"` // Format: "HH:MM"
	End   string `json:"end" bson:"end"`     // Format: "HH:MM"
}

// DateRange represents a date range
type DateRange struct {
	Start time.Time `json:"start" bson:"start"`
	End   time.Time `json:"end" bson:"end"`
}

// EnsureID generates a new UUID if ID is nil
func (m *MenuItem) EnsureID() {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	// Ensure portions have IDs
	for i := range m.Portions {
		if m.Portions[i].ID == uuid.Nil {
			m.Portions[i].ID = uuid.New()
		}
	}
}

// GetID returns the menu item ID
func (m *MenuItem) GetID() uuid.UUID {
	return m.ID
}

// ResourceType returns the resource type for URL generation
func (m *MenuItem) ResourceType() string {
	return "menu/item"
}

// BeforeCreate sets up the menu item before creation
func (m *MenuItem) BeforeCreate() {
	m.EnsureID()
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
	if m.SchemaVersion == 0 {
		m.SchemaVersion = CurrentMenuItemSchemaVersion
	}
	if m.Active == false && m.CreatedAt.IsZero() {
		m.Active = true
	}
	// Initialize maps if nil
	if m.Name == nil {
		m.Name = make(map[string]string)
	}
	if m.Description == nil {
		m.Description = make(map[string]string)
	}
}

// BeforeUpdate updates the timestamp
func (m *MenuItem) BeforeUpdate() {
	m.UpdatedAt = time.Now()
}

// MarshalBSON custom BSON marshaling for UUID handling
func (m *MenuItem) MarshalBSON() ([]byte, error) {
	type Alias MenuItem

	// Convert portions with UUIDs to strings
	portions := make([]bson.M, len(m.Portions))
	for i, p := range m.Portions {
		portions[i] = bson.M{
			"id":              p.ID.String(),
			"name":            p.Name,
			"size_info":       p.SizeInfo,
			"unit":            p.Unit,
			"price_override":  p.PriceOverride,
			"prep_time":       p.PrepTime,
			"active":          p.Active,
			"schema_version":  p.SchemaVersion,
		}
	}

	// Convert allergens
	allergens := make([]string, len(m.Allergens))
	for i, a := range m.Allergens {
		allergens[i] = a.String()
	}

	// Convert dietary options
	dietary := make([]string, len(m.DietaryOptions))
	for i, d := range m.DietaryOptions {
		dietary[i] = d.String()
	}

	// Convert cuisine types
	cuisines := make([]string, len(m.CuisineTypes))
	for i, c := range m.CuisineTypes {
		cuisines[i] = c.String()
	}

	// Convert categories
	categories := make([]string, len(m.Categories))
	for i, c := range m.Categories {
		categories[i] = c.String()
	}

	// Convert media references
	images := make([]bson.M, len(m.Images))
	for i, img := range m.Images {
		images[i] = bson.M{
			"media_id":      img.MediaID.String(),
			"alt_text":      img.AltText,
			"display_order": img.DisplayOrder,
		}
	}

	return bson.Marshal(bson.M{
		"_id":              m.ID.String(),
		"short_code":       m.ShortCode,
		"name":             m.Name,
		"description":      m.Description,
		"prices":           m.Prices,
		"active":           m.Active,
		"portions":         portions,
		"allergens":        allergens,
		"dietary_options":  dietary,
		"cuisine_types":    cuisines,
		"categories":       categories,
		"tags":             m.Tags,
		"ingredients":      m.Ingredients,
		"images":           images,
		"visibility_rules": m.VisibilityRules,
		"display_order":    m.DisplayOrder,
		"schema_version":   m.SchemaVersion,
		"created_at":       m.CreatedAt,
		"created_by":       m.CreatedBy,
		"updated_at":       m.UpdatedAt,
		"updated_by":       m.UpdatedBy,
	})
}

// UnmarshalBSON custom BSON unmarshaling for UUID handling
func (m *MenuItem) UnmarshalBSON(data []byte) error {
	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		return err
	}

	// Parse ID
	if idStr, ok := doc["_id"].(string); ok && idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return fmt.Errorf("invalid UUID format for _id: %w", err)
		}
		m.ID = id
	}

	if v, ok := doc["short_code"].(string); ok {
		m.ShortCode = v
	}

	// Parse name map
	if nameMap, ok := doc["name"].(bson.M); ok {
		m.Name = make(map[string]string)
		for k, v := range nameMap {
			if str, ok := v.(string); ok {
				m.Name[k] = str
			}
		}
	}

	// Parse description map
	if descMap, ok := doc["description"].(bson.M); ok {
		m.Description = make(map[string]string)
		for k, v := range descMap {
			if str, ok := v.(string); ok {
				m.Description[k] = str
			}
		}
	}

	// Parse prices
	if pricesArr, ok := doc["prices"].(bson.A); ok {
		m.Prices = make([]Price, len(pricesArr))
		for i, p := range pricesArr {
			if priceMap, ok := p.(bson.M); ok {
				if amount, ok := priceMap["amount"].(float64); ok {
					m.Prices[i].Amount = amount
				}
				if currency, ok := priceMap["currency_code"].(string); ok {
					m.Prices[i].CurrencyCode = currency
				}
			}
		}
	}

	if v, ok := doc["active"].(bool); ok {
		m.Active = v
	}

	// Parse portions
	if portionsArr, ok := doc["portions"].(bson.A); ok {
		m.Portions = make([]Portion, len(portionsArr))
		for i, p := range portionsArr {
			if portionMap, ok := p.(bson.M); ok {
				if idStr, ok := portionMap["id"].(string); ok {
					id, _ := uuid.Parse(idStr)
					m.Portions[i].ID = id
				}

				if nameMap, ok := portionMap["name"].(bson.M); ok {
					m.Portions[i].Name = make(map[string]string)
					for k, v := range nameMap {
						if str, ok := v.(string); ok {
							m.Portions[i].Name[k] = str
						}
					}
				}

				if v, ok := portionMap["size_info"].(string); ok {
					m.Portions[i].SizeInfo = v
				}
				if v, ok := portionMap["unit"].(string); ok {
					m.Portions[i].Unit = v
				}

				if pricesArr, ok := portionMap["price_override"].(bson.A); ok {
					m.Portions[i].PriceOverride = make([]Price, len(pricesArr))
					for j, pr := range pricesArr {
						if priceMap, ok := pr.(bson.M); ok {
							if amount, ok := priceMap["amount"].(float64); ok {
								m.Portions[i].PriceOverride[j].Amount = amount
							}
							if currency, ok := priceMap["currency_code"].(string); ok {
								m.Portions[i].PriceOverride[j].CurrencyCode = currency
							}
						}
					}
				}

				if v, ok := portionMap["prep_time"].(int32); ok {
					m.Portions[i].PrepTime = int(v)
				} else if v, ok := portionMap["prep_time"].(int64); ok {
					m.Portions[i].PrepTime = int(v)
				}

				if v, ok := portionMap["active"].(bool); ok {
					m.Portions[i].Active = v
				}

				if v, ok := portionMap["schema_version"].(int32); ok {
					m.Portions[i].SchemaVersion = int(v)
				} else if v, ok := portionMap["schema_version"].(int64); ok {
					m.Portions[i].SchemaVersion = int(v)
				}
			}
		}
	}

	// Parse allergens (UUID array)
	if allergensArr, ok := doc["allergens"].(bson.A); ok {
		m.Allergens = make([]uuid.UUID, 0, len(allergensArr))
		for _, a := range allergensArr {
			if str, ok := a.(string); ok {
				if id, err := uuid.Parse(str); err == nil {
					m.Allergens = append(m.Allergens, id)
				}
			}
		}
	}

	// Parse dietary options
	if dietaryArr, ok := doc["dietary_options"].(bson.A); ok {
		m.DietaryOptions = make([]uuid.UUID, 0, len(dietaryArr))
		for _, d := range dietaryArr {
			if str, ok := d.(string); ok {
				if id, err := uuid.Parse(str); err == nil {
					m.DietaryOptions = append(m.DietaryOptions, id)
				}
			}
		}
	}

	// Parse cuisine types
	if cuisineArr, ok := doc["cuisine_types"].(bson.A); ok {
		m.CuisineTypes = make([]uuid.UUID, 0, len(cuisineArr))
		for _, c := range cuisineArr {
			if str, ok := c.(string); ok {
				if id, err := uuid.Parse(str); err == nil {
					m.CuisineTypes = append(m.CuisineTypes, id)
				}
			}
		}
	}

	// Parse categories
	if categoriesArr, ok := doc["categories"].(bson.A); ok {
		m.Categories = make([]uuid.UUID, 0, len(categoriesArr))
		for _, c := range categoriesArr {
			if str, ok := c.(string); ok {
				if id, err := uuid.Parse(str); err == nil {
					m.Categories = append(m.Categories, id)
				}
			}
		}
	}

	// Parse tags
	if tagsArr, ok := doc["tags"].(bson.A); ok {
		m.Tags = make([]string, 0, len(tagsArr))
		for _, t := range tagsArr {
			if str, ok := t.(string); ok {
				m.Tags = append(m.Tags, str)
			}
		}
	}

	// Parse ingredients
	if ingredientsArr, ok := doc["ingredients"].(bson.A); ok {
		m.Ingredients = make([]Ingredient, len(ingredientsArr))
		for i, ing := range ingredientsArr {
			if ingMap, ok := ing.(bson.M); ok {
				if v, ok := ingMap["name"].(string); ok {
					m.Ingredients[i].Name = v
				}
				if v, ok := ingMap["quantity"].(string); ok {
					m.Ingredients[i].Quantity = v
				}
				if v, ok := ingMap["unit"].(string); ok {
					m.Ingredients[i].Unit = v
				}
				if v, ok := ingMap["notes"].(string); ok {
					m.Ingredients[i].Notes = v
				}
			}
		}
	}

	// Parse images
	if imagesArr, ok := doc["images"].(bson.A); ok {
		m.Images = make([]MediaReference, len(imagesArr))
		for i, img := range imagesArr {
			if imgMap, ok := img.(bson.M); ok {
				if idStr, ok := imgMap["media_id"].(string); ok {
					id, _ := uuid.Parse(idStr)
					m.Images[i].MediaID = id
				}

				if altTextMap, ok := imgMap["alt_text"].(bson.M); ok {
					m.Images[i].AltText = make(map[string]string)
					for k, v := range altTextMap {
						if str, ok := v.(string); ok {
							m.Images[i].AltText[k] = str
						}
					}
				}

				if v, ok := imgMap["display_order"].(int32); ok {
					m.Images[i].DisplayOrder = int(v)
				} else if v, ok := imgMap["display_order"].(int64); ok {
					m.Images[i].DisplayOrder = int(v)
				}
			}
		}
	}

	// Parse visibility rules
	if visMap, ok := doc["visibility_rules"].(bson.M); ok {
		// Parse time of day
		if todArr, ok := visMap["time_of_day"].(bson.A); ok {
			m.VisibilityRules.TimeOfDay = make([]TimeWindow, len(todArr))
			for i, tw := range todArr {
				if twMap, ok := tw.(bson.M); ok {
					if v, ok := twMap["start"].(string); ok {
						m.VisibilityRules.TimeOfDay[i].Start = v
					}
					if v, ok := twMap["end"].(string); ok {
						m.VisibilityRules.TimeOfDay[i].End = v
					}
				}
			}
		}

		// Parse days of week
		if dowArr, ok := visMap["days_of_week"].(bson.A); ok {
			m.VisibilityRules.DaysOfWeek = make([]int, len(dowArr))
			for i, d := range dowArr {
				if v, ok := d.(int32); ok {
					m.VisibilityRules.DaysOfWeek[i] = int(v)
				} else if v, ok := d.(int64); ok {
					m.VisibilityRules.DaysOfWeek[i] = int(v)
				}
			}
		}

		// Parse date ranges
		if drArr, ok := visMap["date_ranges"].(bson.A); ok {
			m.VisibilityRules.DateRanges = make([]DateRange, len(drArr))
			for i, dr := range drArr {
				if drMap, ok := dr.(bson.M); ok {
					if v, ok := drMap["start"].(time.Time); ok {
						m.VisibilityRules.DateRanges[i].Start = v
					}
					if v, ok := drMap["end"].(time.Time); ok {
						m.VisibilityRules.DateRanges[i].End = v
					}
				}
			}
		}
	}

	if v, ok := doc["display_order"].(int32); ok {
		m.DisplayOrder = int(v)
	} else if v, ok := doc["display_order"].(int64); ok {
		m.DisplayOrder = int(v)
	}

	if v, ok := doc["schema_version"].(int32); ok {
		m.SchemaVersion = int(v)
	} else if v, ok := doc["schema_version"].(int64); ok {
		m.SchemaVersion = int(v)
	}

	if v, ok := doc["created_at"].(time.Time); ok {
		m.CreatedAt = v
	}
	if v, ok := doc["created_by"].(string); ok {
		m.CreatedBy = v
	}
	if v, ok := doc["updated_at"].(time.Time); ok {
		m.UpdatedAt = v
	}
	if v, ok := doc["updated_by"].(string); ok {
		m.UpdatedBy = v
	}

	return nil
}
