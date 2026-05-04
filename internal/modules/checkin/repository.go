package checkin

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
		coll: db.DB.Collection("checkins"),
	}
}

func (r *Repository) Insert(ctx context.Context, c *Checkin) error {
	c.CreatedAt = time.Now()
	res, err := r.coll.InsertOne(ctx, c)
	if err == nil {
		c.ID = res.InsertedID.(primitive.ObjectID)
	}
	return err
}

func (r *Repository) FindByParticipantID(ctx context.Context, participantID primitive.ObjectID) (*Checkin, error) {
	var c Checkin
	err := r.coll.FindOne(ctx, bson.M{"participant_id": participantID}).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &c, err
}

func (r *Repository) FindAll(ctx context.Context) ([]Checkin, error) {
	cursor, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var checkins []Checkin
	if err = cursor.All(ctx, &checkins); err != nil {
		return nil, err
	}
	return checkins, nil
}

func (r *Repository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
