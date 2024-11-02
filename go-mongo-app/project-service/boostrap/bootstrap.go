package bootstrap

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"os"
	"project-service/db"
	"project-service/models"
)

func InsertInitialProjects() {
	if os.Getenv("ENABLE_BOOTSTRAP") != "true" {
		return
	}

	collection := db.Client.Database("testdb").Collection("projects")

	count, err := collection.CountDocuments(context.TODO(), bson.D{})
	if err != nil {
		fmt.Println("Error counting projects:", err)
		return
	}

	if count > 0 {
		return
	}

	var projects []interface{}
	for i := 1; i <= 10; i++ {
		project := models.Project{
			Title:       fmt.Sprintf("Project %d", i),
			Description: fmt.Sprintf("Description for project %d", i),
			Owner:       fmt.Sprintf("Owner %d", i),
		}
		projects = append(projects, project)
	}

	_, err = collection.InsertMany(context.TODO(), projects)
	if err != nil {
		fmt.Println("Error inserting initial projects:", err)
	} else {
		fmt.Println("Inserted initial projects")
	}
}

func ClearProjects() {
	if os.Getenv("ENABLE_BOOTSTRAP") == "true" {
		return
	}

	collection := db.Client.Database("testdb").Collection("projects")
	_, err := collection.DeleteMany(context.TODO(), bson.D{})
	if err != nil {
		fmt.Println("Error clearing projects:", err)
	} else {
		fmt.Println("Cleared projects from database")
	}
}
