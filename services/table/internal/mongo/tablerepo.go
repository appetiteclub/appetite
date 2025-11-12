package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/aquamarinepk/aqm"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/appetiteclub/appetite/services/table/internal/tables"
)

type TableRepo struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	logger     aqm.Logger
	config     *aqm.Config
}

func NewTableRepo(config *aqm.Config, logger aqm.Logger) *TableRepo {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &TableRepo{
		logger: logger,
		config: config,
	}
}

func (r *TableRepo) Start(ctx context.Context) error {
	mongoURL, _ := r.config.GetString("db.mongo.url")
	connString := mongoURL
	if connString == "" {
		connString = "mongodb://localhost:27017"
	}

	dbName, _ := r.config.GetString("db.mongo.name")
	if dbName == "" {
		dbName = "appetite_tables"
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
	r.collection = r.db.Collection("tables")

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "number", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	if _, err := r.collection.Indexes().CreateOne(ctx, indexModel); err != nil {
		return fmt.Errorf("cannot create index: %w", err)
	}

	r.logger.Infof("Connected to MongoDB: %s, database: %s, collection: tables", connString, dbName)
	return nil
}

func (r *TableRepo) Stop(ctx context.Context) error {
	if r.client != nil {
		if err := r.client.Disconnect(ctx); err != nil {
			return fmt.Errorf("cannot disconnect from MongoDB: %w", err)
		}
		r.logger.Info("Disconnected from MongoDB")
	}
	return nil
}

func (r *TableRepo) GetDatabase() *mongo.Database {
	return r.db
}

func (r *TableRepo) Create(ctx context.Context, table *tables.Table) error {
	if table == nil {
		return fmt.Errorf("table is nil")
	}

	if _, err := r.collection.InsertOne(ctx, table); err != nil {
		return fmt.Errorf("cannot create table: %w", err)
	}

	return nil
}

func (r *TableRepo) Get(ctx context.Context, id uuid.UUID) (*tables.Table, error) {
	var table tables.Table
	err := r.collection.FindOne(ctx, bson.M{"_id": id.String()}).Decode(&table)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot get table: %w", err)
	}
	return &table, nil
}

func (r *TableRepo) GetByNumber(ctx context.Context, number string) (*tables.Table, error) {
	var table tables.Table
	err := r.collection.FindOne(ctx, bson.M{"number": number}).Decode(&table)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot get table by number: %w", err)
	}
	return &table, nil
}

func (r *TableRepo) List(ctx context.Context) ([]*tables.Table, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("cannot list tables: %w", err)
	}
	defer cursor.Close(ctx)

	var result []*tables.Table
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cannot decode tables: %w", err)
	}

	return result, nil
}

func (r *TableRepo) ListByStatus(ctx context.Context, status string) ([]*tables.Table, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, fmt.Errorf("cannot list tables by status: %w", err)
	}
	defer cursor.Close(ctx)

	var result []*tables.Table
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cannot decode tables: %w", err)
	}

	return result, nil
}

func (r *TableRepo) Save(ctx context.Context, table *tables.Table) error {
	if table == nil {
		return fmt.Errorf("table is nil")
	}

	filter := bson.M{"_id": table.ID.String()}
	update := bson.M{"$set": table}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("cannot update table: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("table not found")
	}

	return nil
}

func (r *TableRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id.String()})
	if err != nil {
		return fmt.Errorf("cannot delete table: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("table not found")
	}

	return nil
}
