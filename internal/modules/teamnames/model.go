package teamnames

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TeamName struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Name       string              `bson:"name" json:"name"`
	Used       bool                `bson:"used" json:"used"`
	UsedByTeam *primitive.ObjectID `bson:"used_by_team,omitempty" json:"usedByTeam,omitempty"`
	CreatedAt  time.Time           `bson:"created_at" json:"createdAt"`
	UpdatedAt  time.Time           `bson:"updated_at" json:"updatedAt"`
}
