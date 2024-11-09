package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Task struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Status    string             `bson:"status" json:"status"`
	Users     []string           `bson:"users" json:"users"`
	ProjectID string             `json:"project_id" bson:"project_id"`
}
