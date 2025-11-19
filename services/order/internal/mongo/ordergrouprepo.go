package mongo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/appetiteclub/appetite/services/order/internal/order"
)

type OrderGroupRepo struct {
	collection *mongo.Collection
}

func NewOrderGroupRepo(db *mongo.Database) *OrderGroupRepo {
	return &OrderGroupRepo{collection: db.Collection("order_groups")}
}

func (r *OrderGroupRepo) Create(ctx context.Context, group *order.OrderGroup) error {
	if group == nil {
		return fmt.Errorf("order group is nil")
	}
	_, err := r.collection.InsertOne(ctx, group)
	if err != nil {
		return fmt.Errorf("cannot create order group: %w", err)
	}
	return nil
}

func (r *OrderGroupRepo) ListByOrder(ctx context.Context, orderID uuid.UUID) ([]*order.OrderGroup, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"order_id": orderID})
	if err != nil {
		return nil, fmt.Errorf("cannot list order groups: %w", err)
	}
	defer cursor.Close(ctx)

	var groups []*order.OrderGroup
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, fmt.Errorf("cannot decode order groups: %w", err)
	}
	return groups, nil
}

func (r *OrderGroupRepo) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("cannot delete order group: %w", err)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("order group not found")
	}
	return nil
}
