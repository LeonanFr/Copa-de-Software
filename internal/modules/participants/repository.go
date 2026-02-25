package participants

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
		coll: db.DB.Collection("participants"),
	}
}

func (r *Repository) Insert(ctx context.Context, p *Participant) error {
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	result, err := r.coll.InsertOne(ctx, p)
	if err == nil {
		p.ID = result.InsertedID.(primitive.ObjectID)
	}
	return err
}

func (r *Repository) FindByMatricula(ctx context.Context, matricula string) (*Participant, error) {
	var p Participant
	err := r.coll.FindOne(ctx, bson.M{"matricula": matricula}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &p, err
}

func (r *Repository) FindByID(ctx context.Context, id primitive.ObjectID) (*Participant, error) {
	var p Participant
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &p, err
}

func (r *Repository) FindAll(ctx context.Context) ([]Participant, error) {
	cursor, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {

		}
	}(cursor, ctx)

	var participants []Participant
	if err = cursor.All(ctx, &participants); err != nil {
		return nil, err
	}
	return participants, nil
}

func (r *Repository) Update(ctx context.Context, p *Participant) error {
	p.UpdatedAt = time.Now()
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": p.ID}, p)
	return err
}

func (r *Repository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
