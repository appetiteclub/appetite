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

// RoleMongoRepo implements the RoleRepo interface using MongoDB
type RoleMongoRepo struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	logger     aqm.Logger
	config     *aqm.Config
}

// NewRoleMongoRepo creates a new MongoDB repository for Role entities
func NewRoleMongoRepo(config *aqm.Config, logger aqm.Logger) *RoleMongoRepo {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &RoleMongoRepo{
		logger: logger,
		config: config,
	}
}

// Start connects to MongoDB and initializes the collection
func (r *RoleMongoRepo) Start(ctx context.Context) error {
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
	r.collection = r.db.Collection("roles")

	// Create indexes
	if err := r.createIndexes(ctx); err != nil {
		return fmt.Errorf("cannot create indexes: %w", err)
	}

	return nil
}

// Stop closes the MongoDB connection
func (r *RoleMongoRepo) Stop(ctx context.Context) error {
	if r.client != nil {
		if err := r.client.Disconnect(ctx); err != nil {
			return fmt.Errorf("cannot disconnect from MongoDB: %w", err)
		}
	}
	return nil
}

// Database exposes the MongoDB database for infrastructure concerns (like seeding).
func (r *RoleMongoRepo) Database() *mongo.Database {
	return r.db
}

// createIndexes creates necessary indexes for the roles collection
func (r *RoleMongoRepo) createIndexes(ctx context.Context) error {
	// Index on name (unique)
	nameIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	// Index on status
	statusIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "status", Value: 1}},
	}

	// Index on created_at
	createdAtIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		nameIndex,
		statusIndex,
		createdAtIndex,
	})

	return err
}

// roleDocument represents the MongoDB document structure
type roleDocument struct {
	ID          string    `bson:"_id"`
	Name        string    `bson:"name"`
	Permissions []string  `bson:"permissions"`
	Status      string    `bson:"status"`
	CreatedAt   time.Time `bson:"created_at"`
	CreatedBy   string    `bson:"created_by"`
	UpdatedAt   time.Time `bson:"updated_at"`
	UpdatedBy   string    `bson:"updated_by"`
}

// toDocument converts a Role entity to MongoDB document
func (r *RoleMongoRepo) toDocument(role *authz.Role) *roleDocument {
	return &roleDocument{
		ID:          role.ID.String(),
		Name:        role.Name,
		Permissions: role.Permissions,
		Status:      string(role.Status),
		CreatedAt:   role.CreatedAt,
		CreatedBy:   role.CreatedBy,
		UpdatedAt:   role.UpdatedAt,
		UpdatedBy:   role.UpdatedBy,
	}
}

// fromDocument converts a MongoDB document to Role entity
func (r *RoleMongoRepo) fromDocument(doc *roleDocument) (*authz.Role, error) {
	id, err := uuid.Parse(doc.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid role ID format: %w", err)
	}

	return &authz.Role{
		ID:          id,
		Name:        doc.Name,
		Permissions: doc.Permissions,
		Status:      authpkg.UserStatus(doc.Status),
		CreatedAt:   doc.CreatedAt,
		CreatedBy:   doc.CreatedBy,
		UpdatedAt:   doc.UpdatedAt,
		UpdatedBy:   doc.UpdatedBy,
	}, nil
}

// Create creates a new Role in MongoDB
func (r *RoleMongoRepo) Create(ctx context.Context, role *authz.Role) error {
	if role == nil {
		return fmt.Errorf("role cannot be nil")
	}

	role.EnsureID()
	role.BeforeCreate()

	doc := r.toDocument(role)

	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("error create role: %w", err)
	}

	return nil
}

// Get retrieves a Role by ID from MongoDB
func (r *RoleMongoRepo) Get(ctx context.Context, id uuid.UUID) (*authz.Role, error) {
	filter := bson.M{"_id": id.String()}

	var doc roleDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get role: %w", err)
	}

	return r.fromDocument(&doc)
}

// GetByName retrieves a Role by name from MongoDB
func (r *RoleMongoRepo) GetByName(ctx context.Context, name string) (*authz.Role, error) {
	filter := bson.M{"name": name}

	var doc roleDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get role by name: %w", err)
	}

	return r.fromDocument(&doc)
}

// Save updates an existing Role in MongoDB
func (r *RoleMongoRepo) Save(ctx context.Context, role *authz.Role) error {
	if role == nil {
		return fmt.Errorf("role cannot be nil")
	}

	role.BeforeUpdate()

	filter := bson.M{"_id": role.ID.String()}
	update := bson.M{
		"$set": bson.M{
			"name":        role.Name,
			"permissions": role.Permissions,
			"status":      string(role.Status),
			"updated_at":  role.UpdatedAt,
			"updated_by":  role.UpdatedBy,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error update role: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("role with ID %s not found for update", role.ID.String())
	}

	return nil
}

// Delete performs a soft delete by changing the role status to deleted
func (r *RoleMongoRepo) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id.String()}
	update := bson.M{
		"$set": bson.M{
			"status":     "deleted",
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error delete role: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("role with ID %s not found for deletion", id.String())
	}

	return nil
}

// List retrieves all active Roles from MongoDB
func (r *RoleMongoRepo) List(ctx context.Context) ([]*authz.Role, error) {
	filter := bson.M{"status": bson.M{"$ne": "deleted"}}
	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error query roles: %w", err)
	}
	defer cursor.Close(ctx)

	var roles []*authz.Role

	for cursor.Next(ctx) {
		var doc roleDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("error decode role document: %w", err)
		}

		role, err := r.fromDocument(&doc)
		if err != nil {
			return nil, fmt.Errorf("error convert document to role: %w", err)
		}

		roles = append(roles, role)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return roles, nil
}

// ListByStatus retrieves Roles filtered by status from MongoDB
func (r *RoleMongoRepo) ListByStatus(ctx context.Context, status string) ([]*authz.Role, error) {
	filter := bson.M{"status": status}
	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error query roles by status: %w", err)
	}
	defer cursor.Close(ctx)

	var roles []*authz.Role

	for cursor.Next(ctx) {
		var doc roleDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("error decode role document: %w", err)
		}

		role, err := r.fromDocument(&doc)
		if err != nil {
			return nil, fmt.Errorf("error convert document to role: %w", err)
		}

		roles = append(roles, role)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return roles, nil
}

func (r *RoleMongoRepo) Log() aqm.Logger {
	return r.logger
}
