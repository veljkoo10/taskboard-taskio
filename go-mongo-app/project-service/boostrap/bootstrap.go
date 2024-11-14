package bootstrap

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"project-service/db"
	"project-service/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func InsertInitialProjects() {
	if os.Getenv("ENABLE_BOOTSTRAP") != "true" {
		return
	}

	collection := db.Client.Database("testdb").Collection("projects")
	userCollection := db.Client.Database("testdb").Collection("users")

	ClearProjects()

	count, err := collection.CountDocuments(context.TODO(), bson.D{})
	if err != nil {
		fmt.Println("Error counting projects:", err)
		return
	}

	if count > 0 {
		return // If projects already exist, don't insert again
	}

	// Fetch user IDs
	var users []bson.M
	cursor, err := userCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		fmt.Println("Error fetching users:", err)
		return
	}
	if err = cursor.All(context.TODO(), &users); err != nil {
		fmt.Println("Error decoding user IDs:", err)
		return
	}

	var userIDs []string
	for _, user := range users {
		if id, ok := user["_id"].(primitive.ObjectID); ok {
			userIDs = append(userIDs, id.Hex())
		}
	}

	rand.Seed(time.Now().UnixNano())

	var projects []interface{}
	for i := 1; i <= 10; i++ {
		// Randomly select 2 to 5 users for each project
		numUsers := rand.Intn(4) + 2 // Minimum 2, max 5
		projectUsers := make([]string, numUsers)
		for j := 0; j < numUsers; j++ {
			projectUsers[j] = userIDs[rand.Intn(len(userIDs))]
		}

		project := models.Project{
			Title:       fmt.Sprintf("Project %d", i),
			Description: fmt.Sprintf("Description for project %d", i),
			MinPeople:   2,
			MaxPeople:   10,
			Users:       projectUsers,
			Tasks:       []string{},
		}
		projects = append(projects, project)
	}

	_, err = collection.InsertMany(context.TODO(), projects)
	if err != nil {
		fmt.Println("Error inserting initial projects:", err)
	} else {
		fmt.Println("Inserted initial projects with users")
	}
}

func ClearProjects() {
	collection := db.Client.Database("testdb").Collection("projects")
	_, err := collection.DeleteMany(context.TODO(), bson.D{})
	if err != nil {
		fmt.Println("Error clearing projects:", err)
	} else {
		fmt.Println("Cleared projects from database")
	}
}
