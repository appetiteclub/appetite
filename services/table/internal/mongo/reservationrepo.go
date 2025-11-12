package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/appetiteclub/appetite/services/table/internal/tables"
)

type ReservationRepo struct {
	collection *mongo.Collection
}

func NewReservationRepo(db *mongo.Database) *ReservationRepo {
	return &ReservationRepo{
		collection: db.Collection("reservations"),
	}
}

func (r *ReservationRepo) Create(ctx context.Context, reservation *tables.Reservation) error {
	if reservation == nil {
		return fmt.Errorf("reservation is nil")
	}

	if _, err := r.collection.InsertOne(ctx, reservation); err != nil {
		return fmt.Errorf("cannot create reservation: %w", err)
	}

	return nil
}

func (r *ReservationRepo) Get(ctx context.Context, id uuid.UUID) (*tables.Reservation, error) {
	var reservation tables.Reservation
	err := r.collection.FindOne(ctx, bson.M{"_id": id.String()}).Decode(&reservation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot get reservation: %w", err)
	}
	return &reservation, nil
}

func (r *ReservationRepo) List(ctx context.Context) ([]*tables.Reservation, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("cannot list reservations: %w", err)
	}
	defer cursor.Close(ctx)

	var result []*tables.Reservation
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cannot decode reservations: %w", err)
	}

	return result, nil
}

func (r *ReservationRepo) ListByDate(ctx context.Context, date string) ([]*tables.Reservation, error) {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	startOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, parsedDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	filter := bson.M{
		"reserved_for": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("cannot list reservations by date: %w", err)
	}
	defer cursor.Close(ctx)

	var result []*tables.Reservation
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cannot decode reservations: %w", err)
	}

	return result, nil
}

func (r *ReservationRepo) ListByTable(ctx context.Context, tableID uuid.UUID) ([]*tables.Reservation, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"table_id": tableID.String()})
	if err != nil {
		return nil, fmt.Errorf("cannot list reservations by table: %w", err)
	}
	defer cursor.Close(ctx)

	var result []*tables.Reservation
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cannot decode reservations: %w", err)
	}

	return result, nil
}

func (r *ReservationRepo) Save(ctx context.Context, reservation *tables.Reservation) error {
	if reservation == nil {
		return fmt.Errorf("reservation is nil")
	}

	filter := bson.M{"_id": reservation.ID.String()}
	update := bson.M{"$set": reservation}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("cannot update reservation: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("reservation not found")
	}

	return nil
}

func (r *ReservationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id.String()})
	if err != nil {
		return fmt.Errorf("cannot delete reservation: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("reservation not found")
	}

	return nil
}
