package ranking

import (
	"context"
	"errors"
	"log"
	"time"

	"copasoftware/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct {
	scoreColl           *mongo.Collection
	rankColl            *mongo.Collection
	processedEventsColl *mongo.Collection
}

func NewRepository(db *database.Mongo) *Repository {
	repo := &Repository{
		scoreColl:           db.DB.Collection("scores"),
		rankColl:            db.DB.Collection("ranking"),
		processedEventsColl: db.DB.Collection("processed_events"),
	}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *Repository) ensureIndexes(ctx context.Context) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "team_code", Value: 1},
			{Key: "type", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	_, err := r.processedEventsColl.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Printf("Aviso: não foi possível criar índice único em processed_events: %v", err)
	}
}

func (r *Repository) InsertProcessedEvent(ctx context.Context, event *ProcessedEvent) error {
	event.CreatedAt = time.Now()
	_, err := r.processedEventsColl.InsertOne(ctx, event)
	return err
}

func (r *Repository) InsertScore(ctx context.Context, s *ScoreEntry) error {
	s.CreatedAt = time.Now()
	result, err := r.scoreColl.InsertOne(ctx, s)
	if err == nil {
		s.ID = result.InsertedID.(primitive.ObjectID)
	}
	return err
}

func (r *Repository) FindScoresByTeam(ctx context.Context, teamID primitive.ObjectID) ([]ScoreEntry, error) {
	cursor, err := r.scoreColl.Find(ctx, bson.M{"team_id": teamID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var scores []ScoreEntry
	if err := cursor.All(ctx, &scores); err != nil {
		return nil, err
	}

	return scores, nil
}

func (r *Repository) FindAllScores(ctx context.Context) ([]ScoreEntry, error) {
	cursor, err := r.scoreColl.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {

		}
	}(cursor, ctx)

	var scores []ScoreEntry
	if err = cursor.All(ctx, &scores); err != nil {
		return nil, err
	}
	return scores, nil
}

func (r *Repository) UpsertRanking(ctx context.Context, tr *TeamRanking) error {
	tr.UpdatedAt = time.Now()

	_, err := r.rankColl.UpdateOne(
		ctx,
		bson.M{"_id": tr.TeamID},
		bson.M{
			"$set": bson.M{
				"total":      tr.Total,
				"updated_at": tr.UpdatedAt,
			},
		},
		options.Update().SetUpsert(true),
	)

	return err
}

func (r *Repository) FindRankingByTeam(ctx context.Context, teamID primitive.ObjectID) (*TeamRanking, error) {
	var tr TeamRanking
	err := r.rankColl.FindOne(ctx, bson.M{"_id": teamID}).Decode(&tr)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &tr, err
}

func (r *Repository) FindAllRankings(ctx context.Context) ([]TeamRanking, error) {
	cursor, err := r.rankColl.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {

		}
	}(cursor, ctx)

	var rankings []TeamRanking
	if err = cursor.All(ctx, &rankings); err != nil {
		return nil, err
	}
	return rankings, nil
}

func (r *Repository) DeleteRankingByTeam(ctx context.Context, teamID primitive.ObjectID) error {
	_, err := r.rankColl.DeleteOne(ctx, bson.M{"_id": teamID})
	return err
}
