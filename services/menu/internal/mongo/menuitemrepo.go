package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/appetiteclub/appetite/services/menu/internal/menu"
	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MenuItemRepo implements the menu.MenuItemRepo interface using MongoDB
type MenuItemRepo struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	logger     apt.Logger
	config     *apt.Config
}

// NewMenuItemRepo creates a new MongoDB menu item repository
func NewMenuItemRepo(config *apt.Config, logger apt.Logger) *MenuItemRepo {
	return &MenuItemRepo{
		logger: logger,
		config: config,
	}
}

// Start initializes the MongoDB connection
func (r *MenuItemRepo) Start(ctx context.Context) error {
	mongoURL, _ := r.config.GetString("db.mongo.url")
	if mongoURL == "" {
		mongoURL = "mongodb://localhost:27017"
	}

	dbName, _ := r.config.GetString("db.mongo.name")
	if dbName == "" {
		dbName = "appetite_menu"
	}

	clientOptions := options.Client().ApplyURI(mongoURL).
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
	r.collection = r.db.Collection("menu_items")

	// Create unique index on short_code
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "short_code", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}
	if _, err := r.collection.Indexes().CreateOne(ctx, indexModel); err != nil {
		return fmt.Errorf("cannot create short_code index: %w", err)
	}

	// Create index on active status for faster queries
	activeIndexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "active", Value: 1}},
	}
	if _, err := r.collection.Indexes().CreateOne(ctx, activeIndexModel); err != nil {
		return fmt.Errorf("cannot create active index: %w", err)
	}

	// Create index on categories for faster category queries
	categoryIndexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "categories", Value: 1}},
	}
	if _, err := r.collection.Indexes().CreateOne(ctx, categoryIndexModel); err != nil {
		return fmt.Errorf("cannot create categories index: %w", err)
	}

	r.logger.Infof("Connected to MongoDB: %s, database: %s, collection: menu_items", mongoURL, dbName)
	return nil
}

// Stop closes the MongoDB connection
func (r *MenuItemRepo) Stop(ctx context.Context) error {
	if r.client != nil {
		if err := r.client.Disconnect(ctx); err != nil {
			return fmt.Errorf("cannot disconnect from MongoDB: %w", err)
		}
		r.logger.Info("Disconnected from MongoDB")
	}
	return nil
}

// GetDatabase returns the MongoDB database instance
func (r *MenuItemRepo) GetDatabase() *mongo.Database {
	return r.db
}

// Create inserts a new menu item
func (r *MenuItemRepo) Create(ctx context.Context, item *menu.MenuItem) error {
	if item == nil {
		return fmt.Errorf("menu item cannot be nil")
	}

	item.EnsureID()
	item.BeforeCreate()

	_, err := r.collection.InsertOne(ctx, item)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("menu item with short_code %s already exists", item.ShortCode)
		}
		return fmt.Errorf("could not create menu item: %w", err)
	}
	return nil
}

// Get retrieves a menu item by ID
func (r *MenuItemRepo) Get(ctx context.Context, id uuid.UUID) (*menu.MenuItem, error) {
	var item menu.MenuItem

	filter := bson.M{"_id": id.String()}
	err := r.collection.FindOne(ctx, filter).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("menu item with ID %s not found", id.String())
		}
		return nil, fmt.Errorf("could not get menu item: %w", err)
	}
	return &item, nil
}

// GetByShortCode retrieves a menu item by short code
func (r *MenuItemRepo) GetByShortCode(ctx context.Context, shortCode string) (*menu.MenuItem, error) {
	var item menu.MenuItem

	filter := bson.M{"short_code": shortCode}
	err := r.collection.FindOne(ctx, filter).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("menu item with short_code %s not found", shortCode)
		}
		return nil, fmt.Errorf("could not get menu item by short_code: %w", err)
	}
	return &item, nil
}

// List retrieves all menu items
func (r *MenuItemRepo) List(ctx context.Context) ([]*menu.MenuItem, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("could not list menu items: %w", err)
	}
	defer cursor.Close(ctx)

	var items []*menu.MenuItem
	for cursor.Next(ctx) {
		var item menu.MenuItem
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("could not decode menu item: %w", err)
		}
		items = append(items, &item)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	return items, nil
}

// ListActive retrieves all active menu items
func (r *MenuItemRepo) ListActive(ctx context.Context) ([]*menu.MenuItem, error) {
	filter := bson.M{"active": true}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("could not list active menu items: %w", err)
	}
	defer cursor.Close(ctx)

	var items []*menu.MenuItem
	for cursor.Next(ctx) {
		var item menu.MenuItem
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("could not decode menu item: %w", err)
		}
		items = append(items, &item)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	return items, nil
}

// ListByCategory retrieves menu items by category
func (r *MenuItemRepo) ListByCategory(ctx context.Context, categoryID uuid.UUID) ([]*menu.MenuItem, error) {
	filter := bson.M{"categories": categoryID.String()}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("could not list menu items by category: %w", err)
	}
	defer cursor.Close(ctx)

	var items []*menu.MenuItem
	for cursor.Next(ctx) {
		var item menu.MenuItem
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("could not decode menu item: %w", err)
		}
		items = append(items, &item)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	return items, nil
}

// Save updates an existing menu item
func (r *MenuItemRepo) Save(ctx context.Context, item *menu.MenuItem) error {
	if item == nil {
		return fmt.Errorf("menu item cannot be nil")
	}

	item.BeforeUpdate()

	filter := bson.M{"_id": item.GetID().String()}
	opts := options.Replace().SetUpsert(false)

	result, err := r.collection.ReplaceOne(ctx, filter, item, opts)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("menu item with short_code %s already exists", item.ShortCode)
		}
		return fmt.Errorf("could not save menu item: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("menu item with ID %s not found for update", item.GetID().String())
	}
	return nil
}

// Delete removes a menu item by ID
func (r *MenuItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id.String()}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("could not delete menu item: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("menu item with ID %s not found for deletion", id.String())
	}
	return nil
}
