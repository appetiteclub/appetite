package mongo

import (
	"context"
	"fmt"

	"github.com/appetiteclub/appetite/services/menu/internal/menu"
	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MenuRepo implements the menu.MenuRepo interface using MongoDB
type MenuRepo struct {
	itemRepo   *MenuItemRepo
	collection *mongo.Collection
	logger     apt.Logger
}

// NewMenuRepo creates a new MongoDB menu repository
func NewMenuRepo(itemRepo *MenuItemRepo, logger apt.Logger) *MenuRepo {
	return &MenuRepo{
		itemRepo: itemRepo,
		logger:   logger,
	}
}

// Start initializes the menu repository (uses same DB as MenuItemRepo)
func (r *MenuRepo) Start(ctx context.Context) error {
	if r.itemRepo == nil || r.itemRepo.db == nil {
		return fmt.Errorf("menu item repository must be started first")
	}

	r.collection = r.itemRepo.db.Collection("menus")

	// Create index on version_state for faster queries
	stateIndexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "version_state", Value: 1}},
	}
	if _, err := r.collection.Indexes().CreateOne(ctx, stateIndexModel); err != nil {
		return fmt.Errorf("cannot create version_state index: %w", err)
	}

	r.logger.Info("Menu repository initialized with collection: menus")
	return nil
}

// Stop is a no-op for MenuRepo since connection is managed by MenuItemRepo
func (r *MenuRepo) Stop(ctx context.Context) error {
	return nil
}

// Create inserts a new menu
func (r *MenuRepo) Create(ctx context.Context, m *menu.Menu) error {
	if m == nil {
		return fmt.Errorf("menu cannot be nil")
	}

	m.EnsureID()
	m.BeforeCreate()

	_, err := r.collection.InsertOne(ctx, m)
	if err != nil {
		return fmt.Errorf("could not create menu: %w", err)
	}
	return nil
}

// Get retrieves a menu by ID
func (r *MenuRepo) Get(ctx context.Context, id uuid.UUID) (*menu.Menu, error) {
	var m menu.Menu

	filter := bson.M{"_id": id.String()}
	err := r.collection.FindOne(ctx, filter).Decode(&m)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("menu with ID %s not found", id.String())
		}
		return nil, fmt.Errorf("could not get menu: %w", err)
	}
	return &m, nil
}

// List retrieves all menus
func (r *MenuRepo) List(ctx context.Context) ([]*menu.Menu, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("could not list menus: %w", err)
	}
	defer cursor.Close(ctx)

	var menus []*menu.Menu
	for cursor.Next(ctx) {
		var m menu.Menu
		if err := cursor.Decode(&m); err != nil {
			return nil, fmt.Errorf("could not decode menu: %w", err)
		}
		menus = append(menus, &m)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	return menus, nil
}

// ListPublished retrieves all published menus
func (r *MenuRepo) ListPublished(ctx context.Context) ([]*menu.Menu, error) {
	filter := bson.M{"version_state": string(menu.MenuVersionPublished)}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("could not list published menus: %w", err)
	}
	defer cursor.Close(ctx)

	var menus []*menu.Menu
	for cursor.Next(ctx) {
		var m menu.Menu
		if err := cursor.Decode(&m); err != nil {
			return nil, fmt.Errorf("could not decode menu: %w", err)
		}
		menus = append(menus, &m)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	return menus, nil
}

// Save updates an existing menu
func (r *MenuRepo) Save(ctx context.Context, m *menu.Menu) error {
	if m == nil {
		return fmt.Errorf("menu cannot be nil")
	}

	m.BeforeUpdate()

	filter := bson.M{"_id": m.GetID().String()}
	opts := options.Replace().SetUpsert(false)

	result, err := r.collection.ReplaceOne(ctx, filter, m, opts)
	if err != nil {
		return fmt.Errorf("could not save menu: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("menu with ID %s not found for update", m.GetID().String())
	}
	return nil
}

// Delete removes a menu by ID
func (r *MenuRepo) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id.String()}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("could not delete menu: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("menu with ID %s not found for deletion", id.String())
	}
	return nil
}
