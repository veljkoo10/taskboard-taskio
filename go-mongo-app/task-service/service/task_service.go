package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"task-service/db"
	"task-service/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func UpdateTaskStatus(taskID, status string) (*models.Task, error) {
	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return nil, errors.New("invalid task ID format")
	}

	collection := db.Client.Database("testdb").Collection("tasks")
	var task models.Task
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("task not found")
	} else if err != nil {
		return nil, err
	}

	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": taskObjectID},
		bson.M{"$set": bson.M{"status": status}},
	)
	if err != nil {
		return nil, err
	}

	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&task)
	if err != nil {
		return nil, err
	}

	return &task, nil
}
func userExists(userID string) (bool, error) {
	url := fmt.Sprintf("http://user-service:8080/users/%s/exists", userID)
	fmt.Println(userID)
	fmt.Println("Requesting URL:", url) // Debug log

	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("Response Status Code:", resp.StatusCode) // Debug log

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %v", err)
	}
	fmt.Printf("URL: %s, Response: %s\n", url, string(body))

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("received non-OK response: %s", body)
	}

	fmt.Println("Response Body:", string(body))

	var result map[string]bool
	err = json.Unmarshal(body, &result)
	if err != nil {
		return false, fmt.Errorf("failed to parse response body: %v", err)
	}

	exists, ok := result["exists"]
	if !ok {
		return false, fmt.Errorf("response missing 'exists' field")
	}

	return exists, nil
}

func IsUserInProject(projectID string, userID string) (bool, error) {
	// Prvo pozivamo user-service da bismo dobili podatke o korisniku
	url := fmt.Sprintf("http://user-service:8080/users/%s", userID)
	fmt.Println("Requesting URL:", url) // Debug log

	// Napraviti GET zahtev prema user-service da bismo dobili korisničke podatke
	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to get user data: %v", err)
	}
	defer resp.Body.Close()

	// Pročitaj telo odgovora
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %v", err)
	}

	// Ako status nije OK, vraćamo grešku
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("received non-OK response: %s", body)
	}

	// Parsiraj telo odgovora u strukturu User
	var user models.User
	err = json.Unmarshal(body, &user)
	if err != nil {
		return false, fmt.Errorf("failed to parse user data: %v", err)
	}

	// Ispis korisničkih podataka za debugging
	fmt.Println("User data:", user)

	// Sada pozivamo project-service da proverimo da li je ovaj korisnik u projektu
	// Pretpostavljamo da project-service vraća listu korisničkih ID-jeva za određeni projekat
	projectURL := fmt.Sprintf("http://project-service:8080/projects/%s", projectID)
	resp, err = http.Get(projectURL)
	if err != nil {
		return false, fmt.Errorf("failed to get project data: %v", err)
	}
	defer resp.Body.Close()

	// Pročitaj telo odgovora iz project-service
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read project response body: %v", err)
	}

	// Ako status nije OK, vraćamo grešku
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("received non-OK response from project service: %s", body)
	}

	// Parsiraj telo odgovora kao listu korisničkih ID-jeva koji su članovi projekta
	var projectData struct {
		Users []string `json:"users"` // Pretpostavljamo da projekat sadrži listu korisničkih ID-jeva
	}
	err = json.Unmarshal(body, &projectData)
	if err != nil {
		return false, fmt.Errorf("failed to parse project response: %v", err)
	}

	// Proveri da li je userID u listi korisnika u projektu
	for _, pID := range projectData.Users {
		if pID == userID {
			return true, nil // Korisnik je član projekta
		}
	}

	return false, nil // Korisnik nije član projekta
}

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

// GetTasksByProjectID returns tasks for a specific project.
func GetTasksByProjectID(projectID string) ([]models.Task, error) {
	collection := db.Client.Database("testdb").Collection("tasks")
	var tasks []models.Task

	// Convert the projectID to an ObjectID
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, errors.New("invalid project ID format")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query the database for tasks that match the projectID
	cursor, err := collection.Find(ctx, bson.M{"project_id": projectObjectID.Hex()})
	if err != nil {
		return nil, err
	}

	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

// CreateTask creates a new Task with the provided name, description, initial status "pending", and an empty user list.
func CreateTask(projectID, name, description string) (*models.Task, error) {
	// Validate projectID format
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, errors.New("invalid project ID format")
	}

	// Create a new task with the given name and description
	task := models.Task{
		ID:          primitive.NewObjectID(),
		Name:        name,
		Description: description,
		Status:      "pending",
		Users:       []string{}, // Empty list of users
		Project_ID:  projectObjectID.Hex(),
	}

	// Connect to MongoDB collection and insert the task
	collection := db.Client.Database("testdb").Collection("tasks")
	_, err = collection.InsertOne(context.TODO(), task)
	if err != nil {
		return nil, err
	}

	// JSON payload to send to project-service
	payload := map[string]string{
		"task_id":     task.ID.Hex(),
		"project_id":  projectObjectID.Hex(),
		"name":        name,
		"description": description,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Define the URL for the request to project-service
	projectServiceURL := fmt.Sprintf("http://project-service:8080/projects/%s/tasks/%s", projectObjectID.Hex(), task.ID.Hex())

	// Create the HTTP PUT request
	req, err := http.NewRequest("PUT", projectServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request to project-service
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Verify that project-service successfully updated the project
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to update project with new task details")
	}

	// Return the created task
	return &task, nil
}

func AddUserToTask(taskID string, userID string) error {

	userExists, err := userExists(userID)
	if err != nil {
		return err
	}
	if !userExists {
		return errors.New("user not found")
	}

	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return errors.New("invalid task ID format")
	}

	collection := db.Client.Database("testdb").Collection("tasks")
	var project models.Project
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return errors.New("project not found")
	} else if err != nil {
		return err
	}

	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": taskObjectID},
		bson.M{"$addToSet": bson.M{"Users": userID}},
	)
	if err != nil {
		return err
	}

	return nil
}
func RemoveUserFromTask(taskID string, userID string) error {
	userExists, err := userExists(userID)
	if err != nil {
		return err
	}
	if !userExists {
		return errors.New("user not found")
	}

	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return errors.New("invalid task ID format")
	}

	collection := db.Client.Database("testdb").Collection("tasks")
	var task models.Task
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return errors.New("task not found")
	} else if err != nil {
		return err
	}

	userFound := false
	for _, existingUserID := range task.Users {
		if existingUserID == userID {
			userFound = true
			break
		}
	}
	if !userFound {
		return errors.New("user is not a member of this task")
	}

	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": taskObjectID},
		bson.M{"$pull": bson.M{"Users": userID}},
	)
	if err != nil {
		return err
	}

	return nil
}
func GetUsersForTask(taskID string) ([]models.User, error) {
	// Konvertuj task ID u ObjectID
	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return nil, errors.New("invalid task ID format")
	}

	// Pronađi zadatak u kolekciji
	collection := db.Client.Database("testdb").Collection("tasks")
	var task models.Task
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("task not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to fetch task: %v", err)
	}

	// Ako zadatak nema korisnika, vrati praznu listu
	if len(task.Users) == 0 {
		return []models.User{}, nil
	}

	// Pozovi user-service za svakog korisnika u listi
	var users []models.User
	for _, userID := range task.Users {
		url := fmt.Sprintf("http://user-service:8080/users/%s", userID)
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user %s: %v", userID, err)
		}
		defer resp.Body.Close()

		// Proveri statusni kod
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch user %s, status: %d", userID, resp.StatusCode)
		}

		// Parsiraj odgovor u strukturu User
		var user models.User
		err = json.NewDecoder(resp.Body).Decode(&user)
		if err != nil {
			return nil, fmt.Errorf("failed to parse user %s: %v", userID, err)
		}

		// Dodaj korisnika u rezultat
		users = append(users, user)
	}

	return users, nil
}

func GetTaskByID(taskID string) (*models.Task, error) {
	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return nil, errors.New("invalid task ID")
	}

	collection := db.Client.Database("testdb").Collection("tasks")
	var task models.Task
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("task not found")
	} else if err != nil {
		return nil, err
	}

	return &task, nil
}

func IsUserInTask(taskID string, userID string) (bool, error) {
	// Parsiranje taskID-a u ObjectID
	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return false, errors.New("invalid task ID format")
	}

	// Pretraga task-a u bazi
	collection := db.Client.Database("testdb").Collection("tasks")
	var task models.Task
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&task)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return false, errors.New("task not found")
		}
		return false, err
	}

	// Provera da li userID postoji u listi users
	for _, id := range task.Users {
		if id == userID {
			return true, nil
		}
	}

	return false, nil
}
