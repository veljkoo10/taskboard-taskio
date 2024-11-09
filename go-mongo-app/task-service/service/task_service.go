package service

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"task-service/db"
	"task-service/models"
	"time"
)

func GetTasks() ([]models.Task, error) {
	collection := db.Client.Database("testdb").Collection("tasks")
	var tasks []models.Task

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

// CreateTask stvara novi Task sa početnim statusom "pending" i praznom listom korisnika.
func CreateTask(projectID string) (*models.Task, error) {
	// Proveri validnost projectID
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, errors.New("invalid project ID format")
	}

	// Kreiraj novi task sa početnim vrednostima
	task := models.Task{
		ID:        primitive.NewObjectID(),
		Status:    "pending",
		Users:     []string{},
		ProjectID: projectObjectID.Hex(),
	}

	// Povezivanje sa Mongo kolekcijom i čuvanje zadatka
	collection := db.Client.Database("testdb").Collection("tasks")
	_, err = collection.InsertOne(context.TODO(), task)
	if err != nil {
		return nil, err
	}

	return &task, nil
}
