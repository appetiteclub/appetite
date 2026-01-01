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

	"github.com/appetiteclub/appetite/services/table/internal/tables"
)

type TableRepo struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	logger     apt.Logger
	config     *apt.Config
}

// tableDocument represents the MongoDB document structure.
type tableDocument struct {
	ID          string     `bson:"_id"`
	Number      string     `bson:"number"`
	Status      string     `bson:"status"`
	GuestCount  int        `bson:"guest_count"`
	AssignedTo  *string    `bson:"assigned_to,omitempty"`
	Notes       []bson.M   `bson:"notes,omitempty"`
	CurrentBill *bson.M    `bson:"current_bill,omitempty"`
	CreatedAt   time.Time  `bson:"created_at"`
	CreatedBy   string     `bson:"created_by"`
	UpdatedAt   time.Time  `bson:"updated_at"`
	UpdatedBy   string     `bson:"updated_by"`
}

func NewTableRepo(config *apt.Config, logger apt.Logger) *TableRepo {
	if logger == nil {
		logger = apt.NewNoopLogger()
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

// toDocument converts a Table entity to MongoDB document.
func (r *TableRepo) toDocument(table *tables.Table) *tableDocument {
	doc := &tableDocument{
		ID:         table.ID.String(),
		Number:     table.Number,
		Status:     table.Status,
		GuestCount: table.GuestCount,
		CreatedAt:  table.CreatedAt,
		CreatedBy:  table.CreatedBy,
		UpdatedAt:  table.UpdatedAt,
		UpdatedBy:  table.UpdatedBy,
	}

	if table.AssignedTo != nil {
		assignedToStr := table.AssignedTo.String()
		doc.AssignedTo = &assignedToStr
	}

	if table.Notes != nil && len(table.Notes) > 0 {
		doc.Notes = make([]bson.M, len(table.Notes))
		for i, note := range table.Notes {
			doc.Notes[i] = bson.M{
				"id":         note.ID.String(),
				"content":    note.Content,
				"created_at": note.CreatedAt,
				"created_by": note.CreatedBy,
			}
		}
	}

	if table.CurrentBill != nil {
		billDoc := bson.M{
			"subtotal": table.CurrentBill.Subtotal,
			"tax":      table.CurrentBill.Tax,
			"tip":      table.CurrentBill.Tip,
			"total":    table.CurrentBill.Total,
		}
		doc.CurrentBill = &billDoc
	}

	return doc
}

// fromDocument converts a MongoDB document to Table entity.
func (r *TableRepo) fromDocument(doc *tableDocument) (*tables.Table, error) {
	id, err := uuid.Parse(doc.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid table ID format: %w", err)
	}

	table := &tables.Table{
		ID:         id,
		Number:     doc.Number,
		Status:     doc.Status,
		GuestCount: doc.GuestCount,
		CreatedAt:  doc.CreatedAt,
		CreatedBy:  doc.CreatedBy,
		UpdatedAt:  doc.UpdatedAt,
		UpdatedBy:  doc.UpdatedBy,
	}

	if doc.AssignedTo != nil && *doc.AssignedTo != "" {
		assignedTo, err := uuid.Parse(*doc.AssignedTo)
		if err == nil {
			table.AssignedTo = &assignedTo
		}
	}

	if doc.Notes != nil && len(doc.Notes) > 0 {
		table.Notes = make([]tables.Note, 0, len(doc.Notes))
		for _, noteDoc := range doc.Notes {
			note := tables.Note{}
			if noteIDStr, ok := noteDoc["id"].(string); ok {
				noteID, _ := uuid.Parse(noteIDStr)
				note.ID = noteID
			}
			if content, ok := noteDoc["content"].(string); ok {
				note.Content = content
			}
			if createdAt, ok := noteDoc["created_at"].(time.Time); ok {
				note.CreatedAt = createdAt
			}
			if createdBy, ok := noteDoc["created_by"].(string); ok {
				note.CreatedBy = createdBy
			}
			table.Notes = append(table.Notes, note)
		}
	} else {
		table.Notes = []tables.Note{}
	}

	if doc.CurrentBill != nil {
		billDoc := *doc.CurrentBill
		bill := &tables.Bill{}
		if subtotal, ok := billDoc["subtotal"].(float64); ok {
			bill.Subtotal = subtotal
		}
		if tax, ok := billDoc["tax"].(float64); ok {
			bill.Tax = tax
		}
		if tip, ok := billDoc["tip"].(float64); ok {
			bill.Tip = tip
		}
		if total, ok := billDoc["total"].(float64); ok {
			bill.Total = total
		}
		table.CurrentBill = bill
	}

	return table, nil
}

func (r *TableRepo) Create(ctx context.Context, table *tables.Table) error {
	if table == nil {
		return fmt.Errorf("table is nil")
	}

	doc := r.toDocument(table)
	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		return fmt.Errorf("cannot create table: %w", err)
	}

	return nil
}

func (r *TableRepo) Get(ctx context.Context, id uuid.UUID) (*tables.Table, error) {
	var doc tableDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id.String()}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot get table: %w", err)
	}
	return r.fromDocument(&doc)
}

func (r *TableRepo) GetByNumber(ctx context.Context, number string) (*tables.Table, error) {
	var doc tableDocument
	err := r.collection.FindOne(ctx, bson.M{"number": number}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot get table by number: %w", err)
	}
	return r.fromDocument(&doc)
}

func (r *TableRepo) List(ctx context.Context) ([]*tables.Table, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("cannot list tables: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []tableDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("cannot decode tables: %w", err)
	}

	result := make([]*tables.Table, 0, len(docs))
	for _, doc := range docs {
		table, err := r.fromDocument(&doc)
		if err != nil {
			return nil, err
		}
		result = append(result, table)
	}

	return result, nil
}

func (r *TableRepo) ListByStatus(ctx context.Context, status string) ([]*tables.Table, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, fmt.Errorf("cannot list tables by status: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []tableDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("cannot decode tables: %w", err)
	}

	result := make([]*tables.Table, 0, len(docs))
	for _, doc := range docs {
		table, err := r.fromDocument(&doc)
		if err != nil {
			return nil, err
		}
		result = append(result, table)
	}

	return result, nil
}

func (r *TableRepo) Save(ctx context.Context, table *tables.Table) error {
	if table == nil {
		return fmt.Errorf("table is nil")
	}

	doc := r.toDocument(table)
	filter := bson.M{"_id": table.ID.String()}
	update := bson.M{"$set": doc}

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
