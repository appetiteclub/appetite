package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/appetiteclub/apt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BaseRepo struct {
	client *mongo.Client
	db     *mongo.Database
	logger apt.Logger
	config *apt.Config
}

func NewBaseRepo(config *apt.Config, logger apt.Logger) *BaseRepo {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}
	return &BaseRepo{
		logger: logger,
		config: config,
	}
}

func (r *BaseRepo) Start(ctx context.Context) error {
	mongoURL, _ := r.config.GetString("db.mongo.url")
	connString := mongoURL
	if connString == "" {
		connString = "mongodb://localhost:27017"
	}

	dbName, _ := r.config.GetString("db.mongo.name")
	if dbName == "" {
		dbName = "appetite_order"
	}

	clientOptions := options.Client().ApplyURI(connString).
		SetConnectTimeout(10 * time.Second).
		SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("cannot connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("cannot ping MongoDB: %w", err)
	}

	r.client = client
	r.db = client.Database(dbName)

	r.logger.Infof("Connected to MongoDB: %s, database: %s", connString, dbName)
	return nil
}

func (r *BaseRepo) Stop(ctx context.Context) error {
	if r.client != nil {
		if err := r.client.Disconnect(ctx); err != nil {
			return fmt.Errorf("cannot disconnect from MongoDB: %w", err)
		}
		r.logger.Info("Disconnected from MongoDB")
	}
	return nil
}

func (r *BaseRepo) GetDatabase() *mongo.Database {
	return r.db
}
