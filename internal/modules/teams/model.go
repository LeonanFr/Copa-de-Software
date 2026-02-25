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

type Team struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name         string               `bson:"name,omitempty" json:"name,omitempty"`
	Code         string               `bson:"code,omitempty" json:"code,omitempty"`
	Participants []primitive.ObjectID `bson:"participants" json:"participants"`
	Status       TeamStatus           `bson:"status" json:"status"`
	IsDraw       bool                 `bson:"is_draw" json:"isDraw"`
	CreatedAt    time.Time            `bson:"created_at" json:"createdAt"`
	UpdatedAt    time.Time            `bson:"updated_at" json:"updatedAt"`
}
