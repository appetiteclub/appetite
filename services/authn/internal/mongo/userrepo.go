package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/aquamarinepk/aqm"
	authpkg "github.com/aquamarinepk/aqm/auth"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/appetiteclub/appetite/services/authn/internal/authn"
)

// UserMongoRepo implements the UserRepo interface using MongoDB.
type UserMongoRepo struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	logger     aqm.Logger
	config     *aqm.Config
}

// NewUserMongoRepo creates a new MongoDB repository for User entities.
func NewUserMongoRepo(config *aqm.Config, logger aqm.Logger) *UserMongoRepo {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &UserMongoRepo{
		logger: logger,
		config: config,
	}
}

// Start connects to MongoDB and initializes the collection.
func (r *UserMongoRepo) Start(ctx context.Context) error {
	mongoURL, _ := r.config.GetString("db.mongo.url")
	connString := mongoURL
	if connString == "" {
		connString = "mongodb://localhost:27017"
	}

	dbName, _ := r.config.GetString("db.mongo.name")
	if dbName == "" {
		dbName = "appetite_authn"
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
	r.collection = r.db.Collection("users")

	// Create indexes
	if err := r.createIndexes(ctx); err != nil {
		return fmt.Errorf("cannot create indexes: %w", err)
	}

	return nil
}

// Stop closes the MongoDB connection.
func (r *UserMongoRepo) Stop(ctx context.Context) error {
	if r.client != nil {
		if err := r.client.Disconnect(ctx); err != nil {
			return fmt.Errorf("cannot disconnect from MongoDB: %w", err)
		}
	}
	return nil
}

// Database exposes the underlying MongoDB database, primarily for infrastructure tasks (e.g., seeding).
func (r *UserMongoRepo) Database() *mongo.Database {
	return r.db
}

// createIndexes creates necessary indexes for the users collection.
func (r *UserMongoRepo) createIndexes(ctx context.Context) error {
	// Index on email_lookup (unique)
	emailLookupIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "email_lookup", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	// Index on username (unique)
	usernameIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	// Index on pin_lookup (unique, sparse - only for users with PIN)
	pinLookupIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "pin_lookup", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
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
		emailLookupIndex,
		usernameIndex,
		pinLookupIndex,
		statusIndex,
		createdAtIndex,
	})

	return err
}

// userDocument represents the MongoDB document structure.
type userDocument struct {
	ID           string    `bson:"_id"`
	Username     string    `bson:"username"`
	Name         string    `bson:"name"`
	EmailCT      []byte    `bson:"email_ct"`
	EmailIV      []byte    `bson:"email_iv"`
	EmailTag     []byte    `bson:"email_tag"`
	EmailLookup  []byte    `bson:"email_lookup"`
	PasswordHash []byte    `bson:"password_hash"`
	PasswordSalt []byte    `bson:"password_salt"`
	MFASecretCT  []byte    `bson:"mfa_secret_ct,omitempty"`
	PINCT        []byte    `bson:"pin_ct,omitempty"`
	PINIV        []byte    `bson:"pin_iv,omitempty"`
	PINTag       []byte    `bson:"pin_tag,omitempty"`
	PINLookup    []byte    `bson:"pin_lookup,omitempty"`
	Status       string    `bson:"status"`
	CreatedAt    time.Time `bson:"created_at"`
	CreatedBy    string    `bson:"created_by"`
	UpdatedAt    time.Time `bson:"updated_at"`
	UpdatedBy    string    `bson:"updated_by"`
}

// toDocument converts a User entity to MongoDB document.
func (r *UserMongoRepo) toDocument(user *authn.User) *userDocument {
	return &userDocument{
		ID:           user.ID.String(),
		Username:     user.Username,
		Name:         user.Name,
		EmailCT:      user.EmailCT,
		EmailIV:      user.EmailIV,
		EmailTag:     user.EmailTag,
		EmailLookup:  user.EmailLookup,
		PasswordHash: user.PasswordHash,
		PasswordSalt: user.PasswordSalt,
		MFASecretCT:  user.MFASecretCT,
		PINCT:        user.PINCT,
		PINIV:        user.PINIV,
		PINTag:       user.PINTag,
		PINLookup:    user.PINLookup,
		Status:       string(user.Status),
		CreatedAt:    user.CreatedAt,
		CreatedBy:    user.CreatedBy,
		UpdatedAt:    user.UpdatedAt,
		UpdatedBy:    user.UpdatedBy,
	}
}

// fromDocument converts a MongoDB document to User entity.
func (r *UserMongoRepo) fromDocument(doc *userDocument) (*authn.User, error) {
	id, err := uuid.Parse(doc.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	return &authn.User{
		ID:           id,
		Username:     doc.Username,
		Name:         doc.Name,
		EmailCT:      doc.EmailCT,
		EmailIV:      doc.EmailIV,
		EmailTag:     doc.EmailTag,
		EmailLookup:  doc.EmailLookup,
		PasswordHash: doc.PasswordHash,
		PasswordSalt: doc.PasswordSalt,
		MFASecretCT:  doc.MFASecretCT,
		PINCT:        doc.PINCT,
		PINIV:        doc.PINIV,
		PINTag:       doc.PINTag,
		PINLookup:    doc.PINLookup,
		Status:       authpkg.UserStatus(doc.Status),
		CreatedAt:    doc.CreatedAt,
		CreatedBy:    doc.CreatedBy,
		UpdatedAt:    doc.UpdatedAt,
		UpdatedBy:    doc.UpdatedBy,
	}, nil
}

// Create creates a new User in MongoDB.
func (r *UserMongoRepo) Create(ctx context.Context, user *authn.User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}

	user.EnsureID()
	user.BeforeCreate()

	doc := r.toDocument(user)

	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("error create user: %w", err)
	}

	return nil
}

// Get retrieves a User by ID from MongoDB.
func (r *UserMongoRepo) Get(ctx context.Context, id uuid.UUID) (*authn.User, error) {
	filter := bson.M{"_id": id.String()}

	var doc userDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get user: %w", err)
	}

	return r.fromDocument(&doc)
}

// GetByEmailLookup retrieves a User by encrypted email lookup hash.
func (r *UserMongoRepo) GetByEmailLookup(ctx context.Context, lookup []byte) (*authn.User, error) {
	filter := bson.M{"email_lookup": lookup}

	var doc userDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get user by email lookup: %w", err)
	}

	return r.fromDocument(&doc)
}

// GetByUsername retrieves a User by username from MongoDB.
func (r *UserMongoRepo) GetByUsername(ctx context.Context, username string) (*authn.User, error) {
	filter := bson.M{"username": username}

	var doc userDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get user by username: %w", err)
	}

	return r.fromDocument(&doc)
}

// GetByPINLookup retrieves a User by PIN lookup hash.
func (r *UserMongoRepo) GetByPINLookup(ctx context.Context, lookup []byte) (*authn.User, error) {
	filter := bson.M{"pin_lookup": lookup}

	var doc userDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get user by PIN lookup: %w", err)
	}

	return r.fromDocument(&doc)
}

// Save updates an existing User in MongoDB.
func (r *UserMongoRepo) Save(ctx context.Context, user *authn.User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}

	user.BeforeUpdate()

	filter := bson.M{"_id": user.ID.String()}
	update := bson.M{
		"$set": bson.M{
			"username":      user.Username,
			"name":          user.Name,
			"email_ct":      user.EmailCT,
			"email_iv":      user.EmailIV,
			"email_tag":     user.EmailTag,
			"email_lookup":  user.EmailLookup,
			"password_hash": user.PasswordHash,
			"password_salt": user.PasswordSalt,
			"mfa_secret_ct": user.MFASecretCT,
			"pin_ct":        user.PINCT,
			"pin_iv":        user.PINIV,
			"pin_tag":       user.PINTag,
			"pin_lookup":    user.PINLookup,
			"status":        string(user.Status),
			"updated_at":    user.UpdatedAt,
			"updated_by":    user.UpdatedBy,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error update user: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user with ID %s not found for update", user.ID.String())
	}

	return nil
}

// Delete performs a soft delete by changing the user status to deleted.
func (r *UserMongoRepo) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id.String()}
	update := bson.M{
		"$set": bson.M{
			"status":     "deleted",
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error delete user: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user with ID %s not found for deletion", id.String())
	}

	return nil
}

// List retrieves all active Users from MongoDB.
func (r *UserMongoRepo) List(ctx context.Context) ([]*authn.User, error) {
	filter := bson.M{"status": bson.M{"$ne": "deleted"}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error query users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*authn.User

	for cursor.Next(ctx) {
		var doc userDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("error decode user document: %w", err)
		}

		user, err := r.fromDocument(&doc)
		if err != nil {
			return nil, fmt.Errorf("error convert document to user: %w", err)
		}

		users = append(users, user)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return users, nil
}

// ListByStatus retrieves Users filtered by status from MongoDB.
func (r *UserMongoRepo) ListByStatus(ctx context.Context, status string) ([]*authn.User, error) {
	filter := bson.M{"status": status}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error query users by status: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*authn.User

	for cursor.Next(ctx) {
		var doc userDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("error decode user document: %w", err)
		}

		user, err := r.fromDocument(&doc)
		if err != nil {
			return nil, fmt.Errorf("error convert document to user: %w", err)
		}

		users = append(users, user)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return users, nil
}

func (r *UserMongoRepo) Log() aqm.Logger {
	return r.logger
}
