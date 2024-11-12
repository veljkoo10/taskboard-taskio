package bootstrap

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"os"
	"task-service/db"
	"task-service/models"
)

func InsertInitialTasks() {
	if os.Getenv("ENABLE_BOOTSTRAP") != "true" {
		return
	}

	collection := db.Client.Database("testdb").Collection("tasks")

	count, err := collection.CountDocuments(context.TODO(), bson.D{})
	if err != nil {
		fmt.Println("Error counting users:", err)
		return
	}

	if count > 0 {
		return
	}

	var tasks []interface{}
	for i := 1; i <= 10; i++ {
		task := models.Task{
			Name:   fmt.Sprintf("Ime%d", i),
			Status: fmt.Sprintf("OK%d", i),
		}
		tasks = append(tasks, task)
	}

	_, err = collection.InsertMany(context.TODO(), tasks)
	if err != nil {
		fmt.Println("Error inserting initial tasks:", err)
	} else {
		fmt.Println("Inserted initial tasks")
	}

}

func ClearTasks() {

	collection := db.Client.Database("testdb").Collection("tasks")
	_, err := collection.DeleteMany(context.TODO(), bson.D{})
	if err != nil {
		fmt.Println("Error clearing tasks:", err)
	} else {
		fmt.Println("Cleared tasks from database")
	}
}
