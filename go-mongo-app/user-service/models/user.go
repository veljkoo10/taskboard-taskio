package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`  // Include the ID field
	Username string             `bson:"username" json:"username"` // Username field
	Password string             `bson:"password" json:"password"` // Password field
	Role     string             `bson:"role" json:"role"`         // Role field
	Name     string             `bson:"name" json:"name"`         // Name field
	Surname  string             `bson:"surname" json:"surname"`   // Surname field
	Email    string             `bson:"email" json:"email"`       // Email field
	IsActive bool               `bson:"isActive" json:"isActive"` // Active status field
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
