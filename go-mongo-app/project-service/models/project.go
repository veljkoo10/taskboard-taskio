package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Project struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ManagerID       string             `bson:"manager_id" json:"manager_id"`
	Title           string             `bson:"title" json:"title"`
	Description     string             `bson:"description" json:"description"`
	ExpectedEndDate string             `bson:"expected_end_date" json:"expected_end_date"`
	MinPeople       int                `bson:"min_people" json:"min_people"`
	MaxPeople       int                `bson:"max_people" json:"max_people"`
	Users           []string           `bson:"users" json:"users"`
	Tasks           []string           `bson:"tasks" json:"tasks"`
}
