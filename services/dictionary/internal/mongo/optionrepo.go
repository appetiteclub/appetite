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

	"github.com/appetiteclub/appetite/services/dictionary/internal/dictionary"
)

// OptionRepo implements the dictionary.OptionRepo interface using MongoDB.
type OptionRepo struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	setRepo    *SetRepo
	logger     aqm.Logger
	config     *aqm.Config
}

// NewOptionRepo creates a new MongoDB repository for Option aggregates.
func NewOptionRepo(setRepo *SetRepo, config *aqm.Config, logger aqm.Logger) *OptionRepo {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &OptionRepo{
		setRepo: setRepo,
		logger:  logger,
		config:  config,
	}
}

// Start connects to MongoDB and initializes the collection.
func (r *OptionRepo) Start(ctx context.Context) error {
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
	r.collection = r.db.Collection("options")

	// Create compound index on (set_id, key, locale) for uniqueness within a set per locale
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "set_id", Value: 1},
			{Key: "key", Value: 1},
			{Key: "locale", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	if _, err := r.collection.Indexes().CreateOne(ctx, indexModel); err != nil {
		return fmt.Errorf("cannot create index: %w", err)
	}

	// Create index on set_id for faster queries
	setIndexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "set_id", Value: 1}},
	}
	if _, err := r.collection.Indexes().CreateOne(ctx, setIndexModel); err != nil {
		return fmt.Errorf("cannot create set_id index: %w", err)
	}

	r.logger.Infof("Connected to MongoDB: %s, database: %s, collection: options", connString, dbName)
	return nil
}

// Stop closes the MongoDB connection.
func (r *OptionRepo) Stop(ctx context.Context) error {
	if r.client != nil {
		if err := r.client.Disconnect(ctx); err != nil {
			return fmt.Errorf("cannot disconnect from MongoDB: %w", err)
		}
		r.logger.Info("Disconnected from MongoDB")
	}
	return nil
}

// Create creates a new Option aggregate in MongoDB.
func (r *OptionRepo) Create(ctx context.Context, option *dictionary.Option) error {
	if option == nil {
		return fmt.Errorf("option cannot be nil")
	}

	option.EnsureID()
	option.BeforeCreate()

	_, err := r.collection.InsertOne(ctx, option)
	if err != nil {
		return fmt.Errorf("could not create Option aggregate: %w", err)
	}

	return nil
}

// Get retrieves a complete Option aggregate by ID from MongoDB.
func (r *OptionRepo) Get(ctx context.Context, id uuid.UUID) (*dictionary.Option, error) {
	var option dictionary.Option

	filter := bson.M{"_id": id.String()}
	err := r.collection.FindOne(ctx, filter).Decode(&option)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("Option aggregate with ID %s not found", id.String())
		}
		return nil, fmt.Errorf("could not get Option aggregate: %w", err)
	}

	return &option, nil
}

// Save performs a unit-of-work save operation on the Option aggregate.
func (r *OptionRepo) Save(ctx context.Context, option *dictionary.Option) error {
	if option == nil {
		return fmt.Errorf("option cannot be nil")
	}

	option.BeforeUpdate()

	filter := bson.M{"_id": option.GetID().String()}
	opts := options.Replace().SetUpsert(false)

	result, err := r.collection.ReplaceOne(ctx, filter, option, opts)
	if err != nil {
		return fmt.Errorf("could not save Option aggregate: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("Option aggregate with ID %s not found for update", option.GetID().String())
	}

	return nil
}

// Delete removes the entire Option aggregate from MongoDB.
func (r *OptionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id.String()}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("could not delete Option aggregate: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("Option aggregate with ID %s not found for deletion", id.String())
	}

	return nil
}

// List retrieves all Option aggregates from MongoDB.
func (r *OptionRepo) List(ctx context.Context) ([]*dictionary.Option, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("could not list Option aggregates: %w", err)
	}
	defer cursor.Close(ctx)

	var options []*dictionary.Option

	for cursor.Next(ctx) {
		var option dictionary.Option
		if err := cursor.Decode(&option); err != nil {
			return nil, fmt.Errorf("could not decode Option aggregate: %w", err)
		}
		options = append(options, &option)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while listing Option aggregates: %w", err)
	}

	return options, nil
}

// ListBySet retrieves all options for a specific set.
func (r *OptionRepo) ListBySet(ctx context.Context, setID uuid.UUID) ([]*dictionary.Option, error) {
	filter := bson.M{"set_id": setID.String()}
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "order", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("could not list options by set: %w", err)
	}
	defer cursor.Close(ctx)

	var options []*dictionary.Option

	for cursor.Next(ctx) {
		var option dictionary.Option
		if err := cursor.Decode(&option); err != nil {
			return nil, fmt.Errorf("could not decode Option aggregate: %w", err)
		}
		options = append(options, &option)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while listing options by set: %w", err)
	}

	return options, nil
}

// ListBySetName retrieves all options for a set by set name.
func (r *OptionRepo) ListBySetName(ctx context.Context, setName string) ([]*dictionary.Option, error) {
	// First, get the set by name
	set, err := r.setRepo.GetByName(ctx, setName)
	if err != nil {
		return nil, fmt.Errorf("could not get set by name: %w", err)
	}

	// Then, list options by set ID
	return r.ListBySet(ctx, set.ID)
}

// ListByParent retrieves all options with a specific parent ID.
// If parentID is nil, returns root-level options (no parent).
func (r *OptionRepo) ListByParent(ctx context.Context, setID uuid.UUID, parentID *uuid.UUID) ([]*dictionary.Option, error) {
	filter := bson.M{"set_id": setID.String()}
	if parentID == nil {
		filter["parent_id"] = nil
	} else {
		filter["parent_id"] = parentID.String()
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "order", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("could not list options by parent: %w", err)
	}
	defer cursor.Close(ctx)

	var options []*dictionary.Option

	for cursor.Next(ctx) {
		var option dictionary.Option
		if err := cursor.Decode(&option); err != nil {
			return nil, fmt.Errorf("could not decode Option aggregate: %w", err)
		}
		options = append(options, &option)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while listing options by parent: %w", err)
	}

	return options, nil
}

// ListBySetAndParent retrieves all options in a set filtered by parent ID.
func (r *OptionRepo) ListBySetAndParent(ctx context.Context, setName string, parentID *uuid.UUID) ([]*dictionary.Option, error) {
	// First, get the set by name
	set, err := r.setRepo.GetByName(ctx, setName)
	if err != nil {
		return nil, fmt.Errorf("could not get set by name: %w", err)
	}

	// Then, list options by set ID and parent
	return r.ListByParent(ctx, set.ID, parentID)
}

// ListActive retrieves all active Option aggregates.
func (r *OptionRepo) ListActive(ctx context.Context) ([]*dictionary.Option, error) {
	filter := bson.M{"active": true}
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "order", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("could not list active options: %w", err)
	}
	defer cursor.Close(ctx)

	var options []*dictionary.Option

	for cursor.Next(ctx) {
		var option dictionary.Option
		if err := cursor.Decode(&option); err != nil {
			return nil, fmt.Errorf("could not decode Option aggregate: %w", err)
		}
		options = append(options, &option)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while listing active options: %w", err)
	}

	return options, nil
}
