package commands

import (
	"context"
	"fmt"

	"github.com/aquamarinepk/aqm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ClearDemo removes all demo data from order and kitchen databases
func ClearDemo(ctx context.Context, config *aqm.Config, logger aqm.Logger) error {
	logger.Info("Starting demo data cleanup...")

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

	// Clear order demo data
	orderDB := client.Database("appetite_order")
	if err := clearOrderDemo(ctx, orderDB, logger); err != nil {
		return fmt.Errorf("clear order demo: %w", err)
	}

	// Clear kitchen demo data
	kitchenDB := client.Database("appetite_kitchen")
	if err := clearKitchenDemo(ctx, kitchenDB, logger); err != nil {
		return fmt.Errorf("clear kitchen demo: %w", err)
	}

	return nil
}

func clearOrderDemo(ctx context.Context, db *mongo.Database, logger aqm.Logger) error {
	logger.Info("Clearing order demo data...")

	// Delete demo order items
	itemsCollection := db.Collection("order_items")
	itemsResult, err := itemsCollection.DeleteMany(ctx, bson.M{"created_by": "demo-seed"})
	if err != nil {
		return fmt.Errorf("delete demo order items: %w", err)
	}
	logger.Info("Deleted demo order items", "count", itemsResult.DeletedCount)

	// Delete demo orders
	ordersCollection := db.Collection("orders")
	ordersResult, err := ordersCollection.DeleteMany(ctx, bson.M{"created_by": "demo-seed"})
	if err != nil {
		return fmt.Errorf("delete demo orders: %w", err)
	}
	logger.Info("Deleted demo orders", "count", ordersResult.DeletedCount)

	// Delete demo order groups
	groupsCollection := db.Collection("order_groups")
	groupsResult, err := groupsCollection.DeleteMany(ctx, bson.M{"created_by": "demo-seed"})
	if err != nil {
		return fmt.Errorf("delete demo order groups: %w", err)
	}
	logger.Info("Deleted demo order groups", "count", groupsResult.DeletedCount)

	// Clear seed tracker
	seedsCollection := db.Collection("_seeds")
	trackerResult, err := seedsCollection.DeleteOne(ctx, bson.M{"_id": "demo_orders_v1"})
	if err != nil {
		return fmt.Errorf("delete order seed tracker: %w", err)
	}
	logger.Info("Cleared order seed tracker", "deleted", trackerResult.DeletedCount)

	return nil
}

func clearKitchenDemo(ctx context.Context, db *mongo.Database, logger aqm.Logger) error {
	logger.Info("Clearing kitchen demo data...")

	// Delete demo tickets
	ticketsCollection := db.Collection("tickets")
	ticketsResult, err := ticketsCollection.DeleteMany(ctx, bson.M{"created_by": "demo-seed"})
	if err != nil {
		return fmt.Errorf("delete demo tickets: %w", err)
	}
	logger.Info("Deleted demo kitchen tickets", "count", ticketsResult.DeletedCount)

	// Clear seed tracker
	seedsCollection := db.Collection("_seeds")
	trackerResult, err := seedsCollection.DeleteOne(ctx, bson.M{"_id": "demo_tickets_v1"})
	if err != nil {
		return fmt.Errorf("delete kitchen seed tracker: %w", err)
	}
	logger.Info("Cleared kitchen seed tracker", "deleted", trackerResult.DeletedCount)

	return nil
}
