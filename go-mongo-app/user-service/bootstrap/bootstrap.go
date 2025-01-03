package bootstrap

import (
	"context"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"os"
	"user-service/db"
	"user-service/models"

	"go.mongodb.org/mongo-driver/bson"
)

func InsertInitialUsers() {
	if os.Getenv("ENABLE_BOOTSTRAP") != "true" {
		return
	}

	collection := db.Client.Database("testdb").Collection("users")

	count, err := collection.CountDocuments(context.TODO(), bson.D{})
	if err != nil {
		fmt.Println("Error counting users:", err)
		return
	}

	if count > 0 {
		return // Skip if users already exist
	}

	// Dodaj unapred definisane korisnike
	var users []interface{}

	// Dodavanje korisnika "aca"
	hashedPasswordAca, err := bcrypt.GenerateFromPassword([]byte("Aca2024!"), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Error hashing password for aca:", err)
		return
	}
	acaUser := models.User{
		Username: "aca",
		Password: string(hashedPasswordAca),
		Role:     "Manager",
		Name:     "Aca",
		Surname:  "Admin",
		Email:    "aca@example.com",
		IsActive: true,
	}
	users = append(users, acaUser)

	// Dodavanje korisnika "ana"
	hashedPasswordAna, err := bcrypt.GenerateFromPassword([]byte("Ana2024!"), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Error hashing password for ana:", err)
		return
	}
	anaUser := models.User{
		Username: "ana",
		Password: string(hashedPasswordAna),
		Role:     "Member",
		Name:     "Ana",
		Surname:  "User",
		Email:    "ana@example.com",
		IsActive: true,
	}
	users = append(users, anaUser)

	// Dodavanje drugih korisnika kao što je u originalu
	for i := 1; i <= 10; i++ {
		user := models.User{
			Username: fmt.Sprintf("user%d", i),
			Password: fmt.Sprintf("password%d", i),
			Role:     "member",
			Name:     fmt.Sprintf("Name%d", i),
			Surname:  fmt.Sprintf("Surname%d", i),
			Email:    fmt.Sprintf("user%d@example.com", i),
			IsActive: true,
		}
		users = append(users, user)
	}

	_, err = collection.InsertMany(context.TODO(), users)
	if err != nil {
		fmt.Println("Error inserting initial users:", err)
	} else {
		fmt.Println("Inserted initial users including 'aca' and 'ana'")
	}
}

func ClearUsers() {

	collection := db.Client.Database("testdb").Collection("users")
	_, err := collection.DeleteMany(context.TODO(), bson.D{})
	if err != nil {
		fmt.Println("Error clearing users:", err)
	} else {
		fmt.Println("Cleared users from database")
	}
}
