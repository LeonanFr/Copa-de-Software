package ranking

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ScoreOrigin string

const (
	OriginMatch   ScoreOrigin = "match"
	OriginPenalty ScoreOrigin = "penalty"
	OriginBonus   ScoreOrigin = "bonus"
)

type ScoreEntry struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TeamID      primitive.ObjectID `bson:"team_id" json:"teamId"`
	Value       int                `bson:"value" json:"value"`
	Origin      ScoreOrigin        `bson:"origin" json:"origin"`
	Modality    string             `bson:"modality,omitempty" json:"modality,omitempty"`
	Description string             `bson:"description" json:"description"`
	CreatedAt   time.Time          `bson:"created_at" json:"createdAt"`
}

type TeamRanking struct {
	TeamID    primitive.ObjectID `bson:"_id" json:"teamId"`
	Total     int                `bson:"total" json:"total"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updatedAt"`
}

type RankingEntry struct {
	TeamID   primitive.ObjectID `json:"teamId"`
	TeamName string             `json:"teamName"`
	Total    int                `json:"total"`
}
