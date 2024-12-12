package models

type Task struct {
	ID          string `bson:"_id,omitempty" json:"id"`
	Name        string `bson:"name" json:"name"`
	Description string `bson:"description" json:"description"`
	Status      string `bson:"status" json:"status"`
}
