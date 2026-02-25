package participants

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Participant struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Matricula string             `bson:"matricula" json:"matricula"`
	Nome      string             `bson:"nome" json:"nome"`
	Semestre  int                `bson:"semestre" json:"semestre"`
	CreatedAt time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updatedAt"`
}
