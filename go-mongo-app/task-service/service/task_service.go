package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"task-service/db"
	"task-service/models"
	"time"
)

// GetTasks vraća sve zadatke iz baze podataka.
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
		Users:     []string{}, // Prazna lista korisnika
		ProjectID: projectObjectID.Hex(),
	}

	// Povezivanje sa Mongo kolekcijom i čuvanje zadatka
	collection := db.Client.Database("testdb").Collection("tasks")
	_, err = collection.InsertOne(context.TODO(), task)
	if err != nil {
		return nil, err
	}

	// JSON payload koji ćemo poslati ka project-service
	payload := map[string]string{
		"task_id":    task.ID.Hex(),
		"project_id": projectObjectID.Hex(),
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Definisanje URL-a za slanje zahteva ka project-service sa dinamičkom putanjom
	projectServiceURL := fmt.Sprintf("http://project-service:8080/projects/%s/tasks/%s", projectObjectID.Hex(), task.ID.Hex())

	// Kreiranje HTTP PUT zahteva
	req, err := http.NewRequest("PUT", projectServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Slanje zahteva ka project-service
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Provera odgovora - ukoliko project-service nije uspeo da ažurira projekat
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to update project with new task ID")
	}

	// Vraćanje novog task-a
	return &task, nil
}
