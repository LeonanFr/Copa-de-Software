package reserves

import (
	"context"
	"errors"
	"time"

	"copasoftware/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repository struct {
	coll *mongo.Collection
}

func NewRepository(db *database.Mongo) *Repository {
	return &Repository{
		coll: db.DB.Collection("reserves"),
	}
}

func (r *Repository) Insert(ctx context.Context, entry *ReserveEntry) error {
	entry.CreatedAt = time.Now()
	result, err := r.coll.InsertOne(ctx, entry)
	if err == nil {
		entry.ID = result.InsertedID.(primitive.ObjectID)
	}
	return err
}

func (r *Repository) FindByParticipant(ctx context.Context, participantID primitive.ObjectID) (*ReserveEntry, error) {
	var entry ReserveEntry
	err := r.coll.FindOne(ctx, bson.M{"participant_id": participantID}).Decode(&entry)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &entry, err
}

func (r *Repository) FindAll(ctx context.Context) ([]ReserveEntry, error) {
	cursor, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {

		}
	}(cursor, ctx)

	var entries []ReserveEntry
	if err = cursor.All(ctx, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *Repository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *Repository) DeleteByParticipant(ctx context.Context, participantID primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"participant_id": participantID})
	return err
}
