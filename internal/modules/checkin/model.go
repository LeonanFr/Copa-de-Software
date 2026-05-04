package checkin

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Tipo string

const (
	TipoCompetidor Tipo = "competidor"
	TipoOuvinte    Tipo = "ouvinte"
)

type Checkin struct {
	ID            primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ParticipantID *primitive.ObjectID `bson:"participant_id,omitempty" json:"participantId,omitempty"`
	Nome          string              `bson:"nome" json:"nome"`
	Tipo          Tipo                `bson:"tipo" json:"tipo"`
	CreatedAt     time.Time           `bson:"created_at" json:"createdAt"`
}
