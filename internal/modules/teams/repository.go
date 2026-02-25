package teams

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
		coll: db.DB.Collection("teams"),
	}
}

func (r *Repository) Insert(ctx context.Context, t *Team) error {
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	result, err := r.coll.InsertOne(ctx, t)
	if err == nil {
		t.ID = result.InsertedID.(primitive.ObjectID)
	}
	return err
}

func (r *Repository) FindByID(ctx context.Context, id primitive.ObjectID) (*Team, error) {
	var t Team
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &t, err
}

func (r *Repository) FindByParticipant(ctx context.Context, participantID primitive.ObjectID) ([]Team, error) {
	cursor, err := r.coll.Find(ctx, bson.M{
		"participants": participantID,
		"status":       bson.M{"$ne": TeamStatusCancelled},
	})
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {

		}
	}(cursor, ctx)

	var teams []Team
	if err = cursor.All(ctx, &teams); err != nil {
		return nil, err
	}
	return teams, nil
}

func (r *Repository) FindAll(ctx context.Context) ([]Team, error) {
	cursor, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {

		}
	}(cursor, ctx)

	var teams []Team
	if err = cursor.All(ctx, &teams); err != nil {
		return nil, err
	}
	return teams, nil
}

func (r *Repository) Update(ctx context.Context, t *Team) error {
	t.UpdatedAt = time.Now()
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": t.ID}, t)
	return err
}

func (r *Repository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status TeamStatus) error {
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}
