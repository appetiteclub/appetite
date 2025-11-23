package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/seed"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const orderDemoSeedApplication = "order_demo"

// ApplyDemoSeeds creates realistic demo orders with items for different stations.
func ApplyDemoSeeds(ctx context.Context, repos Repos, db *mongo.Database, logger aqm.Logger) error {
	if db == nil {
		return errors.New("database is required for demo seeding")
	}

	demoSeeds := buildDemoOrderSeeds(repos, db, logger)
	if len(demoSeeds) == 0 {
		logger.Info("No demo order seeds to apply")
		return nil
	}

	tracker := seed.NewMongoTracker(db)

	logger.Info("Applying demo order seeds")
	if err := seed.Apply(ctx, tracker, demoSeeds, orderDemoSeedApplication); err != nil {
		return err
	}
	logger.Info("Demo order seeds applied successfully")
	return nil
}

func buildDemoOrderSeeds(repos Repos, db *mongo.Database, logger aqm.Logger) []seed.Seed {
	return []seed.Seed{
		{
			ID:          "2024-11-23_demo_orders_v1",
			Description: "Create demo orders with realistic distribution across stations",
			Run: func(ctx context.Context) error {
				return seedDemoOrders(ctx, repos, db, logger)
			},
		},
	}
}

func seedDemoOrders(ctx context.Context, repos Repos, db *mongo.Database, logger aqm.Logger) error {
	// Get table IDs from the table database
	tableDB := db.Client().Database("appetite_table")
	tablesCollection := tableDB.Collection("tables")

	// Map table numbers to IDs
	tableIDs := make(map[string]uuid.UUID)
	tableNumbers := []string{"Window-1", "Center-2", "Patio-3", "Booth-7", "Terrace-8"}

	for _, number := range tableNumbers {
		var result struct {
			ID string `bson:"_id"`
		}
		err := tablesCollection.FindOne(ctx, bson.M{"number": number}).Decode(&result)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				logger.Info("Demo table not found, skipping", "number", number)
				continue
			}
			return fmt.Errorf("find table %s: %w", number, err)
		}
		tableID, err := uuid.Parse(result.ID)
		if err != nil {
			return fmt.Errorf("parse table ID for %s: %w", number, err)
		}
		tableIDs[number] = tableID
	}

	if len(tableIDs) < 3 {
		return fmt.Errorf("need at least 3 tables for demo orders, found %d", len(tableIDs))
	}

	now := time.Now()

	// Demo Scenario 1: Window-1 - Couple having drinks and desserts
	if tableID, ok := tableIDs["Window-1"]; ok {
		if err := createScenario1(ctx, repos, tableID, now, logger); err != nil {
			return fmt.Errorf("create scenario 1: %w", err)
		}
	}

	// Demo Scenario 2: Center-2 - Group of 4 having dinner
	if tableID, ok := tableIDs["Center-2"]; ok {
		if err := createScenario2(ctx, repos, tableID, now, logger); err != nil {
			return fmt.Errorf("create scenario 2: %w", err)
		}
	}

	// Demo Scenario 3: Patio-3 - Solo diner
	if tableID, ok := tableIDs["Patio-3"]; ok {
		if err := createScenario3(ctx, repos, tableID, now, logger); err != nil {
			return fmt.Errorf("create scenario 3: %w", err)
		}
	}

	// Demo Scenario 4: Booth-7 - Small group with cocktails
	if tableID, ok := tableIDs["Booth-7"]; ok {
		if err := createScenario4(ctx, repos, tableID, now, logger); err != nil {
			return fmt.Errorf("create scenario 4: %w", err)
		}
	}

	logger.Info("Demo orders created successfully")
	return nil
}

// Scenario 1: Couple having drinks and desserts
func createScenario1(ctx context.Context, repos Repos, tableID uuid.UUID, now time.Time, logger aqm.Logger) error {
	order := NewOrder()
	order.TableID = tableID
	order.Status = "pending"
	order.CreatedAt = now.Add(-30 * time.Minute)
	order.UpdatedAt = now.Add(-30 * time.Minute)
	order.CreatedBy = "seed:demo"
	order.UpdatedBy = "seed:demo"

	if err := repos.OrderRepo.Create(ctx, order); err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	group := NewOrderGroup(order.ID, "Drinks & Desserts")
	group.CreatedBy = "seed:demo"
	group.UpdatedBy = "seed:demo"
	if err := repos.OrderGroupRepo.Create(ctx, group); err != nil {
		return fmt.Errorf("create group: %w", err)
	}

	// Bar items (ready) - using string station codes
	barStation := "bar"
	groupID := group.ID
	items := []*OrderItem{
		createItem(order.ID, &groupID, "Aperol Spritz", "cocktail", 1, 11.0, "ready", "", &barStation, true, now.Add(-28*time.Minute), now.Add(-10*time.Minute)),
		createItem(order.ID, &groupID, "Espresso Martini", "cocktail", 1, 12.5, "ready", "", &barStation, true, now.Add(-28*time.Minute), now.Add(-10*time.Minute)),
	}

	// Dessert items (preparing)
	dessertStation := "dessert"
	items = append(items,
		createItem(order.ID, &groupID, "Chocolate Lava Cake", "dessert", 1, 10.0, "preparing", "Extra vanilla ice cream", &dessertStation, true, now.Add(-15*time.Minute), now.Add(-5*time.Minute)),
		createItem(order.ID, &groupID, "Tiramisu", "dessert", 1, 9.5, "preparing", "", &dessertStation, true, now.Add(-15*time.Minute), now.Add(-5*time.Minute)),
	)

	for _, item := range items {
		if err := repos.OrderItemRepo.Create(ctx, item); err != nil {
			return fmt.Errorf("create item %s: %w", item.DishName, err)
		}
	}

	logger.Info("Created demo order scenario 1", "table_id", tableID, "order_id", order.ID)
	return nil
}

// Scenario 2: Group of 4 having dinner
func createScenario2(ctx context.Context, repos Repos, tableID uuid.UUID, now time.Time, logger aqm.Logger) error {
	order := NewOrder()
	order.TableID = tableID
	order.Status = "pending"
	order.CreatedAt = now.Add(-45 * time.Minute)
	order.UpdatedAt = now.Add(-45 * time.Minute)
	order.CreatedBy = "seed:demo"
	order.UpdatedBy = "seed:demo"

	if err := repos.OrderRepo.Create(ctx, order); err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	// Create two groups
	groupMains := NewOrderGroup(order.ID, "Mains")
	groupMains.CreatedBy = "seed:demo"
	groupMains.UpdatedBy = "seed:demo"
	if err := repos.OrderGroupRepo.Create(ctx, groupMains); err != nil {
		return fmt.Errorf("create mains group: %w", err)
	}

	groupDrinks := NewOrderGroup(order.ID, "Drinks")
	groupDrinks.CreatedBy = "seed:demo"
	groupDrinks.UpdatedBy = "seed:demo"
	if err := repos.OrderGroupRepo.Create(ctx, groupDrinks); err != nil {
		return fmt.Errorf("create drinks group: %w", err)
	}

	kitchenStation := "kitchen"
	barStation := "bar"
	coffeeStation := "coffee"

	groupMainsID := groupMains.ID
	groupDrinksID := groupDrinks.ID

	// Mains (kitchen)
	items := []*OrderItem{
		createItem(order.ID, &groupMainsID, "Bistro Steak", "entree", 2, 24.0, "preparing", "Medium rare, no sauce on one", &kitchenStation, true, now.Add(-45*time.Minute), now.Add(-20*time.Minute)),
		createItem(order.ID, &groupMainsID, "Seared Salmon", "entree", 1, 21.0, "preparing", "", &kitchenStation, true, now.Add(-45*time.Minute), now.Add(-20*time.Minute)),
		createItem(order.ID, &groupMainsID, "Harvest Bowl", "entree", 1, 15.0, "pending", "Gluten free", &kitchenStation, true, now.Add(-45*time.Minute), now.Add(-45*time.Minute)),
	}

	// Drinks (bar and coffee)
	items = append(items,
		createItem(order.ID, &groupDrinksID, "West Coast IPA", "beer", 2, 8.5, "ready", "", &barStation, true, now.Add(-40*time.Minute), now.Add(-25*time.Minute)),
		createItem(order.ID, &groupDrinksID, "Sparkling Water", "beverage", 1, 4.5, "ready", "", nil, false, now.Add(-40*time.Minute), now.Add(-40*time.Minute)),
		createItem(order.ID, &groupDrinksID, "Cappuccino", "coffee", 2, 4.5, "pending", "", &coffeeStation, true, now.Add(-10*time.Minute), now.Add(-10*time.Minute)),
	)

	for _, item := range items {
		if err := repos.OrderItemRepo.Create(ctx, item); err != nil {
			return fmt.Errorf("create item %s: %w", item.DishName, err)
		}
	}

	logger.Info("Created demo order scenario 2", "table_id", tableID, "order_id", order.ID)
	return nil
}

// Scenario 3: Solo diner
func createScenario3(ctx context.Context, repos Repos, tableID uuid.UUID, now time.Time, logger aqm.Logger) error {
	order := NewOrder()
	order.TableID = tableID
	order.Status = "pending"
	order.CreatedAt = now.Add(-20 * time.Minute)
	order.UpdatedAt = now.Add(-20 * time.Minute)
	order.CreatedBy = "seed:demo"
	order.UpdatedBy = "seed:demo"

	if err := repos.OrderRepo.Create(ctx, order); err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	group := NewOrderGroup(order.ID, "Solo Lunch")
	group.CreatedBy = "seed:demo"
	group.UpdatedBy = "seed:demo"
	if err := repos.OrderGroupRepo.Create(ctx, group); err != nil {
		return fmt.Errorf("create group: %w", err)
	}

	kitchenStation := "kitchen"
	groupID := group.ID

	items := []*OrderItem{
		createItem(order.ID, &groupID, "Smash Burger", "entree", 1, 14.5, "ready", "No pickles", &kitchenStation, true, now.Add(-20*time.Minute), now.Add(-5*time.Minute)),
		createItem(order.ID, &groupID, "House Iced Tea", "beverage", 1, 5.0, "ready", "No ice", nil, false, now.Add(-20*time.Minute), now.Add(-20*time.Minute)),
	}

	for _, item := range items {
		if err := repos.OrderItemRepo.Create(ctx, item); err != nil {
			return fmt.Errorf("create item %s: %w", item.DishName, err)
		}
	}

	logger.Info("Created demo order scenario 3", "table_id", tableID, "order_id", order.ID)
	return nil
}

// Scenario 4: Small group with cocktails
func createScenario4(ctx context.Context, repos Repos, tableID uuid.UUID, now time.Time, logger aqm.Logger) error {
	order := NewOrder()
	order.TableID = tableID
	order.Status = "pending"
	order.CreatedAt = now.Add(-35 * time.Minute)
	order.UpdatedAt = now.Add(-35 * time.Minute)
	order.CreatedBy = "seed:demo"
	order.UpdatedBy = "seed:demo"

	if err := repos.OrderRepo.Create(ctx, order); err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	group := NewOrderGroup(order.ID, "Cocktails")
	group.CreatedBy = "seed:demo"
	group.UpdatedBy = "seed:demo"
	if err := repos.OrderGroupRepo.Create(ctx, group); err != nil {
		return fmt.Errorf("create group: %w", err)
	}

	barStation := "bar"
	groupID := group.ID

	items := []*OrderItem{
		createItem(order.ID, &groupID, "Classic Martini", "cocktail", 2, 13.0, "ready", "Shaken, not stirred", &barStation, true, now.Add(-35*time.Minute), now.Add(-15*time.Minute)),
		createItem(order.ID, &groupID, "Old Fashioned", "cocktail", 1, 12.0, "preparing", "Extra orange peel", &barStation, true, now.Add(-30*time.Minute), now.Add(-10*time.Minute)),
		createItem(order.ID, &groupID, "Negroni", "cocktail", 1, 11.5, "pending", "", &barStation, true, now.Add(-25*time.Minute), now.Add(-25*time.Minute)),
	}

	for _, item := range items {
		if err := repos.OrderItemRepo.Create(ctx, item); err != nil {
			return fmt.Errorf("create item %s: %w", item.DishName, err)
		}
	}

	logger.Info("Created demo order scenario 4", "table_id", tableID, "order_id", order.ID)
	return nil
}

func createItem(orderID uuid.UUID, groupID *uuid.UUID, dishName, category string, quantity int, price float64, status, notes string, station *string, requiresProduction bool, createdAt, updatedAt time.Time) *OrderItem {
	item := NewOrderItem()
	item.OrderID = orderID
	item.GroupID = groupID
	item.DishName = dishName
	item.Category = category
	item.Quantity = quantity
	item.Price = price
	item.Status = status
	item.Notes = notes
	item.ProductionStation = station
	item.RequiresProduction = requiresProduction
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt
	item.CreatedBy = "seed:demo"
	item.UpdatedBy = "seed:demo"
	return item
}

// DemoSeedingFunc returns an aqm lifecycle OnStart-compatible function for demo seeding.
func DemoSeedingFunc(seedCtx context.Context, repos Repos, db *mongo.Database, logger aqm.Logger) func(ctx context.Context) error {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}

	return func(ctx context.Context) error {
		logger.Info("Starting demo order seeding in background")
		go func() {
			if err := ApplyDemoSeeds(seedCtx, repos, db, logger); err != nil && !errors.Is(err, context.Canceled) {
				logger.Errorf("❌ Demo order seeds failed: %v", err)
			} else if err == nil {
				logger.Info("✓ Demo order seeding completed successfully")
			}
		}()
		return nil
	}
}
