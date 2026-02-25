package pendingteams

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
		coll: db.DB.Collection("pending_teams"),
	}
}

func (r *Repository) Insert(ctx context.Context, pt *PendingTeam) error {
	pt.CreatedAt = time.Now()
	pt.UpdatedAt = time.Now()
	pt.Status = StatusPending
	result, err := r.coll.InsertOne(ctx, pt)
	if err == nil {
		pt.ID = result.InsertedID.(primitive.ObjectID)
	}
	return err
}

func (r *Repository) FindByID(ctx context.Context, id primitive.ObjectID) (*PendingTeam, error) {
	var pt PendingTeam
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&pt)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &pt, err
}

func (r *Repository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status Status) error {
	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"updatedAt": time.Now(),
		},
	}
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *Repository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
