package auth

import (
	"context"
	"errors"
	"time"

	"copasoftware/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repository struct {
	coll *mongo.Collection
}

func NewRepository(db *database.Mongo) *Repository {
	return &Repository{
		coll: db.DB.Collection("admins"),
	}
}

func (r *Repository) Insert(ctx context.Context, admin *Admin) error {
	admin.CreatedAt = time.Now()
	_, err := r.coll.InsertOne(ctx, admin)
	return err
}

func (r *Repository) FindByUsername(ctx context.Context, username string) (*Admin, error) {
	var admin Admin
	err := r.coll.FindOne(ctx, bson.M{"username": username}).Decode(&admin)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &admin, err
}
