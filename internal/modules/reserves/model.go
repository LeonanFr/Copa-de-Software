package reserves

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReserveEntry struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Participant primitive.ObjectID `bson:"participant_id" json:"participantId"`
	Semestre    int                `bson:"semestre" json:"semestre"`
	CreatedAt   time.Time          `bson:"created_at" json:"createdAt"`
}
