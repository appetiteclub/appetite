package seeding

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SeedOrders creates demo orders with realistic distribution of items
func SeedOrders(ctx context.Context, db *mongo.Database) error {
	ordersCollection := db.Collection("orders")
	itemsCollection := db.Collection("order_items")
	groupsCollection := db.Collection("order_groups")

	// Get reference to menu and table databases
	menuDB := db.Client().Database("appetite_menu")
	tableDB := db.Client().Database("appetite_table")

	now := time.Now()

	// Fetch tables that can accept orders (available, open, or reserved)
	tablesCollection := tableDB.Collection("tables")
	cursor, err := tablesCollection.Find(ctx, bson.M{"status": bson.M{"$in": []string{"available", "open", "reserved"}}})
	if err != nil {
		return fmt.Errorf("cannot fetch tables: %w", err)
	}
	var tables []struct {
		ID string `bson:"_id"`
	}
	if err := cursor.All(ctx, &tables); err != nil {
		return fmt.Errorf("cannot decode tables: %w", err)
	}
	cursor.Close(ctx)

	if len(tables) < 3 {
		return fmt.Errorf("need at least 3 tables for demo data (found %d)", len(tables))
	}

	// Note: Menu items are available but we use hardcoded data for demo consistency
	_ = menuDB

	// Demo Scenario 1: Table 1 - Couple having drinks and desserts
	order1ID := uuid.New()
	group1ID := uuid.New()

	// Parse table ID as UUID
	table1ID, _ := uuid.Parse(tables[0].ID)

	order1 := bson.M{
		"_id":        order1ID,
		"table_id":   table1ID,
		"status":     "pending",
		"created_at": now.Add(-30 * time.Minute),
		"updated_at": now.Add(-30 * time.Minute),
		"created_by": "demo-seed",
		"updated_by": "demo-seed",
	}

	_, err = ordersCollection.UpdateOne(ctx, bson.M{"_id": order1ID}, bson.M{"$setOnInsert": order1}, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("cannot create demo order 1: %w", err)
	}

	group1 := bson.M{
		"_id":        group1ID,
		"order_id":   order1ID,
		"name":       "Drinks & Desserts",
		"created_at": now.Add(-30 * time.Minute),
		"updated_at": now.Add(-30 * time.Minute),
		"created_by": "demo-seed",
		"updated_by": "demo-seed",
	}

	_, err = groupsCollection.UpdateOne(ctx, bson.M{"_id": group1ID}, bson.M{"$setOnInsert": group1}, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("cannot create demo group 1: %w", err)
	}

	// Order 1 Items
	demoItems1 := []bson.M{
		{
			"_id":                 uuid.New(),
			"order_id":            order1ID,
			"group_id":            group1ID,
			"dish_name":           "Aperol Spritz",
			"category":            "cocktail",
			"quantity":            1,
			"price":               11.0,
			"status":              "ready",
			"production_station":  "bar",
			"requires_production": true,
			"created_at":          now.Add(-28 * time.Minute),
			"updated_at":          now.Add(-10 * time.Minute),
			"created_by":          "demo-seed",
			"updated_by":          "demo-seed",
		},
		{
			"_id":                 uuid.New(),
			"order_id":            order1ID,
			"group_id":            group1ID,
			"dish_name":           "Espresso Martini",
			"category":            "cocktail",
			"quantity":            1,
			"price":               12.5,
			"status":              "ready",
			"production_station":  "bar",
			"requires_production": true,
			"created_at":          now.Add(-28 * time.Minute),
			"updated_at":          now.Add(-10 * time.Minute),
			"created_by":          "demo-seed",
			"updated_by":          "demo-seed",
		},
		{
			"_id":                 uuid.New(),
			"order_id":            order1ID,
			"group_id":            group1ID,
			"dish_name":           "Chocolate Lava Cake",
			"category":            "dessert",
			"quantity":            1,
			"price":               10.0,
			"status":              "preparing",
			"notes":               "Extra vanilla ice cream",
			"production_station":  "dessert",
			"requires_production": true,
			"created_at":          now.Add(-15 * time.Minute),
			"updated_at":          now.Add(-5 * time.Minute),
			"created_by":          "demo-seed",
			"updated_by":          "demo-seed",
		},
		{
			"_id":                 uuid.New(),
			"order_id":            order1ID,
			"group_id":            group1ID,
			"dish_name":           "Tiramisu",
			"category":            "dessert",
			"quantity":            1,
			"price":               9.5,
			"status":              "preparing",
			"production_station":  "dessert",
			"requires_production": true,
			"created_at":          now.Add(-15 * time.Minute),
			"updated_at":          now.Add(-5 * time.Minute),
			"created_by":          "demo-seed",
			"updated_by":          "demo-seed",
		},
	}

	for _, item := range demoItems1 {
		_, err = itemsCollection.UpdateOne(ctx, bson.M{"_id": item["_id"]}, bson.M{"$setOnInsert": item}, options.Update().SetUpsert(true))
		if err != nil {
			return fmt.Errorf("cannot create demo item: %w", err)
		}
	}

	// Demo Scenario 2: Table 2 - Group of 4 having dinner
	order2ID := uuid.New()
	group2MainsID := uuid.New()
	group2DrinksID := uuid.New()

	// Parse table ID as UUID
	table2ID, _ := uuid.Parse(tables[1].ID)

	order2 := bson.M{
		"_id":        order2ID,
		"table_id":   table2ID,
		"status":     "pending",
		"created_at": now.Add(-45 * time.Minute),
		"updated_at": now.Add(-45 * time.Minute),
		"created_by": "demo-seed",
		"updated_by": "demo-seed",
	}

	_, err = ordersCollection.UpdateOne(ctx, bson.M{"_id": order2ID}, bson.M{"$setOnInsert": order2}, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("cannot create demo order 2: %w", err)
	}

	group2Mains := bson.M{
		"_id":        group2MainsID,
		"order_id":   order2ID,
		"name":       "Mains",
		"created_at": now.Add(-45 * time.Minute),
		"updated_at": now.Add(-45 * time.Minute),
		"created_by": "demo-seed",
		"updated_by": "demo-seed",
	}

	group2Drinks := bson.M{
		"_id":        group2DrinksID,
		"order_id":   order2ID,
		"name":       "Drinks",
		"created_at": now.Add(-40 * time.Minute),
		"updated_at": now.Add(-40 * time.Minute),
		"created_by": "demo-seed",
		"updated_by": "demo-seed",
	}

	_, err = groupsCollection.UpdateOne(ctx, bson.M{"_id": group2MainsID}, bson.M{"$setOnInsert": group2Mains}, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("cannot create demo group 2 mains: %w", err)
	}

	_, err = groupsCollection.UpdateOne(ctx, bson.M{"_id": group2DrinksID}, bson.M{"$setOnInsert": group2Drinks}, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("cannot create demo group 2 drinks: %w", err)
	}

	demoItems2 := []bson.M{
		{
			"_id":                 uuid.New(),
			"order_id":            order2ID,
			"group_id":            group2MainsID,
			"dish_name":           "Bistro Steak",
			"category":            "entree",
			"quantity":            2,
			"price":               24.0,
			"status":              "preparing",
			"notes":               "Medium rare, no sauce on one",
			"production_station":  "kitchen",
			"requires_production": true,
			"created_at":          now.Add(-45 * time.Minute),
			"updated_at":          now.Add(-20 * time.Minute),
			"created_by":          "demo-seed",
			"updated_by":          "demo-seed",
		},
		{
			"_id":                 uuid.New(),
			"order_id":            order2ID,
			"group_id":            group2MainsID,
			"dish_name":           "Seared Salmon",
			"category":            "entree",
			"quantity":            1,
			"price":               21.0,
			"status":              "preparing",
			"production_station":  "kitchen",
			"requires_production": true,
			"created_at":          now.Add(-45 * time.Minute),
			"updated_at":          now.Add(-20 * time.Minute),
			"created_by":          "demo-seed",
			"updated_by":          "demo-seed",
		},
		{
			"_id":                 uuid.New(),
			"order_id":            order2ID,
			"group_id":            group2MainsID,
			"dish_name":           "Harvest Bowl",
			"category":            "entree",
			"quantity":            1,
			"price":               15.0,
			"status":              "pending",
			"notes":               "Gluten free",
			"production_station":  "kitchen",
			"requires_production": true,
			"created_at":          now.Add(-45 * time.Minute),
			"updated_at":          now.Add(-45 * time.Minute),
			"created_by":          "demo-seed",
			"updated_by":          "demo-seed",
		},
		{
			"_id":                 uuid.New(),
			"order_id":            order2ID,
			"group_id":            group2DrinksID,
			"dish_name":           "West Coast IPA",
			"category":            "beer",
			"quantity":            2,
			"price":               8.5,
			"status":              "ready",
			"production_station":  "bar",
			"requires_production": true,
			"created_at":          now.Add(-40 * time.Minute),
			"updated_at":          now.Add(-25 * time.Minute),
			"created_by":          "demo-seed",
			"updated_by":          "demo-seed",
		},
		{
			"_id":                 uuid.New(),
			"order_id":            order2ID,
			"group_id":            group2DrinksID,
			"dish_name":           "Sparkling Water",
			"category":            "beverage",
			"quantity":            1,
			"price":               4.5,
			"status":              "ready",
			"requires_production": false,
			"created_at":          now.Add(-40 * time.Minute),
			"updated_at":          now.Add(-40 * time.Minute),
			"created_by":          "demo-seed",
			"updated_by":          "demo-seed",
		},
		{
			"_id":                 uuid.New(),
			"order_id":            order2ID,
			"group_id":            group2DrinksID,
			"dish_name":           "Cappuccino",
			"category":            "coffee",
			"quantity":            2,
			"price":               4.5,
			"status":              "pending",
			"production_station":  "coffee",
			"requires_production": true,
			"created_at":          now.Add(-10 * time.Minute),
			"updated_at":          now.Add(-10 * time.Minute),
			"created_by":          "demo-seed",
			"updated_by":          "demo-seed",
		},
	}

	for _, item := range demoItems2 {
		_, err = itemsCollection.UpdateOne(ctx, bson.M{"_id": item["_id"]}, bson.M{"$setOnInsert": item}, options.Update().SetUpsert(true))
		if err != nil {
			return fmt.Errorf("cannot create demo item: %w", err)
		}
	}

	// Demo Scenario 3: Table 3 - Solo diner, simple order
	order3ID := uuid.New()
	group3ID := uuid.New()

	// Parse table ID as UUID
	table3ID, _ := uuid.Parse(tables[2].ID)

	order3 := bson.M{
		"_id":        order3ID,
		"table_id":   table3ID,
		"status":     "pending",
		"created_at": now.Add(-20 * time.Minute),
		"updated_at": now.Add(-20 * time.Minute),
		"created_by": "demo-seed",
		"updated_by": "demo-seed",
	}

	_, err = ordersCollection.UpdateOne(ctx, bson.M{"_id": order3ID}, bson.M{"$setOnInsert": order3}, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("cannot create demo order 3: %w", err)
	}

	group3 := bson.M{
		"_id":        group3ID,
		"order_id":   order3ID,
		"name":       "Solo Lunch",
		"created_at": now.Add(-20 * time.Minute),
		"updated_at": now.Add(-20 * time.Minute),
		"created_by": "demo-seed",
		"updated_by": "demo-seed",
	}

	_, err = groupsCollection.UpdateOne(ctx, bson.M{"_id": group3ID}, bson.M{"$setOnInsert": group3}, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("cannot create demo group 3: %w", err)
	}

	demoItems3 := []bson.M{
		{
			"_id":                 uuid.New(),
			"order_id":            order3ID,
			"group_id":            group3ID,
			"dish_name":           "Smash Burger",
			"category":            "entree",
			"quantity":            1,
			"price":               14.5,
			"status":              "ready",
			"production_station":  "kitchen",
			"requires_production": true,
			"created_at":          now.Add(-20 * time.Minute),
			"updated_at":          now.Add(-5 * time.Minute),
			"created_by":          "demo-seed",
			"updated_by":          "demo-seed",
		},
		{
			"_id":                 uuid.New(),
			"order_id":            order3ID,
			"group_id":            group3ID,
			"dish_name":           "House Iced Tea",
			"category":            "beverage",
			"quantity":            1,
			"price":               5.0,
			"status":              "ready",
			"notes":               "No ice",
			"requires_production": false,
			"created_at":          now.Add(-20 * time.Minute),
			"updated_at":          now.Add(-20 * time.Minute),
			"created_by":          "demo-seed",
			"updated_by":          "demo-seed",
		},
	}

	for _, item := range demoItems3 {
		_, err = itemsCollection.UpdateOne(ctx, bson.M{"_id": item["_id"]}, bson.M{"$setOnInsert": item}, options.Update().SetUpsert(true))
		if err != nil {
			return fmt.Errorf("cannot create demo item: %w", err)
		}
	}

	return nil
}
