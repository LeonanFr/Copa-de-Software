package teamnames

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
		coll: db.DB.Collection("team_names"),
	}
}

func (r *Repository) Insert(ctx context.Context, tn *TeamName) error {
	tn.CreatedAt = time.Now()
	tn.UpdatedAt = time.Now()
	result, err := r.coll.InsertOne(ctx, tn)
	if err == nil {
		tn.ID = result.InsertedID.(primitive.ObjectID)
	}
	return err
}

func (r *Repository) FindAvailable(ctx context.Context) ([]TeamName, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"used": false})
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {

		}
	}(cursor, ctx)

	var names []TeamName
	if err = cursor.All(ctx, &names); err != nil {
		return nil, err
	}
	return names, nil
}

func (r *Repository) MarkAsUsed(ctx context.Context, id primitive.ObjectID, teamID primitive.ObjectID) error {
	update := bson.M{
		"$set": bson.M{
			"used":         true,
			"used_by_team": teamID,
			"updated_at":   time.Now(),
		},
	}
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id, "used": false}, update)
	return err
}

func (r *Repository) AssignToTeam(ctx context.Context, id primitive.ObjectID, teamID primitive.ObjectID) error {
	update := bson.M{
		"$set": bson.M{
			"used_by_team": teamID,
			"updated_at":   time.Now(),
		},
	}
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *Repository) MarkAsAvailableByTeam(ctx context.Context, teamID primitive.ObjectID) error {
	update := bson.M{
		"$set": bson.M{
			"used":         false,
			"used_by_team": nil,
			"updated_at":   time.Now(),
		},
	}
	_, err := r.coll.UpdateOne(ctx, bson.M{"used_by_team": teamID}, update)
	return err
}

func (r *Repository) MarkAsAvailableByID(ctx context.Context, id primitive.ObjectID) error {
	update := bson.M{
		"$set": bson.M{
			"used":         false,
			"used_by_team": nil,
			"updated_at":   time.Now(),
		},
	}
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *Repository) FindByName(ctx context.Context, name string) (*TeamName, error) {
	var tn TeamName
	err := r.coll.FindOne(ctx, bson.M{"name": name}).Decode(&tn)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &tn, err
}

func (r *Repository) FindByID(ctx context.Context, id primitive.ObjectID) (*TeamName, error) {
	var tn TeamName
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&tn)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &tn, err
}
