package menu

import (
	"context"
	"fmt"
	"time"

	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/seed"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Seeds returns all seeds for the Menu service
func Seeds(db *mongo.Database) []seed.Seed {
	return []seed.Seed{
		{
			ID:          "2025-11-15_menu_dictionary",
			Description: "Load menu-related dictionary entries",
			Run: func(ctx context.Context) error {
				return seedMenuDictionary(ctx, db)
			},
		},
		{
			ID:          "2025-11-18_menu_sample_items",
			Description: "Seed representative menu items for kitchen, bar, and direct service",
			Run: func(ctx context.Context) error {
				return seedSampleMenuItems(ctx, db)
			},
		},
	}
}

// seedMenuDictionary creates dictionary sets and options for menu management
// Note: This assumes the dictionary service database is accessible
func seedMenuDictionary(ctx context.Context, db *mongo.Database) error {
	// Get reference to dictionary database
	// In production, this would be a separate database connection
	dictDB := db.Client().Database("appetite_dictionary")
	setsCollection := dictDB.Collection("sets")
	optionsCollection := dictDB.Collection("options")

	now := time.Now()

	// ========================================
	// ALLERGENS
	// ========================================
	setID_allergens_en := uuid.New().String()
	_, _ = setsCollection.UpdateOne(ctx,
		bson.M{"name": "allergens", "locale": "en"},
		bson.M{"$setOnInsert": bson.M{
			"_id":         setID_allergens_en,
			"name":        "allergens",
			"locale":      "en",
			"label":       "Allergens",
			"description": "Common food allergens",
			"active":      true,
			"created_at":  now,
			"updated_at":  now,
			"created_by":  "system",
			"updated_by":  "system",
		}},
		options.Update().SetUpsert(true))

	allergens := []struct {
		key   string
		label string
		order int
	}{
		{"peanuts", "Peanuts", 1},
		{"tree_nuts", "Tree Nuts", 2},
		{"milk", "Milk", 3},
		{"eggs", "Eggs", 4},
		{"wheat", "Wheat", 5},
		{"soy", "Soy", 6},
		{"fish", "Fish", 7},
		{"shellfish", "Shellfish", 8},
		{"sesame", "Sesame", 9},
		{"gluten", "Gluten", 10},
		{"celery", "Celery", 11},
		{"mustard", "Mustard", 12},
		{"sulfites", "Sulfites", 13},
		{"lupin", "Lupin", 14},
		{"molluscs", "Molluscs", 15},
	}

	for _, a := range allergens {
		_, _ = optionsCollection.UpdateOne(ctx,
			bson.M{"set_id": setID_allergens_en, "key": a.key},
			bson.M{"$setOnInsert": bson.M{
				"_id":        uuid.New().String(),
				"set_id":     setID_allergens_en,
				"locale":     "en",
				"short_code": a.key,
				"key":        a.key,
				"label":      a.label,
				"value":      a.key,
				"order":      a.order,
				"active":     true,
				"created_at": now,
				"updated_at": now,
				"created_by": "system",
				"updated_by": "system",
			}},
			options.Update().SetUpsert(true))
	}

	// ========================================
	// DIETARY OPTIONS
	// ========================================
	setID_dietary_en := uuid.New().String()
	_, _ = setsCollection.UpdateOne(ctx,
		bson.M{"name": "dietary", "locale": "en"},
		bson.M{"$setOnInsert": bson.M{
			"_id":         setID_dietary_en,
			"name":        "dietary",
			"locale":      "en",
			"label":       "Dietary Options",
			"description": "Dietary preferences and restrictions",
			"active":      true,
			"created_at":  now,
			"updated_at":  now,
			"created_by":  "system",
			"updated_by":  "system",
		}},
		options.Update().SetUpsert(true))

	dietaryOptions := []struct {
		key   string
		label string
		order int
	}{
		{"vegetarian", "Vegetarian", 1},
		{"vegan", "Vegan", 2},
		{"gluten_free", "Gluten-Free", 3},
		{"dairy_free", "Dairy-Free", 4},
		{"halal", "Halal", 5},
		{"kosher", "Kosher", 6},
		{"paleo", "Paleo", 7},
		{"keto", "Keto", 8},
		{"low_carb", "Low Carb", 9},
		{"sugar_free", "Sugar-Free", 10},
		{"organic", "Organic", 11},
		{"raw", "Raw", 12},
	}

	for _, d := range dietaryOptions {
		_, _ = optionsCollection.UpdateOne(ctx,
			bson.M{"set_id": setID_dietary_en, "key": d.key},
			bson.M{"$setOnInsert": bson.M{
				"_id":        uuid.New().String(),
				"set_id":     setID_dietary_en,
				"locale":     "en",
				"short_code": d.key,
				"key":        d.key,
				"label":      d.label,
				"value":      d.key,
				"order":      d.order,
				"active":     true,
				"created_at": now,
				"updated_at": now,
				"created_by": "system",
				"updated_by": "system",
			}},
			options.Update().SetUpsert(true))
	}

	// ========================================
	// CUISINE TYPES
	// ========================================
	setID_cuisine_en := uuid.New().String()
	_, _ = setsCollection.UpdateOne(ctx,
		bson.M{"name": "cuisine_type", "locale": "en"},
		bson.M{"$setOnInsert": bson.M{
			"_id":         setID_cuisine_en,
			"name":        "cuisine_type",
			"locale":      "en",
			"label":       "Cuisine Types",
			"description": "Types of cuisine",
			"active":      true,
			"created_at":  now,
			"updated_at":  now,
			"created_by":  "system",
			"updated_by":  "system",
		}},
		options.Update().SetUpsert(true))

	cuisineTypes := []struct {
		key   string
		label string
		order int
	}{
		{"italian", "Italian", 1},
		{"mexican", "Mexican", 2},
		{"chinese", "Chinese", 3},
		{"japanese", "Japanese", 4},
		{"thai", "Thai", 5},
		{"indian", "Indian", 6},
		{"french", "French", 7},
		{"spanish", "Spanish", 8},
		{"mediterranean", "Mediterranean", 9},
		{"american", "American", 10},
		{"greek", "Greek", 11},
		{"middle_eastern", "Middle Eastern", 12},
		{"korean", "Korean", 13},
		{"vietnamese", "Vietnamese", 14},
		{"brazilian", "Brazilian", 15},
	}

	for _, c := range cuisineTypes {
		_, _ = optionsCollection.UpdateOne(ctx,
			bson.M{"set_id": setID_cuisine_en, "key": c.key},
			bson.M{"$setOnInsert": bson.M{
				"_id":        uuid.New().String(),
				"set_id":     setID_cuisine_en,
				"locale":     "en",
				"short_code": c.key,
				"key":        c.key,
				"label":      c.label,
				"value":      c.key,
				"order":      c.order,
				"active":     true,
				"created_at": now,
				"updated_at": now,
				"created_by": "system",
				"updated_by": "system",
			}},
			options.Update().SetUpsert(true))
	}

	// ========================================
	// MENU CATEGORIES
	// ========================================
	setID_categories_en := uuid.New().String()
	_, _ = setsCollection.UpdateOne(ctx,
		bson.M{"name": "menu_categories", "locale": "en"},
		bson.M{"$setOnInsert": bson.M{
			"_id":         setID_categories_en,
			"name":        "menu_categories",
			"locale":      "en",
			"label":       "Menu Categories",
			"description": "Categories for organizing menu items",
			"active":      true,
			"created_at":  now,
			"updated_at":  now,
			"created_by":  "system",
			"updated_by":  "system",
		}},
		options.Update().SetUpsert(true))

	menuCategories := []struct {
		key   string
		label string
		order int
	}{
		{"appetizers", "Appetizers", 1},
		{"soups", "Soups", 2},
		{"salads", "Salads", 3},
		{"main_courses", "Main Courses", 4},
		{"pasta", "Pasta", 5},
		{"seafood", "Seafood", 6},
		{"meat", "Meat", 7},
		{"poultry", "Poultry", 8},
		{"vegetarian", "Vegetarian", 9},
		{"sides", "Sides", 10},
		{"desserts", "Desserts", 11},
		{"beverages", "Beverages", 12},
		{"coffee_tea", "Coffee & Tea", 13},
		{"alcohol", "Alcohol", 14},
		{"breakfast", "Breakfast", 15},
		{"brunch", "Brunch", 16},
		{"lunch", "Lunch", 17},
		{"dinner", "Dinner", 18},
	}

	for _, cat := range menuCategories {
		_, _ = optionsCollection.UpdateOne(ctx,
			bson.M{"set_id": setID_categories_en, "key": cat.key},
			bson.M{"$setOnInsert": bson.M{
				"_id":        uuid.New().String(),
				"set_id":     setID_categories_en,
				"locale":     "en",
				"short_code": cat.key,
				"key":        cat.key,
				"label":      cat.label,
				"value":      cat.key,
				"order":      cat.order,
				"active":     true,
				"created_at": now,
				"updated_at": now,
				"created_by": "system",
				"updated_by": "system",
			}},
			options.Update().SetUpsert(true))
	}

	return nil
}

func seedSampleMenuItems(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("menu_items")
	now := time.Now()
	items := []struct {
		ShortCode   string
		Name        string
		Description string
		Price       float64
		Station     string
		Class       string
	}{
		// Kitchen items
		{"BURG-001", "Smash Burger", "Double smash patties with cheddar and pickles", 14.50, "kitchen", "entree"},
		{"PAST-001", "Truffle Pasta", "Handmade tagliatelle with truffle cream", 18.00, "kitchen", "entree"},
		{"FISH-001", "Seared Salmon", "Atlantic salmon with citrus glaze", 21.00, "kitchen", "entree"},
		{"STEAK-001", "Bistro Steak", "Sirloin with herb butter and fries", 24.00, "kitchen", "entree"},
		{"TACO-001", "Baja Fish Tacos", "Beer-battered cod with chipotle slaw", 16.00, "kitchen", "shareable"},
		{"VEG-001", "Harvest Bowl", "Roasted vegetables, quinoa, tahini", 15.00, "kitchen", "entree"},
		{"BBQ-001", "Smokehouse Ribs", "Slow cooked pork ribs with house sauce", 22.00, "kitchen", "entree"},
		{"PIZ-001", "Margherita Pizza", "San Marzano tomatoes, mozzarella, basil", 17.00, "kitchen", "entree"},
		{"SAL-001", "Citrus Kale Salad", "Baby kale, grapefruit, toasted seeds", 13.00, "kitchen", "starter"},

		// Dessert items (dessert station)
		{"DESS-001", "Chocolate Lava Cake", "Warm cake with vanilla gelato", 10.00, "dessert", "dessert"},
		{"DESS-002", "Classic Cheesecake", "NY style cheesecake, berry compote", 9.00, "dessert", "dessert"},
		{"DESS-003", "Tiramisu", "Coffee-soaked ladyfingers, mascarpone", 9.50, "dessert", "dessert"},
		{"DESS-004", "Crème Brûlée", "Vanilla custard with caramelized sugar", 10.50, "dessert", "dessert"},

		// Bar items (cocktails, draft beer)
		{"DRK-OLD", "Smoked Old Fashioned", "Rye whiskey, bitters, orange peel", 13.00, "bar", "cocktail"},
		{"DRK-MARG", "Spicy Margarita", "Reposado tequila, jalapeño cordial", 12.00, "bar", "cocktail"},
		{"DRK-ESP", "Espresso Martini", "Vodka, espresso, coffee liqueur", 12.50, "bar", "cocktail"},
		{"DRK-SPRZ", "Aperol Spritz", "Aperol, prosecco, soda", 11.00, "bar", "cocktail"},
		{"DRK-GTON", "Garden Gin & Tonic", "Botanical gin, tonic, herbs", 11.50, "bar", "cocktail"},
		{"BEER-IPA", "West Coast IPA", "16oz draft craft IPA", 8.50, "bar", "beer"},
		{"BEER-LAGER", "Czech Pilsner", "16oz draft lager", 7.50, "bar", "beer"},

		// Coffee items (coffee station)
		{"COFF-ESP", "Espresso", "Double shot espresso", 3.50, "coffee", "coffee"},
		{"COFF-CAP", "Cappuccino", "Espresso with steamed milk and foam", 4.50, "coffee", "coffee"},
		{"COFF-LATTE", "Caffè Latte", "Espresso with steamed milk", 5.00, "coffee", "coffee"},
		{"COFF-MACH", "Macchiato", "Espresso with foam", 4.00, "coffee", "coffee"},
		{"COFF-AMER", "Americano", "Espresso with hot water", 4.00, "coffee", "coffee"},

		// Direct service (no production needed)
		{"BEV-SPRK", "Sparkling Water", "Chilled bottled sparkling water", 4.50, "direct", "beverage"},
		{"BEV-COLA", "Bottled Cola", "12oz glass bottle cola", 4.00, "direct", "beverage"},
		{"BEV-ICED", "House Iced Tea", "Fresh brewed black tea, lemon", 5.00, "direct", "beverage"},
		{"BEV-LEMO", "Cucumber Lemonade", "Pressed lemons, cucumber syrup", 6.00, "direct", "beverage"},

		// Other/unusual items (atypical station assignments)
		{"SPEC-WINE", "Wine Pairing", "Sommelier wine selection", 18.00, "other", "beverage"},
		{"SPEC-CHEESE", "Cheese Board", "Curated selection with preserves", 16.00, "other", "shareable"},
	}

	for idx, item := range items {
		doc := bson.M{
			"_id":             uuid.New().String(),
			"short_code":      item.ShortCode,
			"name":            bson.M{"en": item.Name},
			"description":     bson.M{"en": item.Description},
			"prices":          []bson.M{{"amount": item.Price, "currency_code": "USD"}},
			"active":          true,
			"portions":        []bson.M{},
			"allergens":       []string{},
			"dietary_options": []string{},
			"cuisine_types":   []string{},
			"categories":      []string{},
			"tags": []string{
				fmt.Sprintf("station:%s", item.Station),
				fmt.Sprintf("class:%s", item.Class),
			},
			"ingredients":      []bson.M{},
			"images":           []bson.M{},
			"visibility_rules": bson.M{},
			"display_order":    idx + 1,
			"schema_version":   CurrentMenuItemSchemaVersion,
			"created_at":       now,
			"created_by":       "seed",
			"updated_at":       now,
			"updated_by":       "seed",
		}

		filter := bson.M{"short_code": item.ShortCode}
		update := bson.M{"$setOnInsert": doc}
		if _, err := collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true)); err != nil {
			return fmt.Errorf("seed menu item %s: %w", item.ShortCode, err)
		}
	}

	return nil
}

// SeedingFunc returns a function for running seeds during service startup
func SeedingFunc(appName string, dbFn func() *mongo.Database, logger apt.Logger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		logger.Info("Applying menu service database seeds...")
		db := dbFn()
		tracker := seed.NewMongoTracker(db)
		seeds := Seeds(db)
		if err := seed.Apply(ctx, tracker, seeds, appName); err != nil {
			return fmt.Errorf("apply seeds: %w", err)
		}
		logger.Info("Menu service database seeds applied successfully")
		return nil
	}
}
