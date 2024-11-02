package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// User represents a user in the system.
type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"` // Include the ID field
	Username string             `json:"username"`
	Password string             `json:"password"`
	Role     string             `json:"role"`
	Name     string             `json:"name"`
	Surname  string             `json:"surname"`
	Email    string             `json:"email"`
	IsActive bool               `json:"isActive"`
}

// NewUser creates a new User instance.
func NewUser(username, password, role, name, surname, email string) User {
	return User{
		Username: username,
		Password: password,
		Role:     role,
		Name:     name,
		Surname:  surname,
		Email:    email,
		IsActive: false,
	}
}
