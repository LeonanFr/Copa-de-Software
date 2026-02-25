package teams

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TeamStatus string

const (
	TeamStatusPending   TeamStatus = "pending"
	TeamStatusApproved  TeamStatus = "approved"
	TeamStatusRejected  TeamStatus = "rejected"
	TeamStatusCancelled TeamStatus = "cancelled"
)

type ParticipantData struct {
	Matricula string `bson:"matricula" json:"matricula"`
	Nome      string `bson:"nome" json:"nome"`
	Semestre  int    `bson:"semestre" json:"semestre"`
}

type Team struct {
	ID              primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name            string               `bson:"name,omitempty" json:"name,omitempty"`
	Code            string               `bson:"code,omitempty" json:"code,omitempty"`
	Participants    []primitive.ObjectID `bson:"participants" json:"participants"`
	ParticipantData []ParticipantData    `bson:"participantData,omitempty" json:"participantData,omitempty"`
	Status          TeamStatus           `bson:"status" json:"status"`
	IsDraw          bool                 `bson:"isDraw" json:"isDraw"`
	CreatedAt       time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time            `bson:"updatedAt" json:"updatedAt"`
}
