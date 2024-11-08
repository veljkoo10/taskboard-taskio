package bootstrap

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
	"os"
	"user-service/db"
	"user-service/models"
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
		return
	}

	var users []interface{}
	for i := 1; i <= 10; i++ {
		password := fmt.Sprintf("password%d", i)

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			fmt.Println("Error hashing password:", err)
			return
		}

		user := models.User{
			Username: fmt.Sprintf("user%d", i),
			Password: string(hashedPassword),
			Role:     "user",
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
		fmt.Println("Inserted initial users")
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
