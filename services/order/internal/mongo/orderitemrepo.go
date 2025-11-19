package mongo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/appetiteclub/appetite/services/order/internal/order"
)

type OrderItemRepo struct {
	collection *mongo.Collection
}

func NewOrderItemRepo(db *mongo.Database) *OrderItemRepo {
	return &OrderItemRepo{
		collection: db.Collection("order_items"),
	}
}

func (r *OrderItemRepo) Create(ctx context.Context, item *order.OrderItem) error {
	if item == nil {
		return fmt.Errorf("order item is nil")
	}

	if _, err := r.collection.InsertOne(ctx, item); err != nil {
		return fmt.Errorf("cannot create order item: %w", err)
	}

	return nil
}

func (r *OrderItemRepo) Get(ctx context.Context, id uuid.UUID) (*order.OrderItem, error) {
	var item order.OrderItem
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot get order item: %w", err)
	}
	return &item, nil
}

func (r *OrderItemRepo) ListByOrder(ctx context.Context, orderID uuid.UUID) ([]*order.OrderItem, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"order_id": orderID})
	if err != nil {
		return nil, fmt.Errorf("cannot list order items by order: %w", err)
	}
	defer cursor.Close(ctx)

	var result []*order.OrderItem
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cannot decode order items: %w", err)
	}

	return result, nil
}

func (r *OrderItemRepo) ListByGroup(ctx context.Context, groupID uuid.UUID) ([]*order.OrderItem, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"group_id": groupID})
	if err != nil {
		return nil, fmt.Errorf("cannot list order items by group: %w", err)
	}
	defer cursor.Close(ctx)

	var result []*order.OrderItem
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cannot decode order items: %w", err)
	}

	return result, nil
}

func (r *OrderItemRepo) Save(ctx context.Context, item *order.OrderItem) error {
	if item == nil {
		return fmt.Errorf("order item is nil")
	}

	filter := bson.M{"_id": item.ID}
	update := bson.M{"$set": item}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("cannot update order item: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("order item not found")
	}

	return nil
}

func (r *OrderItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("cannot delete order item: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("order item not found")
	}

	return nil
}
