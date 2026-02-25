package pendingteams

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

type ParticipantData struct {
	Matricula string `bson:"matricula" json:"matricula"`
	Nome      string `bson:"nome" json:"nome"`
	Semestre  int    `bson:"semestre" json:"semestre"`
}

type PendingTeam struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TeamName     string             `bson:"teamName" json:"teamName"`
	Participants []ParticipantData  `bson:"participants" json:"participants"`
	Status       Status             `bson:"status" json:"status"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
}
