package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/appetiteclub/appetite/services/dictionary/internal/dictionary"
)

// SetRepo implements the dictionary.SetRepo interface using MongoDB.
type SetRepo struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	logger     apt.Logger
	config     *apt.Config
}

// NewSetRepo creates a new MongoDB repository for Set aggregates.
func NewSetRepo(config *apt.Config, logger apt.Logger) *SetRepo {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}
	return &SetRepo{
		logger: logger,
		config: config,
	}
}

// Start connects to MongoDB and initializes the collection.
func (r *SetRepo) Start(ctx context.Context) error {
	// Set default MongoDB configuration if not provided
	mongoURL, _ := r.config.GetString("db.mongo.url")
	connString := mongoURL
	if connString == "" {
		connString = "mongodb://localhost:27017"
	}

	dbName, _ := r.config.GetString("db.mongo.name")
	if dbName == "" {
		dbName = "appetite_dictionary"
	}

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(connString).
		SetConnectTimeout(10 * time.Second).
		SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("cannot connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("cannot ping MongoDB: %w", err)
	}

	r.client = client
	r.db = client.Database(dbName)
	r.collection = r.db.Collection("sets")

	// Create unique compound index on (name, locale) to ensure name uniqueness per locale
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "name", Value: 1},
			{Key: "locale", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	if _, err := r.collection.Indexes().CreateOne(ctx, indexModel); err != nil {
		return fmt.Errorf("cannot create index: %w", err)
	}

	r.logger.Infof("Connected to MongoDB: %s, database: %s, collection: sets", connString, dbName)
	return nil
}

// Stop closes the MongoDB connection.
func (r *SetRepo) Stop(ctx context.Context) error {
	if r.client != nil {
		if err := r.client.Disconnect(ctx); err != nil {
			return fmt.Errorf("cannot disconnect from MongoDB: %w", err)
		}
		r.logger.Info("Disconnected from MongoDB")
	}
	return nil
}

// GetDatabase returns the MongoDB database instance for seeding and other operations.
func (r *SetRepo) GetDatabase() *mongo.Database {
	return r.db
}

// Create creates a new Set aggregate in MongoDB.
func (r *SetRepo) Create(ctx context.Context, set *dictionary.Set) error {
	if set == nil {
		return fmt.Errorf("set cannot be nil")
	}

	set.EnsureID()
	set.BeforeCreate()

	_, err := r.collection.InsertOne(ctx, set)
	if err != nil {
		return fmt.Errorf("could not create Set aggregate: %w", err)
	}

	return nil
}

// Get retrieves a complete Set aggregate by ID from MongoDB.
func (r *SetRepo) Get(ctx context.Context, id uuid.UUID) (*dictionary.Set, error) {
	var set dictionary.Set

	filter := bson.M{"_id": id.String()}
	err := r.collection.FindOne(ctx, filter).Decode(&set)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("Set aggregate with ID %s not found", id.String())
		}
		return nil, fmt.Errorf("could not get Set aggregate: %w", err)
	}

	return &set, nil
}

// GetByName retrieves a Set by its unique name (returns first match, use GetByNameAndLocale for specific locale).
func (r *SetRepo) GetByName(ctx context.Context, name string) (*dictionary.Set, error) {
	var set dictionary.Set

	filter := bson.M{"name": name}
	err := r.collection.FindOne(ctx, filter).Decode(&set)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("Set with name %s not found", name)
		}
		return nil, fmt.Errorf("could not get Set by name: %w", err)
	}

	return &set, nil
}

// GetByNameAndLocale retrieves a Set by its unique name and locale.
func (r *SetRepo) GetByNameAndLocale(ctx context.Context, name, locale string) (*dictionary.Set, error) {
	var set dictionary.Set

	filter := bson.M{"name": name, "locale": locale}
	err := r.collection.FindOne(ctx, filter).Decode(&set)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("Set with name %s and locale %s not found", name, locale)
		}
		return nil, fmt.Errorf("could not get Set by name and locale: %w", err)
	}

	return &set, nil
}

// Save performs a unit-of-work save operation on the Set aggregate.
func (r *SetRepo) Save(ctx context.Context, set *dictionary.Set) error {
	if set == nil {
		return fmt.Errorf("set cannot be nil")
	}

	set.BeforeUpdate()

	filter := bson.M{"_id": set.GetID().String()}
	opts := options.Replace().SetUpsert(false)

	result, err := r.collection.ReplaceOne(ctx, filter, set, opts)
	if err != nil {
		return fmt.Errorf("could not save Set aggregate: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("Set aggregate with ID %s not found for update", set.GetID().String())
	}

	return nil
}

// Delete removes the entire Set aggregate from MongoDB.
func (r *SetRepo) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id.String()}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("could not delete Set aggregate: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("Set aggregate with ID %s not found for deletion", id.String())
	}

	return nil
}

// List retrieves all Set aggregates from MongoDB.
func (r *SetRepo) List(ctx context.Context) ([]*dictionary.Set, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("could not list Set aggregates: %w", err)
	}
	defer cursor.Close(ctx)

	var sets []*dictionary.Set

	for cursor.Next(ctx) {
		var set dictionary.Set
		if err := cursor.Decode(&set); err != nil {
			return nil, fmt.Errorf("could not decode Set aggregate: %w", err)
		}
		sets = append(sets, &set)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while listing Set aggregates: %w", err)
	}

	return sets, nil
}

// ListActive retrieves all active Set aggregates.
func (r *SetRepo) ListActive(ctx context.Context) ([]*dictionary.Set, error) {
	filter := bson.M{"active": true}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("could not list active sets: %w", err)
	}
	defer cursor.Close(ctx)

	var sets []*dictionary.Set

	for cursor.Next(ctx) {
		var set dictionary.Set
		if err := cursor.Decode(&set); err != nil {
			return nil, fmt.Errorf("could not decode Set aggregate: %w", err)
		}
		sets = append(sets, &set)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while listing active sets: %w", err)
	}

	return sets, nil
}
