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

	"github.com/appetiteclub/appetite/services/authz/internal/authz"
	authpkg "github.com/aquamarinepk/aqm/auth"
)

// GrantMongoRepo implements the GrantRepo interface using MongoDB
type GrantMongoRepo struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	logger     aqm.Logger
	config      *aqm.Config
}

// NewGrantMongoRepo creates a new MongoDB repository for Grant entities
func NewGrantMongoRepo(logger aqm.Logger, config *aqm.Config) *GrantMongoRepo {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &GrantMongoRepo{
		logger: logger,
		config:  config,
	}
}

// Start connects to MongoDB and initializes the collection
func (r *GrantMongoRepo) Start(ctx context.Context) error {
	// Set default MongoDB configuration if not provided
	mongoURL, _ := r.config.GetString("db.mongo.url")
	connString := mongoURL
	if connString == "" {
		connString = "mongodb://localhost:27017"
	}

	dbName, _ := r.config.GetString("db.mongo.name")
	if dbName == "" {
		dbName = "appetite_authz"
	}

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(connString)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("cannot connect to MongoDB: %w", err)
	}

	// Ping MongoDB to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("cannot ping MongoDB: %w", err)
	}

	r.client = client
	r.db = client.Database(dbName)
	r.collection = r.db.Collection("grants")

	// Create indexes
	if err := r.createIndexes(ctx); err != nil {
		return fmt.Errorf("cannot create indexes: %w", err)
	}

	return nil
}

// Stop closes the MongoDB connection
func (r *GrantMongoRepo) Stop(ctx context.Context) error {
	if r.client != nil {
		if err := r.client.Disconnect(ctx); err != nil {
			return fmt.Errorf("cannot disconnect from MongoDB: %w", err)
		}
	}
	return nil
}

// createIndexes creates necessary indexes for the grants collection
func (r *GrantMongoRepo) createIndexes(ctx context.Context) error {
	// Index on user_id (primary lookup)
	userIDIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}},
	}

	// Index on expires_at (TTL cleanup)
	expiresAtIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0), // TTL index
	}

	// Index on scope.type for scope filtering
	scopeTypeIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "scope.type", Value: 1}},
	}

	// Index on scope.id for context filtering
	scopeIDIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "scope.id", Value: 1}},
	}

	// Index on status
	statusIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "status", Value: 1}},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		userIDIndex,
		expiresAtIndex,
		scopeTypeIndex,
		scopeIDIndex,
		statusIndex,
	})

	return err
}

// grantDocument represents the MongoDB document structure
type grantDocument struct {
	ID        string                 `bson:"_id"`
	UserID    string                 `bson:"user_id"`
	GrantType string                 `bson:"grant_type"`
	Value     string                 `bson:"value"`
	Scope     map[string]interface{} `bson:"scope"`
	ExpiresAt *time.Time             `bson:"expires_at,omitempty"`
	Status    string                 `bson:"status"`
	CreatedAt time.Time              `bson:"created_at"`
	CreatedBy string                 `bson:"created_by"`
	UpdatedAt time.Time              `bson:"updated_at"`
	UpdatedBy string                 `bson:"updated_by"`
}

// toDocument converts a Grant entity to MongoDB document
func (r *GrantMongoRepo) toDocument(grant *authz.Grant) *grantDocument {
	scope := map[string]interface{}{
		"type": grant.Scope.Type,
		"id":   grant.Scope.ID,
	}

	return &grantDocument{
		ID:        grant.ID.String(),
		UserID:    grant.UserID.String(),
		GrantType: string(grant.GrantType),
		Value:     grant.Value,
		Scope:     scope,
		ExpiresAt: grant.ExpiresAt,
		Status:    string(grant.Status),
		CreatedAt: grant.CreatedAt,
		CreatedBy: grant.CreatedBy,
		UpdatedAt: grant.UpdatedAt,
		UpdatedBy: grant.UpdatedBy,
	}
}

// fromDocument converts a MongoDB document to Grant entity
func (r *GrantMongoRepo) fromDocument(doc *grantDocument) (*authz.Grant, error) {
	id, err := uuid.Parse(doc.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid grant ID format: %w", err)
	}

	userID, err := uuid.Parse(doc.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	// Parse scope
	scope := authz.Scope{
		Type: doc.Scope["type"].(string),
		ID:   doc.Scope["id"].(string),
	}

	return &authz.Grant{
		ID:        id,
		UserID:    userID,
		GrantType: authz.GrantType(doc.GrantType),
		Value:     doc.Value,
		Scope:     scope,
		ExpiresAt: doc.ExpiresAt,
		Status:    authpkg.UserStatus(doc.Status),
		CreatedAt: doc.CreatedAt,
		CreatedBy: doc.CreatedBy,
		UpdatedAt: doc.UpdatedAt,
		UpdatedBy: doc.UpdatedBy,
	}, nil
}

// Create creates a new Grant in MongoDB
func (r *GrantMongoRepo) Create(ctx context.Context, grant *authz.Grant) error {
	if grant == nil {
		return fmt.Errorf("grant cannot be nil")
	}

	grant.EnsureID()
	grant.BeforeCreate()

	doc := r.toDocument(grant)

	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("error create grant: %w", err)
	}

	return nil
}

// Get retrieves a Grant by ID from MongoDB
func (r *GrantMongoRepo) Get(ctx context.Context, id uuid.UUID) (*authz.Grant, error) {
	filter := bson.M{"_id": id.String()}

	var doc grantDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get grant: %w", err)
	}

	return r.fromDocument(&doc)
}

// Save updates an existing Grant in MongoDB
func (r *GrantMongoRepo) Save(ctx context.Context, grant *authz.Grant) error {
	if grant == nil {
		return fmt.Errorf("grant cannot be nil")
	}

	grant.BeforeUpdate()

	filter := bson.M{"_id": grant.ID.String()}
	doc := r.toDocument(grant)

	update := bson.M{
		"$set": bson.M{
			"user_id":    doc.UserID,
			"grant_type": doc.GrantType,
			"value":      doc.Value,
			"scope":      doc.Scope,
			"expires_at": doc.ExpiresAt,
			"status":     doc.Status,
			"updated_at": doc.UpdatedAt,
			"updated_by": doc.UpdatedBy,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error update grant: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("grant with ID %s not found for update", grant.ID.String())
	}

	return nil
}

// Delete performs a soft delete by changing the grant status to deleted
func (r *GrantMongoRepo) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id.String()}
	update := bson.M{
		"$set": bson.M{
			"status":     "deleted",
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error delete grant: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("grant with ID %s not found for deletion", id.String())
	}

	return nil
}

// List retrieves all active Grants from MongoDB
func (r *GrantMongoRepo) List(ctx context.Context) ([]*authz.Grant, error) {
	filter := bson.M{"status": bson.M{"$ne": "deleted"}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error query grants: %w", err)
	}
	defer cursor.Close(ctx)

	var grants []*authz.Grant

	for cursor.Next(ctx) {
		var doc grantDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("error decode grant document: %w", err)
		}

		grant, err := r.fromDocument(&doc)
		if err != nil {
			return nil, fmt.Errorf("error convert document to grant: %w", err)
		}

		grants = append(grants, grant)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return grants, nil
}

// ListByUserID retrieves Grants for a specific user from MongoDB
func (r *GrantMongoRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*authz.Grant, error) {
	filter := bson.M{
		"user_id": userID.String(),
		"status":  bson.M{"$ne": "deleted"},
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error query grants by user ID: %w", err)
	}
	defer cursor.Close(ctx)

	var grants []*authz.Grant

	for cursor.Next(ctx) {
		var doc grantDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("error decode grant document: %w", err)
		}

		grant, err := r.fromDocument(&doc)
		if err != nil {
			return nil, fmt.Errorf("error convert document to grant: %w", err)
		}

		grants = append(grants, grant)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return grants, nil
}

// ListByScope retrieves Grants for a specific scope from MongoDB
func (r *GrantMongoRepo) ListByScope(ctx context.Context, scope authz.Scope) ([]*authz.Grant, error) {
	filter := bson.M{
		"scope.type": scope.Type,
		"scope.id":   scope.ID,
		"status":     bson.M{"$ne": "deleted"},
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error query grants by scope: %w", err)
	}
	defer cursor.Close(ctx)

	var grants []*authz.Grant

	for cursor.Next(ctx) {
		var doc grantDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("error decode grant document: %w", err)
		}

		grant, err := r.fromDocument(&doc)
		if err != nil {
			return nil, fmt.Errorf("error convert document to grant: %w", err)
		}

		grants = append(grants, grant)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return grants, nil
}

// ListExpired retrieves expired Grants from MongoDB
func (r *GrantMongoRepo) ListExpired(ctx context.Context) ([]*authz.Grant, error) {
	now := time.Now()
	filter := bson.M{
		"expires_at": bson.M{"$lt": now},
		"status":     bson.M{"$ne": "deleted"},
	}
	opts := options.Find().SetSort(bson.D{{Key: "expires_at", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error query expired grants: %w", err)
	}
	defer cursor.Close(ctx)

	var grants []*authz.Grant

	for cursor.Next(ctx) {
		var doc grantDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("error decode grant document: %w", err)
		}

		grant, err := r.fromDocument(&doc)
		if err != nil {
			return nil, fmt.Errorf("error convert document to grant: %w", err)
		}

		grants = append(grants, grant)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return grants, nil
}

func (r *GrantMongoRepo) Log() aqm.Logger {
	return r.logger
}
