package kitchen

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/appetiteclub/appetite/pkg/enums/kitchenstatus"
	"github.com/appetiteclub/appetite/pkg/enums/station"
	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/seed"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const kitchenDemoSeedApplication = "kitchen_demo"

// ApplyDemoSeeds creates realistic demo kitchen tickets from order items.
func ApplyDemoSeeds(ctx context.Context, repo TicketRepository, cache *TicketStateCache, db *mongo.Database, logger apt.Logger) error {
	if db == nil {
		return errors.New("database is required for demo seeding")
	}

	demoSeeds := buildDemoKitchenSeeds(repo, db, logger)
	if len(demoSeeds) == 0 {
		logger.Info("No demo kitchen seeds to apply")
		return nil
	}

	tracker := seed.NewMongoTracker(db)

	logger.Info("Applying demo kitchen seeds")
	if err := seed.Apply(ctx, tracker, demoSeeds, kitchenDemoSeedApplication); err != nil {
		return err
	}
	logger.Info("Demo kitchen seeds applied successfully")

	// Reload cache from database after seeding to make tickets available
	// Note: We need to load from repo directly because tickets were created in DB without events
	logger.Info("Reloading ticket cache from database after demo seeding")
	err := cache.WarmFromRepo(ctx)
	if err != nil {
		logger.Errorf("Failed to reload cache after seeding: %v", err)
		return fmt.Errorf("reload cache after seeding: %w", err)
	}
	ticketCount := cache.Count()
	logger.Info("Ticket cache reloaded successfully", "ticket_count", ticketCount)

	return nil
}

func buildDemoKitchenSeeds(repo TicketRepository, db *mongo.Database, logger apt.Logger) []seed.Seed {
	return []seed.Seed{
		{
			ID:          "2024-11-23_demo_kitchen_tickets_v1",
			Description: "Create demo kitchen tickets from order items",
			Run: func(ctx context.Context) error {
				return seedDemoKitchenTickets(ctx, repo, db, logger)
			},
		},
	}
}

// OrderItemForTicket represents the subset of OrderItem fields needed for ticket creation
type OrderItemForTicket struct {
	ID                 uuid.UUID `bson:"_id"`
	OrderID            uuid.UUID `bson:"order_id"`
	DishName           string    `bson:"dish_name"`
	Quantity           int       `bson:"quantity"`
	Status             string    `bson:"status"`
	Notes              string    `bson:"notes"`
	ProductionStation  *string   `bson:"production_station"`
	RequiresProduction bool      `bson:"requires_production"`
	CreatedAt          time.Time `bson:"created_at"`
	UpdatedAt          time.Time `bson:"updated_at"`
}

func seedDemoKitchenTickets(ctx context.Context, repo TicketRepository, db *mongo.Database, logger apt.Logger) error {
	// Fetch demo order items from the order database
	orderDB := db.Client().Database("appetite_order")
	itemsCollection := orderDB.Collection("order_items")

	// Find all order items that require production and were created by demo seeding
	cursor, err := itemsCollection.Find(ctx, bson.M{
		"requires_production": true,
		"created_by":          "seed:demo",
	})
	if err != nil {
		return fmt.Errorf("find demo order items: %w", err)
	}
	defer cursor.Close(ctx)

	var orderItems []OrderItemForTicket
	if err := cursor.All(ctx, &orderItems); err != nil {
		return fmt.Errorf("decode demo order items: %w", err)
	}

	if len(orderItems) == 0 {
		logger.Info("No demo order items found for kitchen tickets")
		return nil
	}

	logger.Info("Found demo order items for kitchen tickets", "count", len(orderItems))

	// Station name mapping for denormalized display
	stationNames := map[string]string{
		station.Stations.Bar.Code():     station.Stations.Bar.Label(),
		station.Stations.Kitchen.Code(): station.Stations.Kitchen.Label(),
		station.Stations.Dessert.Code(): station.Stations.Dessert.Label(),
		station.Stations.Coffee.Code():  station.Stations.Coffee.Label(),
		station.Stations.Other.Code():   station.Stations.Other.Label(),
	}

	// Create a kitchen ticket for each order item
	for _, item := range orderItems {
		if item.ProductionStation == nil {
			logger.Info("Skipping item without production station", "item_id", item.ID)
			continue
		}

		stationCode := *item.ProductionStation
		stationName := stationNames[stationCode]
		if stationName == "" {
			stationName = stationCode // Fallback to code if name not found
		}

		// Generate new ticket ID
		ticketID := uuid.New()

		// Map order item status to kitchen ticket status
		ticketStatus := mapOrderItemStatusToKitchenStatus(item.Status)

		// Create ticket
		ticket := &Ticket{
			ID:               ticketID,
			OrderID:          item.OrderID,
			OrderItemID:      item.ID,
			MenuItemID:       uuid.Nil, // Demo doesn't use menu items
			Station:          stationCode,
			Quantity:         item.Quantity,
			Status:           ticketStatus,
			Notes:            item.Notes,
			DecisionRequired: false,
			DecisionPayload:  nil,
			// Denormalized display data
			MenuItemName: item.DishName,
			StationName:  stationName,
			TableNumber:  "", // Not fetching table number for demo
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
			StartedAt:    nil,
			FinishedAt:   nil,
			DeliveredAt:  nil,
			ModelVersion: 1,
		}

		// Set timestamps based on status
		switch ticketStatus {
		case kitchenstatus.Statuses.Started.Code():
			startedAt := item.UpdatedAt
			ticket.StartedAt = &startedAt
		case kitchenstatus.Statuses.Ready.Code():
			startedAt := item.CreatedAt.Add(5 * time.Minute)
			finishedAt := item.UpdatedAt
			ticket.StartedAt = &startedAt
			ticket.FinishedAt = &finishedAt
		case kitchenstatus.Statuses.Delivered.Code():
			startedAt := item.CreatedAt.Add(5 * time.Minute)
			finishedAt := item.CreatedAt.Add(15 * time.Minute)
			deliveredAt := item.UpdatedAt
			ticket.StartedAt = &startedAt
			ticket.FinishedAt = &finishedAt
			ticket.DeliveredAt = &deliveredAt
		}

		if err := repo.Create(ctx, ticket); err != nil {
			return fmt.Errorf("create demo kitchen ticket: %w", err)
		}

		logger.Info("Created demo kitchen ticket",
			"dish", item.DishName,
			"station", stationName,
			"status", ticketStatus)
	}

	return nil
}

// mapOrderItemStatusToKitchenStatus maps order item statuses to kitchen ticket statuses
func mapOrderItemStatusToKitchenStatus(orderStatus string) string {
	switch orderStatus {
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

// DemoSeedingFunc returns an aqm lifecycle OnStart-compatible function for demo seeding.
func DemoSeedingFunc(seedCtx context.Context, repo TicketRepository, cache *TicketStateCache, db *mongo.Database, logger apt.Logger) func(ctx context.Context) error {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}

	return func(ctx context.Context) error {
		logger.Info("Starting demo kitchen seeding in background")
		go func() {
			if err := ApplyDemoSeeds(seedCtx, repo, cache, db, logger); err != nil && !errors.Is(err, context.Canceled) {
				logger.Errorf("❌ Demo kitchen seeds failed: %v", err)
			} else if err == nil {
				logger.Info("✓ Demo kitchen seeding completed successfully")
			}
		}()
		return nil
	}
}
