# Menu Service API

REST API for managing restaurant menus, menu items, portions, and categorizations.

**Base URL:** `http://localhost:8088`

---

## Menu Items

### Create Menu Item

Create a new menu item (dish, drink, or product).

**Endpoint:** `POST /menu/items`

**Request Body:**
```json
{
  "short_code": "PASTA-CARB",
  "name": {
    "en": "Spaghetti Carbonara",
    "es": "Espagueti a la Carbonara"
  },
  "description": {
    "en": "Classic Italian pasta with eggs, pecorino cheese, guanciale, and black pepper",
    "es": "Pasta italiana clásica con huevos, queso pecorino, guanciale y pimienta negra"
  },
  "prices": [
    {
      "amount": 14.50,
      "currency_code": "USD"
    },
    {
      "amount": 13.00,
      "currency_code": "EUR"
    }
  ],
  "active": true,
  "portions": [
    {
      "name": {
        "en": "Regular",
        "es": "Normal"
      },
      "size_info": "350g",
      "prep_time": 15,
      "active": true
    },
    {
      "name": {
        "en": "Large",
        "es": "Grande"
      },
      "size_info": "500g",
      "price_override": [
        {
          "amount": 18.50,
          "currency_code": "USD"
        }
      ],
      "prep_time": 18,
      "active": true
    }
  ],
  "allergens": ["<uuid-of-eggs>", "<uuid-of-gluten>", "<uuid-of-milk>"],
  "dietary_options": [],
  "cuisine_types": ["<uuid-of-italian>"],
  "categories": ["<uuid-of-pasta>", "<uuid-of-main-courses>"],
  "tags": ["signature", "popular"],
  "ingredients": [
    {
      "name": "Spaghetti",
      "quantity": "300",
      "unit": "g"
    },
    {
      "name": "Eggs",
      "quantity": "3",
      "unit": "units"
    },
    {
      "name": "Guanciale",
      "quantity": "100",
      "unit": "g"
    },
    {
      "name": "Pecorino Romano",
      "quantity": "50",
      "unit": "g"
    }
  ],
  "images": [
    {
      "media_id": "<uuid-from-media-service>",
      "alt_text": {
        "en": "Spaghetti Carbonara on white plate",
        "es": "Espagueti a la Carbonara en plato blanco"
      },
      "display_order": 1
    }
  ],
  "visibility_rules": {
    "time_of_day": [
      {
        "start": "11:00",
        "end": "23:00"
      }
    ],
    "days_of_week": [1, 2, 3, 4, 5, 6, 0]
  },
  "display_order": 10
}
```

**Response:** `201 Created`
```json
{
  "data": {
    "id": "<uuid>",
    "short_code": "PASTA-CARB",
    "name": { ... },
    "description": { ... },
    "prices": [ ... ],
    "portions": [ ... ],
    "allergens": [ ... ],
    "schema_version": 1,
    "created_at": "2025-11-15T15:30:00Z",
    "created_by": "system",
    "updated_at": "2025-11-15T15:30:00Z",
    "updated_by": "system"
  },
  "links": [
    {
      "rel": "self",
      "href": "/menu/items/<uuid>"
    }
  ]
}
```

---

### Get Menu Item

Retrieve a specific menu item by ID.

**Endpoint:** `GET /menu/items/{id}`

**Response:** `200 OK`
```json
{
  "data": { ... },
  "links": [ ... ]
}
```

---

### Get Menu Item by Short Code

Retrieve a menu item by its unique short code.

**Endpoint:** `GET /menu/items/code/{shortCode}`

**Example:** `GET /menu/items/code/PASTA-CARB`

**Response:** `200 OK`

---

### List Menu Items

Retrieve all menu items with optional filtering.

**Endpoint:** `GET /menu/items`

**Query Parameters:**
- `active` (boolean) - Filter by active status (e.g., `?active=true`)

**Response:** `200 OK`
```json
{
  "data": [
    { ... },
    { ... }
  ],
  "meta": {
    "type": "menu/items",
    "count": 2
  }
}
```

---

### List Menu Items by Category

Retrieve menu items filtered by category.

**Endpoint:** `GET /menu/items/category/{categoryID}`

**Example:** `GET /menu/items/category/<uuid-of-pasta-category>`

**Response:** `200 OK`

---

### Update Menu Item

Update an existing menu item.

**Endpoint:** `PUT /menu/items/{id}`

**Request Body:** Same structure as Create Menu Item

**Response:** `200 OK`

---

### Delete Menu Item

Delete a menu item.

**Endpoint:** `DELETE /menu/items/{id}`

**Response:** `204 No Content`

---

## Menus

### Create Menu

Create a new menu container.

**Endpoint:** `POST /menu/menus`

**Request Body:**
```json
{
  "name": {
    "en": "Dinner Menu",
    "es": "Menú de Cena"
  },
  "description": {
    "en": "Our dinner selection available from 6 PM",
    "es": "Nuestra selección de cena disponible desde las 6 PM"
  },
  "sections": [
    {
      "category_id": "<uuid-of-appetizers>",
      "display_order": 1,
      "menu_items": [
        "<uuid-of-item-1>",
        "<uuid-of-item-2>"
      ]
    },
    {
      "category_id": "<uuid-of-main-courses>",
      "display_order": 2,
      "menu_items": [
        "<uuid-of-item-3>",
        "<uuid-of-item-4>"
      ]
    },
    {
      "category_id": "<uuid-of-desserts>",
      "display_order": 3,
      "menu_items": [
        "<uuid-of-item-5>"
      ]
    }
  ],
  "version_state": "draft",
  "visibility_rules": {
    "time_of_day": [
      {
        "start": "18:00",
        "end": "23:30"
      }
    ],
    "days_of_week": [1, 2, 3, 4, 5, 6, 0]
  },
  "display_order": 1
}
```

**Response:** `201 Created`
```json
{
  "data": {
    "id": "<uuid>",
    "name": { ... },
    "sections": [ ... ],
    "version_state": "draft",
    "schema_version": 1,
    "created_at": "2025-11-15T16:00:00Z",
    "updated_at": "2025-11-15T16:00:00Z"
  },
  "links": [ ... ]
}
```

---

### Get Menu

Retrieve a specific menu by ID.

**Endpoint:** `GET /menu/menus/{id}`

**Response:** `200 OK`

---

### List Menus

Retrieve all menus with optional filtering.

**Endpoint:** `GET /menu/menus`

**Query Parameters:**
- `published` (boolean) - Filter by published status (e.g., `?published=true`)

**Response:** `200 OK`
```json
{
  "data": [
    { ... }
  ],
  "meta": {
    "type": "menu/menus",
    "count": 1
  }
}
```

---

### Update Menu

Update an existing menu.

**Endpoint:** `PUT /menu/menus/{id}`

**Request Body:** Same structure as Create Menu

**Response:** `200 OK`

---

### Delete Menu

Delete a menu.

**Endpoint:** `DELETE /menu/menus/{id}`

**Response:** `204 No Content`

---

## Data Models

### MenuItem

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | UUID | Auto | Unique identifier |
| `short_code` | string | Yes | Unique code within menu |
| `name` | object | Yes | Localized names (lang → text) |
| `description` | object | No | Localized descriptions |
| `prices` | array | Yes | Multi-currency prices |
| `active` | boolean | No | Available for ordering (default: true) |
| `portions` | array | No | Portion options |
| `allergens` | array[UUID] | No | References to Dictionary allergens |
| `dietary_options` | array[UUID] | No | References to Dictionary dietary options |
| `cuisine_types` | array[UUID] | No | References to Dictionary cuisine types |
| `categories` | array[UUID] | No | References to Dictionary menu categories |
| `tags` | array[string] | No | Free-form tags |
| `ingredients` | array | No | Ingredient definitions |
| `images` | array | No | Media Service references |
| `visibility_rules` | object | No | Time/date-based visibility |
| `display_order` | integer | No | Sort order |
| `schema_version` | integer | Auto | Model version |
| `created_at` | timestamp | Auto | Creation timestamp |
| `created_by` | string | Auto | Creator identifier |
| `updated_at` | timestamp | Auto | Last update timestamp |
| `updated_by` | string | Auto | Last updater identifier |

### Price

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `amount` | float | Yes | Price value |
| `currency_code` | string | Yes | ISO 4217 currency code (3 chars) |

### Portion

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | UUID | Auto | Unique identifier |
| `name` | object | Yes | Localized portion names |
| `size_info` | string | No | Size description (e.g., "350g") |
| `unit` | string | No | Unit of measure |
| `price_override` | array[Price] | No | Override base item prices |
| `prep_time` | integer | No | Estimated prep time (minutes) |
| `active` | boolean | No | Portion availability |
| `schema_version` | integer | Auto | Model version |

### Ingredient

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Ingredient name |
| `quantity` | string | No | Amount |
| `unit` | string | No | Unit of measure |
| `notes` | string | No | Additional notes |

### MediaReference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `media_id` | UUID | Yes | Reference to Media Service |
| `alt_text` | object | No | Localized alt text |
| `display_order` | integer | No | Sort order for images |

### VisibilityRules

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `time_of_day` | array[TimeWindow] | No | Daily time ranges |
| `days_of_week` | array[int] | No | 0=Sunday, 6=Saturday |
| `date_ranges` | array[DateRange] | No | Seasonal availability |

### Menu

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | UUID | Auto | Unique identifier |
| `name` | object | Yes | Localized names |
| `description` | object | No | Localized descriptions |
| `sections` | array[MenuSection] | No | Organized sections |
| `version_state` | string | No | draft/published/archived |
| `visibility_rules` | VisibilityRules | No | Time-based visibility |
| `display_order` | integer | No | Sort order |
| `schema_version` | integer | Auto | Model version |
| `created_at` | timestamp | Auto | Creation timestamp |
| `created_by` | string | Auto | Creator identifier |
| `updated_at` | timestamp | Auto | Last update timestamp |
| `updated_by` | string | Auto | Last updater identifier |

### MenuSection

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | UUID | Auto | Unique identifier |
| `category_id` | UUID | Yes | Reference to Dictionary menu_categories |
| `display_order` | integer | No | Sort order within menu |
| `menu_items` | array[UUID] | No | References to MenuItem IDs |

---

## Validation Rules

### Menu Items
- `short_code` must be unique within the system
- `name` must have at least one language translation
- `prices` must have at least one price with valid currency code (ISO 4217, 3 chars)
- Price amounts cannot be negative
- Allergen, dietary, cuisine type, and category UUIDs must exist in Dictionary Service
- Portion names must be provided if portions are defined
- Ingredient names are required if ingredients are specified

### Menus
- `name` must have at least one language translation
- `version_state` must be one of: `draft`, `published`, `archived`
- Category IDs in sections must exist in Dictionary Service

---

## Error Responses

### Validation Error

**Status:** `400 Bad Request`
```json
{
  "error": "Validation failed",
  "errors": [
    {
      "field": "short_code",
      "message": "short_code is required"
    },
    {
      "field": "prices[0].currency_code",
      "message": "currency code must be 3 characters (ISO 4217)"
    }
  ]
}
```

### Not Found

**Status:** `404 Not Found`
```json
{
  "error": "Menu item not found"
}
```

### Server Error

**Status:** `500 Internal Server Error`
```json
{
  "error": "Could not create menu item"
}
```

---

## Dictionary Dependencies

The Menu Service requires the following dictionary sets to be available:

- **allergens** - Common food allergens (peanuts, tree nuts, milk, eggs, wheat, soy, fish, shellfish, etc.)
- **dietary** - Dietary preferences (vegetarian, vegan, gluten-free, halal, kosher, etc.)
- **cuisine_type** - Cuisine classifications (italian, mexican, chinese, japanese, thai, etc.)
- **menu_categories** - Menu organization (appetizers, soups, salads, main courses, pasta, desserts, beverages, etc.)

These are automatically seeded when the Menu Service starts for the first time.

---

## Health Check

**Endpoint:** `GET /health`

**Response:** `200 OK`
```json
{
  "status": "healthy",
  "service": "menu",
  "version": "0.1.0"
}
```
