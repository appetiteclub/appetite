package menu

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

const CurrentMenuSchemaVersion = 1

// MenuVersionState represents the publication state of a menu
type MenuVersionState string

const (
	MenuVersionDraft     MenuVersionState = "draft"
	MenuVersionPublished MenuVersionState = "published"
	MenuVersionArchived  MenuVersionState = "archived"
)

// Menu is a container of items and combos presented to end users
type Menu struct {
	ID              uuid.UUID         `json:"id" bson:"_id"`
	Name            map[string]string `json:"name" bson:"name"`                       // Localized names
	Description     map[string]string `json:"description" bson:"description"`         // Localized descriptions
	Sections        []MenuSection     `json:"sections" bson:"sections"`               // Organized by categories
	VersionState    MenuVersionState  `json:"version_state" bson:"version_state"`     // draft/published/archived
	VisibilityRules VisibilityRules   `json:"visibility_rules" bson:"visibility_rules"` // Optional visibility windows
	DisplayOrder    int               `json:"display_order" bson:"display_order"`     // Ordering for multiple menus
	SchemaVersion   int               `json:"schema_version" bson:"schema_version"`   // Model versioning
	CreatedAt       time.Time         `json:"created_at" bson:"created_at"`
	CreatedBy       string            `json:"created_by" bson:"created_by"`
	UpdatedAt       time.Time         `json:"updated_at" bson:"updated_at"`
	UpdatedBy       string            `json:"updated_by" bson:"updated_by"`
}

// MenuSection represents a section within a menu organized by category
type MenuSection struct {
	ID           uuid.UUID   `json:"id" bson:"id"`
	CategoryID   uuid.UUID   `json:"category_id" bson:"category_id"`     // Ref: Dictionary menu_categories
	DisplayOrder int         `json:"display_order" bson:"display_order"` // Order within menu
	MenuItems    []uuid.UUID `json:"menu_items" bson:"menu_items"`       // References to MenuItem IDs
}

// EnsureID generates a new UUID if ID is nil
func (m *Menu) EnsureID() {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	// Ensure sections have IDs
	for i := range m.Sections {
		if m.Sections[i].ID == uuid.Nil {
			m.Sections[i].ID = uuid.New()
		}
	}
}

// GetID returns the menu ID
func (m *Menu) GetID() uuid.UUID {
	return m.ID
}

// ResourceType returns the resource type for URL generation
func (m *Menu) ResourceType() string {
	return "menu/menu"
}

// BeforeCreate sets up the menu before creation
func (m *Menu) BeforeCreate() {
	m.EnsureID()
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
	if m.SchemaVersion == 0 {
		m.SchemaVersion = CurrentMenuSchemaVersion
	}
	if m.VersionState == "" {
		m.VersionState = MenuVersionDraft
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
func (m *Menu) BeforeUpdate() {
	m.UpdatedAt = time.Now()
}

// MarshalBSON custom BSON marshaling for UUID handling
func (m *Menu) MarshalBSON() ([]byte, error) {
	// Convert sections
	sections := make([]bson.M, len(m.Sections))
	for i, s := range m.Sections {
		// Convert menu item IDs to strings
		menuItemIDs := make([]string, len(s.MenuItems))
		for j, itemID := range s.MenuItems {
			menuItemIDs[j] = itemID.String()
		}

		sections[i] = bson.M{
			"id":            s.ID.String(),
			"category_id":   s.CategoryID.String(),
			"display_order": s.DisplayOrder,
			"menu_items":    menuItemIDs,
		}
	}

	return bson.Marshal(bson.M{
		"_id":              m.ID.String(),
		"name":             m.Name,
		"description":      m.Description,
		"sections":         sections,
		"version_state":    string(m.VersionState),
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
func (m *Menu) UnmarshalBSON(data []byte) error {
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

	// Parse sections
	if sectionsArr, ok := doc["sections"].(bson.A); ok {
		m.Sections = make([]MenuSection, len(sectionsArr))
		for i, s := range sectionsArr {
			if sectionMap, ok := s.(bson.M); ok {
				if idStr, ok := sectionMap["id"].(string); ok {
					id, _ := uuid.Parse(idStr)
					m.Sections[i].ID = id
				}

				if catIDStr, ok := sectionMap["category_id"].(string); ok {
					catID, _ := uuid.Parse(catIDStr)
					m.Sections[i].CategoryID = catID
				}

				if v, ok := sectionMap["display_order"].(int32); ok {
					m.Sections[i].DisplayOrder = int(v)
				} else if v, ok := sectionMap["display_order"].(int64); ok {
					m.Sections[i].DisplayOrder = int(v)
				}

				// Parse menu item IDs
				if itemsArr, ok := sectionMap["menu_items"].(bson.A); ok {
					m.Sections[i].MenuItems = make([]uuid.UUID, 0, len(itemsArr))
					for _, itemID := range itemsArr {
						if itemIDStr, ok := itemID.(string); ok {
							if id, err := uuid.Parse(itemIDStr); err == nil {
								m.Sections[i].MenuItems = append(m.Sections[i].MenuItems, id)
							}
						}
					}
				}
			}
		}
	}

	if v, ok := doc["version_state"].(string); ok {
		m.VersionState = MenuVersionState(v)
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
