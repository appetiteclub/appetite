package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/appetiteclub/appetite/services/kitchen/internal/kitchen"
	"github.com/aquamarinepk/aqm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TicketRepo struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	logger     aqm.Logger
	config     *aqm.Config
}

func NewTicketRepo(config *aqm.Config, logger aqm.Logger) *TicketRepo {
	return &TicketRepo{
		logger: logger,
		config: config,
	}
}

func (r *TicketRepo) Start(ctx context.Context) error {
	mongoURL, _ := r.config.GetString("db.mongo.url")
	if mongoURL == "" {
		mongoURL = "mongodb://localhost:27017"
	}

	dbName, _ := r.config.GetString("db.mongo.name")
	if dbName == "" {
		dbName = "appetite_kitchen"
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
	r.collection = r.db.Collection("tickets")

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "order_item_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	if _, err := r.collection.Indexes().CreateOne(ctx, indexModel); err != nil {
		return fmt.Errorf("cannot create order_item_id index: %w", err)
	}

	stationIndexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "station", Value: 1}},
	}
	if _, err := r.collection.Indexes().CreateOne(ctx, stationIndexModel); err != nil {
		return fmt.Errorf("cannot create station index: %w", err)
	}

	statusIndexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "status", Value: 1}},
	}
	if _, err := r.collection.Indexes().CreateOne(ctx, statusIndexModel); err != nil {
		return fmt.Errorf("cannot create status index: %w", err)
	}

	r.logger.Infof("Connected to MongoDB: %s, database: %s, collection: tickets", mongoURL, dbName)
	return nil
}

func (r *TicketRepo) GetDatabase() *mongo.Database {
	return r.db
}

func (r *TicketRepo) Stop(ctx context.Context) error {
	if r.client != nil {
		if err := r.client.Disconnect(ctx); err != nil {
			return fmt.Errorf("cannot disconnect from MongoDB: %w", err)
		}
		r.logger.Info("Disconnected from MongoDB")
	}
	return nil
}

func (r *TicketRepo) Create(ctx context.Context, t *kitchen.Ticket) error {
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	t.ModelVersion = 1

	_, err := r.collection.InsertOne(ctx, t)
	if err != nil {
		return fmt.Errorf("cannot insert ticket: %w", err)
	}
	return nil
}

func (r *TicketRepo) Update(ctx context.Context, t *kitchen.Ticket) error {
	t.UpdatedAt = time.Now()

	filter := bson.M{"_id": t.ID}
	update := bson.M{"$set": t}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("cannot update ticket: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("ticket not found")
	}

	return nil
}

func (r *TicketRepo) FindByID(ctx context.Context, id kitchen.TicketID) (*kitchen.Ticket, error) {
	var ticket kitchen.Ticket
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&ticket)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, fmt.Errorf("cannot find ticket: %w", err)
	}
	return &ticket, nil
}

func (r *TicketRepo) FindByOrderItemID(ctx context.Context, id kitchen.OrderItemID) (*kitchen.Ticket, error) {
	var ticket kitchen.Ticket
	err := r.collection.FindOne(ctx, bson.M{"order_item_id": id}).Decode(&ticket)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot find ticket by order_item_id: %w", err)
	}
	return &ticket, nil
}

func (r *TicketRepo) List(ctx context.Context, filter kitchen.TicketFilter) ([]kitchen.Ticket, error) {
	query := bson.M{}

	if filter.Station != nil {
		query["station"] = *filter.Station
	}

	if filter.Status != nil {
		query["status"] = *filter.Status
	}

	if filter.OrderID != nil {
		query["order_id"] = *filter.OrderID
	}

	if filter.OrderItemID != nil {
		query["order_item_id"] = *filter.OrderItemID
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}

	if filter.Offset > 0 {
		opts.SetSkip(int64(filter.Offset))
	}

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("cannot find tickets: %w", err)
	}
	defer cursor.Close(ctx)

	var tickets []kitchen.Ticket
	if err := cursor.All(ctx, &tickets); err != nil {
		return nil, fmt.Errorf("cannot decode tickets: %w", err)
	}

	return tickets, nil
}
