package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username string             `bson:"username" json:"username"`
	Password string             `bson:"password" json:"password"`
	Role     string             `bson:"role" json:"role"`
	Name     string             `bson:"name" json:"name"`
	Surname  string             `bson:"surname" json:"surname"`
	Email    string             `bson:"email" json:"email"`
	IsActive bool               `bson:"isActive" json:"isActive"`
}

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
