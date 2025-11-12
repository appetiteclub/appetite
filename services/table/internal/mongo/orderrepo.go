package mongo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/appetiteclub/appetite/services/table/internal/tables"
)

type OrderRepo struct {
	collection *mongo.Collection
}

func NewOrderRepo(db *mongo.Database) *OrderRepo {
	return &OrderRepo{
		collection: db.Collection("orders"),
	}
}

func (r *OrderRepo) Create(ctx context.Context, order *tables.Order) error {
	if order == nil {
		return fmt.Errorf("order is nil")
	}

	if _, err := r.collection.InsertOne(ctx, order); err != nil {
		return fmt.Errorf("cannot create order: %w", err)
	}

	return nil
}

func (r *OrderRepo) Get(ctx context.Context, id uuid.UUID) (*tables.Order, error) {
	var order tables.Order
	err := r.collection.FindOne(ctx, bson.M{"_id": id.String()}).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot get order: %w", err)
	}
	return &order, nil
}

func (r *OrderRepo) ListByTable(ctx context.Context, tableID uuid.UUID) ([]*tables.Order, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"table_id": tableID.String()})
	if err != nil {
		return nil, fmt.Errorf("cannot list orders by table: %w", err)
	}
	defer cursor.Close(ctx)

	var result []*tables.Order
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cannot decode orders: %w", err)
	}

	return result, nil
}

func (r *OrderRepo) ListByStatus(ctx context.Context, status string) ([]*tables.Order, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, fmt.Errorf("cannot list orders by status: %w", err)
	}
	defer cursor.Close(ctx)

	var result []*tables.Order
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cannot decode orders: %w", err)
	}

	return result, nil
}

func (r *OrderRepo) Save(ctx context.Context, order *tables.Order) error {
	if order == nil {
		return fmt.Errorf("order is nil")
	}

	filter := bson.M{"_id": order.ID.String()}
	update := bson.M{"$set": order}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("cannot update order: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("order not found")
	}

	return nil
}

func (r *OrderRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id.String()})
	if err != nil {
		return fmt.Errorf("cannot delete order: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("order not found")
	}

	return nil
}
