package commands

import (
	"context"
	"fmt"

	"github.com/appetiteclub/appetite/cmd/utils/internal/seeding"
	"github.com/appetiteclub/apt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SeedDemo applies demo seeding to order and kitchen databases
func SeedDemo(ctx context.Context, config *apt.Config, logger apt.Logger) error {
	logger.Info("Starting demo seeding process...")

	// Connect to MongoDB
	mongoURL, _ := config.GetString("mongo.url")
	if mongoURL == "" {
		mongoURL = "mongodb://admin:password@localhost:27017/admin?authSource=admin"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		return fmt.Errorf("connect to mongodb: %w", err)
	}
	defer client.Disconnect(ctx)

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("ping mongodb: %w", err)
	}

	logger.Info("Connected to MongoDB")

	// Seed order database
	orderDB := client.Database("appetite_order")
	if err := seedOrderDemo(ctx, orderDB, logger); err != nil {
		return fmt.Errorf("seed order demo: %w", err)
	}

	// Seed kitchen database
	kitchenDB := client.Database("appetite_kitchen")
	if err := seedKitchenDemo(ctx, kitchenDB, logger); err != nil {
		return fmt.Errorf("seed kitchen demo: %w", err)
	}

	return nil
}

func seedOrderDemo(ctx context.Context, db *mongo.Database, logger apt.Logger) error {
	// Check if already seeded
	seedsCollection := db.Collection("_seeds")
	count, err := seedsCollection.CountDocuments(ctx, bson.M{"_id": "demo_orders_v1"})
	if err != nil {
		return fmt.Errorf("check seed status: %w", err)
	}

	if count > 0 {
		logger.Info("Order demo seeds already applied, skipping")
		return nil
	}

	// Apply the seed
	if err := seeding.SeedOrders(ctx, db); err != nil {
		return fmt.Errorf("seed orders: %w", err)
	}

	// Mark as seeded
	_, err = seedsCollection.InsertOne(ctx, bson.M{
		"_id":         "demo_orders_v1",
		"description": "Create demo orders with realistic distribution of items across stations and statuses",
		"applied_at":  bson.M{"$currentDate": bson.M{"$type": "timestamp"}},
	})
	if err != nil {
		logger.Infof("⚠️  Failed to mark seed as applied: %v", err)
	}

	logger.Info("Order demo seeds applied successfully")
	return nil
}

func seedKitchenDemo(ctx context.Context, db *mongo.Database, logger apt.Logger) error {
	// Check if already seeded
	seedsCollection := db.Collection("_seeds")
	count, err := seedsCollection.CountDocuments(ctx, bson.M{"_id": "demo_tickets_v1"})
	if err != nil {
		return fmt.Errorf("check seed status: %w", err)
	}

	if count > 0 {
		logger.Info("Kitchen demo seeds already applied, skipping")
		return nil
	}

	// Apply the seed
	if err := seeding.SeedKitchenTickets(ctx, db); err != nil {
		return fmt.Errorf("seed kitchen tickets: %w", err)
	}

	// Mark as seeded
	_, err = seedsCollection.InsertOne(ctx, bson.M{
		"_id":         "demo_tickets_v1",
		"description": "Create demo kitchen tickets matching the demo orders",
		"applied_at":  bson.M{"$currentDate": bson.M{"$type": "timestamp"}},
	})
	if err != nil {
		logger.Infof("⚠️  Failed to mark seed as applied: %v", err)
	}

	logger.Info("Kitchen demo seeds applied successfully")
	return nil
}
