package kitchen

import (
	"context"
	"fmt"
	"time"

	"github.com/appetiteclub/appetite/pkg/enums/kitchenstatus"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/seed"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Seeds returns all seeds for the Kitchen service
func Seeds(db *mongo.Database) []seed.Seed {
	return []seed.Seed{
		{
			ID:          "demo_tickets_v1",
			Description: "Create demo kitchen tickets matching the demo orders",
			Run: func(ctx context.Context) error {
				return seedDemoTickets(ctx, db)
			},
		},
	}
}

func seedDemoTickets(ctx context.Context, db *mongo.Database) error {
	ticketsCollection := db.Collection("tickets")

	// Get reference to order database to fetch the demo order items
	orderDB := db.Client().Database("appetite_order")
	itemsCollection := orderDB.Collection("order_items")

	// Fetch all order items that require production
	cursor, err := itemsCollection.Find(ctx, bson.M{
		"requires_production": true,
		"created_by":          "demo-seed",
	})
	if err != nil {
		return fmt.Errorf("cannot fetch demo order items: %w", err)
	}

	var orderItems []struct {
		ID                uuid.UUID  `bson:"_id"`
		OrderID           uuid.UUID  `bson:"order_id"`
		MenuItemID        *uuid.UUID `bson:"menu_item_id"`
		DishName          string     `bson:"dish_name"`
		Category          string     `bson:"category"`
		Quantity          int        `bson:"quantity"`
		Status            string     `bson:"status"`
		Notes             string     `bson:"notes"`
		ProductionStation *string    `bson:"production_station"`
		CreatedAt         time.Time  `bson:"created_at"`
		UpdatedAt         time.Time  `bson:"updated_at"`
	}
	if err := cursor.All(ctx, &orderItems); err != nil {
		return fmt.Errorf("cannot decode demo order items: %w", err)
	}
	cursor.Close(ctx)

	if len(orderItems) == 0 {
		return fmt.Errorf("no demo order items found - run order demo seed first")
	}

	// Map OrderItem status to KitchenTicket status
	mapStatus := func(orderItemStatus string) string {
		switch orderItemStatus {
		case "pending":
			return kitchenstatus.Statuses.Created.Code()
		case "preparing":
			return kitchenstatus.Statuses.Started.Code()
		case "ready":
			return kitchenstatus.Statuses.Ready.Code()
		case "delivered":
			return kitchenstatus.Statuses.Delivered.Code()
		default:
			return kitchenstatus.Statuses.Created.Code()
		}
	}

	// Create matching kitchen tickets
	for _, item := range orderItems {
		if item.ProductionStation == nil {
			continue // Skip items without production station
		}

		ticketID := uuid.New()
		station := *item.ProductionStation

		ticket := bson.M{
			"_id":            ticketID,
			"order_id":       item.OrderID,
			"order_item_id":  item.ID,
			"menu_item_id":   item.MenuItemID,
			"menu_item_name": item.DishName,
			"station":        station,
			"station_name":   capitalizeFirst(station),
			"quantity":       item.Quantity,
			"status":         mapStatus(item.Status),
			"notes":          item.Notes,
			"table_number":   "",
			"created_at":     item.CreatedAt,
			"updated_at":     item.UpdatedAt,
			"created_by":     "demo-seed",
			"updated_by":     "demo-seed",
		}

		// Add timestamps based on status
		if mapStatus(item.Status) == kitchenstatus.Statuses.Started.Code() {
			ticket["started_at"] = item.UpdatedAt
		}
		if mapStatus(item.Status) == kitchenstatus.Statuses.Ready.Code() {
			ticket["started_at"] = item.CreatedAt.Add(5 * time.Minute)
			ticket["finished_at"] = item.UpdatedAt
		}

		_, err := ticketsCollection.UpdateOne(
			ctx,
			bson.M{"_id": ticketID},
			bson.M{"$setOnInsert": ticket},
			options.Update().SetUpsert(true),
		)
		if err != nil {
			return fmt.Errorf("cannot create demo ticket for item %s: %w", item.DishName, err)
		}
	}

	return nil
}

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) == 1 {
		return string(s[0] - 32)
	}
	return string(s[0]-32) + s[1:]
}

// ApplyDemoSeeds applies demo seeds if enabled via config
func ApplyDemoSeeds(ctx context.Context, config *aqm.Config, dbFn func() *mongo.Database, logger aqm.Logger) error {
	enabled, _ := config.GetString("seed.demo.enabled")
	if enabled != "true" {
		return nil
	}

	logger.Info("Demo seeding enabled, applying demo kitchen tickets...")
	db := dbFn()
	tracker := seed.NewMongoTracker(db)
	seeds := Seeds(db)

	if err := seed.Apply(ctx, tracker, seeds, "kitchen"); err != nil {
		return fmt.Errorf("demo seed failed: %w", err)
	}

	logger.Info("Demo kitchen tickets seeded successfully")
	return nil
}
