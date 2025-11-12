package mongo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/appetiteclub/appetite/services/table/internal/tables"
)

type GroupRepo struct {
	collection *mongo.Collection
}

func NewGroupRepo(db *mongo.Database) *GroupRepo {
	return &GroupRepo{
		collection: db.Collection("groups"),
	}
}

func (r *GroupRepo) Create(ctx context.Context, group *tables.Group) error {
	if group == nil {
		return fmt.Errorf("group is nil")
	}

	if _, err := r.collection.InsertOne(ctx, group); err != nil {
		return fmt.Errorf("cannot create group: %w", err)
	}

	return nil
}

func (r *GroupRepo) Get(ctx context.Context, id uuid.UUID) (*tables.Group, error) {
	var group tables.Group
	err := r.collection.FindOne(ctx, bson.M{"_id": id.String()}).Decode(&group)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot get group: %w", err)
	}
	return &group, nil
}

func (r *GroupRepo) ListByTable(ctx context.Context, tableID uuid.UUID) ([]*tables.Group, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"table_id": tableID.String()})
	if err != nil {
		return nil, fmt.Errorf("cannot list groups by table: %w", err)
	}
	defer cursor.Close(ctx)

	var result []*tables.Group
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cannot decode groups: %w", err)
	}

	return result, nil
}

func (r *GroupRepo) Save(ctx context.Context, group *tables.Group) error {
	if group == nil {
		return fmt.Errorf("group is nil")
	}

	filter := bson.M{"_id": group.ID.String()}
	update := bson.M{"$set": group}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("cannot update group: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("group not found")
	}

	return nil
}

func (r *GroupRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id.String()})
	if err != nil {
		return fmt.Errorf("cannot delete group: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("group not found")
	}

	return nil
}
