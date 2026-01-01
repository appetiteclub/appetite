package commands

import (
	"context"
	"fmt"

	"github.com/appetiteclub/apt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var allDatabases = []string{
	"appetite_authn",
	"appetite_authz",
	"appetite_dictionary",
	"appetite_kitchen",
	"appetite_menu",
	"appetite_order",
	"appetite_table",
	"appetite_media",
	"appetite_operations",
	"appetite_admin",
}

// ResetDB drops all Appetite databases - USE WITH CAUTION
func ResetDB(ctx context.Context, config *apt.Config, logger apt.Logger) error {
	logger.Infof("⚠️  DANGER: This will drop ALL Appetite databases!")
	logger.Infof("⚠️  This action cannot be undone!")

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

	// Drop each database
	for _, dbName := range allDatabases {
		logger.Info("Dropping database", "database", dbName)
		db := client.Database(dbName)
		result := db.RunCommand(ctx, bson.D{{Key: "dropDatabase", Value: 1}})
		if result.Err() != nil {
			logger.Infof("⚠️  Failed to drop database %s (may not exist): %v", dbName, result.Err())
		} else {
			logger.Info("Database dropped", "database", dbName)
		}
	}

	logger.Info("All databases have been dropped")
	return nil
}
