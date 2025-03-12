package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/colinmarc/hdfs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"task-service/db"
	"task-service/models"
	"time"
)

// isValidUserID proverava da li korisnički ID sadrži samo dozvoljene karaktere
func isValidUserID(userID string) bool {
	// Ovaj primer dopušta samo alfanumeričke karaktere i crtica (možete prilagoditi prema potrebama)
	for _, char := range userID {
		if !(char >= 'a' && char <= 'z') && !(char >= 'A' && char <= 'Z') && !(char >= '0' && char <= '9') && char != '-' {
			return false
		}
	}
	return true
}

// Funkcija koja escapuje HTML kako bi sprečila XSS napade prilikom prikaza unosa
func EscapeHTML(input string) string {
	// Koristi standardnu funkciju za escape HTML-a
	return strings.ReplaceAll(input, "<", "&lt;")
}

// UpdateTaskStatus ažurira status zadatka u bazi podataka
func UpdateTaskStatus(taskID, status string, token string) (*models.Task, error) {
	// Validacija i konverzija taskID-a u ObjectID
	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return nil, fmt.Errorf("invalid task ID format: %w", err)
	}

	// Validacija statusa
	status = SanitizeInput(status)
	allowedStatuses := map[string]bool{
		"pending":          true,
		"work in progress": true,
		"done":             true,
	}

	if !allowedStatuses[status] {
		return nil, errors.New("invalid status value")
	}

	// Pretraga zadatka u bazi
	collection := db.Client.Database("testdb").Collection("tasks")
	var task models.Task
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("task not found")
		}
		return nil, fmt.Errorf("error finding task: %w", err)
	}

	// Provera zavisnosti
	dependencies, err := GetDependenciesFromWorkflowService(taskID, token)
	if err != nil {
		return nil, fmt.Errorf("error fetching dependencies: %w", err)
	}

	// Provera statusa zavisnih zadataka
	for _, dependencyTaskID := range dependencies.DependencyTasks {
		depTaskObjectID, err := primitive.ObjectIDFromHex(dependencyTaskID)
		if err != nil {
			return nil, fmt.Errorf("invalid dependency task ID: %s", dependencyTaskID)
		}

		var dependentTask models.Task
		err = collection.FindOne(context.TODO(), bson.M{"_id": depTaskObjectID}).Decode(&dependentTask)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, fmt.Errorf("dependency task not found: %s", dependencyTaskID)
			}
			return nil, fmt.Errorf("error fetching dependency task: %w", err)
		}

		// Ako je neki zavisni zadatak "pending", ne možeš promeniti status osim ako ne vraćaš u "pending"
		if dependentTask.Status == "pending" && status != "pending" {
			return nil, fmt.Errorf("cannot change status: dependency task %s is pending", dependentTask.ID.Hex())
		}

		// Ako je neki zavisni zadatak u "work in progress", možeš preći samo u "work in progress" ili "pending"
		if dependentTask.Status == "work in progress" {
			if status != "work in progress" && status != "pending" {
				return nil, fmt.Errorf("cannot change status: dependency task %s is in progress", dependentTask.ID.Hex())
			}
		}

		// Ako su svi zavisni zadaci u "done", možeš preći u bilo koji status
		if status == "done" && dependentTask.Status != "done" {
			return nil, fmt.Errorf("cannot change status to 'done': dependency task %s is not done", dependentTask.ID.Hex())
		}
	}

	// Ažuriranje statusa zadatka u bazi
	updateResult, err := collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": taskObjectID},
		bson.M{"$set": bson.M{"status": status}},
	)

	sendToAnalyticsService(map[string]interface{}{
		"task_id":         taskID,
		"previous_status": task.Status,
		"new_status":      status,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}, token)

	if err != nil {
		return nil, fmt.Errorf("error updating task status: %w", err)
	}
	if updateResult.MatchedCount == 0 {
		return nil, errors.New("task not found during update")
	}

	// Ponovno čitanje ažuriranog zadatka
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&task)
	if err != nil {
		return nil, fmt.Errorf("error fetching updated task: %w", err)
	}

	return &task, nil
}

// userExists proverava da li korisnik sa datim userID postoji
func userExists(userID string, token string) (bool, error) {
	// Sanitize user ID to prevent XSS attacks
	userID = SanitizeInput(userID)

	// Validate userID to ensure it has acceptable characters
	if !isValidUserID(userID) {
		return false, errors.New("invalid user ID format")
	}

	// Use URL encoding for security
	encodedUserID := url.QueryEscape(userID)

	// Create the URL for the user-service request
	url := fmt.Sprintf("http://user-service:8080/users/%s/exists", encodedUserID)

	// Log for debugging
	fmt.Println("Requesting URL:", url)

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Add the Authorization header with the Bearer token
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("received non-OK response: %s", body)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse the response body into a map
	var result map[string]bool
	err = json.Unmarshal(body, &result)
	if err != nil {
		return false, fmt.Errorf("failed to parse response body: %v", err)
	}

	// Check if the "exists" field is present in the response
	exists, ok := result["exists"]
	if !ok {
		return false, fmt.Errorf("response missing 'exists' field")
	}

	return exists, nil
}

// GetTasks vraća sve zadatke iz baze podataka.
func GetTasks() ([]models.Task, error) {
	collection := db.Client.Database("testdb").Collection("tasks")
	var tasks []models.Task

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Upit za sve zadatke (bez filtera)
	cursor, err := collection.Find(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

func GetTasksByProjectID(projectID string) ([]models.Task, error) {
	collection := db.Client.Database("testdb").Collection("tasks")
	var tasks []models.Task

	projectID = SanitizeInput(projectID)

	// Validacija ObjectID formata za projekt ID
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, errors.New("invalid project ID format")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Upit za zadatke koji odgovaraju projectID-u
	options := options.Find().SetSort(bson.M{"position": 1})
	cursor, err := collection.Find(ctx, bson.M{"project_id": projectObjectID}, options)
	if err != nil {
		return nil, err
	}

	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

func SanitizeInput(input string) string {
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, `"`, "&quot;")
	input = strings.ReplaceAll(input, "'", "&#39;")
	return input
}

func CreateTask(projectID, name, description string, dependsOn []string, token string) (*models.Task, error) {
	// Sanitize inputs
	projectID = SanitizeInput(projectID)
	name = SanitizeInput(name)
	description = SanitizeInput(description)

	// Sanitize each ID in the dependsOn list
	var sanitizedDependsOn []string
	for _, dep := range dependsOn {
		sanitizedDependsOn = append(sanitizedDependsOn, SanitizeInput(dep))
	}

	// Validate projectID format
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, errors.New("invalid project ID format")
	}

	// Connect to MongoDB collection
	collection := db.Client.Database("testdb").Collection("tasks")

	// Check if a task with the same name already exists in the project
	var existingTask models.Task
	err = collection.FindOne(context.TODO(), bson.M{
		"name":       strings.ToLower(name),
		"project_id": projectObjectID.Hex(),
	}).Decode(&existingTask)

	if err != mongo.ErrNoDocuments {
		if err != nil {
			return nil, fmt.Errorf("error checking for existing task: %v", err)
		}
		return nil, errors.New("a task with the same name already exists in this project")
	}

	var lastTask models.Task
	err = collection.FindOne(context.TODO(), bson.M{"project_id": projectObjectID.Hex()}, options.FindOne().SetSort(bson.M{"position": -1})).Decode(&lastTask)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("error finding last task: %v", err)
	}

	// Dodeli position vrednost
	position := 1
	if err != mongo.ErrNoDocuments {
		position = lastTask.Position + 1
	}

	// Create ObjectID list for dependencies
	var dependsOnObjectIDs []primitive.ObjectID
	for _, dep := range sanitizedDependsOn {
		depID, err := primitive.ObjectIDFromHex(dep)
		if err != nil {
			return nil, fmt.Errorf("invalid dependsOn ID format: %v", err)
		}
		dependsOnObjectIDs = append(dependsOnObjectIDs, depID)
	}

	// Create the new task
	task := models.Task{
		ID:          primitive.NewObjectID(),
		Name:        strings.ToLower(name),
		Description: description,
		Status:      "pending",
		Users:       []string{}, // Prazna lista korisnika
		Project_ID:  projectObjectID.Hex(),
		DependsOn:   dependsOnObjectIDs,
		FilePaths:   []string{},
		Position:    position, // Dodato position polje
	}

	// Insert the new task into the database
	_, err = collection.InsertOne(context.TODO(), task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %v", err)
	}

	// Notify project-service about the new task
	payload := map[string]interface{}{
		"task_id":     task.ID.Hex(),
		"project_id":  projectObjectID.Hex(),
		"name":        name,
		"description": description,
		"depends_on":  dependsOn,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize task payload: %v", err)
	}

	// URL for calling the project-service
	projectServiceURL := fmt.Sprintf("http://project-service:8080/projects/%s/tasks/%s", projectObjectID.Hex(), task.ID.Hex())
	req, err := http.NewRequest("PUT", projectServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request to project-service: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token)) // Add the Bearer token

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to project-service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("project-service failed to update with new task details: %s", string(body))
	}

	// Return the created task
	return &task, nil
}

func AddUserToTask(taskID string, userID string, token string) error {
	// Sanitizacija unosa kako bi se sprečili XSS napadi
	taskID = SanitizeInput(taskID)
	userID = SanitizeInput(userID)

	// Provera da li korisnik postoji
	userExists, err := userExists(userID, token)
	if err != nil {
		return err
	}
	if !userExists {
		return errors.New("user not found")
	}

	// Validacija formata taskID
	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return errors.New("invalid task ID format")
	}

	collection := db.Client.Database("testdb").Collection("tasks")

	// Provera da li zadatak postoji
	var task models.Task
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return errors.New("task not found")
	} else if err != nil {
		return err
	}

	// Dodavanje korisnika u zadatak
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

func RemoveUserFromTask(taskID string, userID string, token string) error {
	taskID = SanitizeInput(taskID)
	userID = SanitizeInput(userID)

	userExists, err := userExists(userID, token)
	if err != nil {
		return err
	}
	if !userExists {
		return errors.New("user not found")
	}

	// Validacija formata taskID
	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return errors.New("invalid task ID format")
	}

	collection := db.Client.Database("testdb").Collection("tasks")

	// Provera da li zadatak postoji
	var task models.Task
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return errors.New("task not found")
	} else if err != nil {
		return err
	}

	// Provera da li je korisnik već dodeljen zadatku
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

	// Uklanjanje korisnika sa zadatka
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

func GetUsersForTask(taskID string, token string) ([]models.User, error) {
	// Sanitize input
	taskID = SanitizeInput(taskID)

	// Convert task ID to ObjectID
	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return nil, errors.New("invalid task ID format")
	}

	// Find the task in the collection
	collection := db.Client.Database("testdb").Collection("tasks")
	var task models.Task
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("task not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to fetch task: %v", err)
	}

	// If the task has no users, return an empty list
	if len(task.Users) == 0 {
		return []models.User{}, nil
	}

	// Fetch user details from user-service
	var users []models.User
	client := &http.Client{}
	for _, userID := range task.Users {
		url := fmt.Sprintf("http://user-service:8080/users/%s", userID)

		// Create a new HTTP request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for user %s: %v", userID, err)
		}

		// Set the Authorization header with the Bearer token
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		// Send the request
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user %s: %v", userID, err)
		}
		defer resp.Body.Close()

		// Check the status code
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch user %s, status: %d", userID, resp.StatusCode)
		}

		// Parse the response into the User struct
		var user models.User
		err = json.NewDecoder(resp.Body).Decode(&user)
		if err != nil {
			return nil, fmt.Errorf("failed to parse user %s: %v", userID, err)
		}

		// Add the user to the result list
		users = append(users, user)
	}

	return users, nil
}

// GetTaskByID vraća zadatak sa zadatim ID-jem
func GetTaskByID(taskID string) (*models.Task, error) {
	// Sanitizacija unosa
	taskID = SanitizeInput(taskID)

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

func IsUserInTask(taskID string, userID string, token string) (bool, error) {
	// Sanitize input
	taskID = SanitizeInput(taskID)
	userID = SanitizeInput(userID)
	fmt.Sprintf(userID)

	// Construct the URL for the user-service
	url := fmt.Sprintf("http://user-service:8080/users/%s", userID)
	fmt.Println("Requesting URL:", url) // Debug log

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %v", err)
	}

	// Set the Authorization header with the Bearer token
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to fetch user info: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is OK
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to fetch user info, status: %d", resp.StatusCode)
	}

	// Decode the response into a User object
	var user models.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return false, fmt.Errorf("failed to decode user info: %v", err)
	}

	fmt.Printf("Fetched User: %+v\n", user) // Debug log

	// Parse taskID into an ObjectID
	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return false, errors.New("invalid task ID format")
	}

	// Search for the task in the database
	collection := db.Client.Database("testdb").Collection("tasks")
	var task models.Task
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&task)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return false, errors.New("task not found")
		}
		return false, err
	}

	// Check if the userID exists in the task's user list, skipping the first user
	for i, id := range task.Users {
		if i == 0 {
			continue // Skip the first user
		}
		fmt.Printf("Checking userID: %s, user role: %s\n", userID, user.Role) // Log the userID and the role

		if id == userID || user.Role == "Manager" {
			return true, nil
		}
	}

	return false, nil
}

// FileExistsInHDFS proverava da li fajl sa datom putanjom postoji u HDFS-u
func FileExistsInHDFS(hdfsFilePath string) (bool, error) {
	// Učitaj HDFS_NAMENODE_ADDRESS iz .env fajla
	hdfsNamenodeAddress := os.Getenv("HDFS_NAMENODE_ADDRESS")
	if hdfsNamenodeAddress == "" {
		return false, fmt.Errorf("HDFS_NAMENODE_ADDRESS is not set in .env file")
	}

	// Konektovanje na HDFS namenode
	client, err := hdfs.NewClient(hdfs.ClientOptions{
		Addresses: []string{hdfsNamenodeAddress}, // Koristi promenljivu iz .env fajla
	})
	if err != nil {
		return false, fmt.Errorf("failed to connect to HDFS: %v", err)
	}
	defer client.Close()

	// Provera da li fajl postoji
	_, err = client.Stat(hdfsFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Fajl ne postoji
			return false, nil
		}
		// Došlo je do druge greške
		return false, fmt.Errorf("failed to check file existence: %v", err)
	}

	// Fajl postoji
	return true, nil
}

func AddDependencyToTask(taskIDStr, dependencyIDStr string) error {
	// Logovanje vrednosti ID-ova
	fmt.Println("Task ID:", taskIDStr)
	fmt.Println("Dependency ID:", dependencyIDStr)

	if len(dependencyIDStr) != 24 {
		return fmt.Errorf("dependency ID must be 24 characters long, but got %d characters", len(dependencyIDStr))
	}

	taskID, err := primitive.ObjectIDFromHex(taskIDStr)
	if err != nil {
		fmt.Println("Error converting taskID:", err)
		return fmt.Errorf("invalid task ID format: %v", err)
	}

	dependencyID, err := primitive.ObjectIDFromHex(dependencyIDStr)
	if err != nil {
		fmt.Println("Error converting dependencyID:", err) // Log error
		return fmt.Errorf("invalid dependency ID format: %v", err)
	}

	// Povezivanje sa MongoDB kolekcijom
	collection := db.Client.Database("testdb").Collection("tasks")

	// Pronalaženje taska u bazi
	var task models.Task
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskID}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("failed to find task: %v", err)
	}

	// Proveriti da li zavisnost već postoji
	for _, dep := range task.DependsOn {
		if dep == dependencyID {
			return fmt.Errorf("task is already dependent on this task")
		}
	}

	// Dodavanje nove zavisnosti
	task.DependsOn = append(task.DependsOn, dependencyID)

	// Ažuriranje taska u bazi sa novom zavisnošću
	update := bson.M{"$set": bson.M{"dependsOn": task.DependsOn}}
	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": taskID}, update)
	if err != nil {
		return fmt.Errorf("failed to update task with new dependency: %v", err)
	}

	return nil
}
func GetTaskIDsForProject(projectID string, token string) ([]string, error) {
	// URL for the project-service endpoint
	url := fmt.Sprintf("http://project-service:8080/projects/%s", projectID)

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set the Authorization header with the Bearer token
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Create an HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project from project-service: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch project, status: %d", resp.StatusCode)
	}

	// Define a struct to parse the response
	var project struct {
		Tasks []string `json:"tasks"`
	}

	// Parse the response JSON into the struct
	err = json.NewDecoder(resp.Body).Decode(&project)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task IDs: %v", err)
	}

	return project.Tasks, nil
}

func GetDependenciesFromWorkflowService(taskID string, token string) (*models.Workflow, error) {
	url := fmt.Sprintf("http://workflow-service:8080/workflow/%s/dependencies", taskID)

	fmt.Println(url)
	var workflow models.Workflow

	for i := 0; i < 3; i++ { // Try 3 times
		// Create a new HTTP GET request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		// Set the Authorization header with the Bearer token
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		// Create an HTTP client and send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making request to workflow-service: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("no dependencies found for task_id %s (status 404)", taskID)
		} else if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error fetching dependenciesssss: received status %v", resp.StatusCode)
		}

		if err := json.NewDecoder(resp.Body).Decode(&workflow); err != nil {
			return nil, fmt.Errorf("error decoding response: %v", err)
		}

		return &workflow, nil // Return successfully if request is successful
	}

	return nil, fmt.Errorf("failed to fetch dependencies after 3 attempts")
}

func UploadFileToHDFS(localFilePath, hdfsDirPath, fileName, token string) error {
	// Učitaj HDFS_NAMENODE_ADDRESS iz .env fajla
	hdfsNamenodeAddress := os.Getenv("HDFS_NAMENODE_ADDRESS")
	if hdfsNamenodeAddress == "" {
		return fmt.Errorf("HDFS_NAMENODE_ADDRESS is not set in .env file")
	}

	// Konektovanje na HDFS namenode
	client, err := hdfs.NewClient(hdfs.ClientOptions{
		Addresses: []string{hdfsNamenodeAddress}, // Koristi promenljivu iz .env fajla
	})
	if err != nil {
		return fmt.Errorf("failed to connect to HDFS: %v", err)
	}
	defer client.Close()

	// Proveriti da li direktorijum postoji, ako ne, kreirati ga
	_, err = client.Stat(hdfsDirPath)
	if err != nil && os.IsNotExist(err) {
		err := client.MkdirAll(hdfsDirPath, os.ModePerm) // Kreira direktorijum ako ne postoji
		if err != nil {
			return fmt.Errorf("failed to create directory on HDFS: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check if directory exists: %v", err)
	}

	// Formiranje pune putanje za fajl (direktorijum + ime fajla)
	hdfsFilePath := path.Join(hdfsDirPath, fileName)

	// Proveriti da li fajl već postoji
	_, err = client.Stat(hdfsFilePath)
	if err == nil {
		// Ako fajl postoji, obriši ga pre nego što ga ponovo postaviš
		err = client.Remove(hdfsFilePath)
		if err != nil {
			return fmt.Errorf("failed to remove existing file on HDFS: %v", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check if file exists: %v", err)
	}

	// Otvoriti lokalni fajl koji treba da bude uploadovan
	localFile, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %v", err)
	}
	defer localFile.Close()

	// Kreirati fajl u HDFS-u
	hdfsFile, err := client.Create(hdfsFilePath)
	if err != nil {
		return fmt.Errorf("failed to create file on HDFS: %v", err)
	}
	defer hdfsFile.Close()

	// Kopirati sadržaj sa lokalnog fajla na HDFS
	_, err = io.Copy(hdfsFile, localFile)
	if err != nil {
		return fmt.Errorf("failed to copy data to HDFS: %v", err)
	}

	return nil
}

func ReadFileFromHDFS(hdfsPath string, token string) ([]byte, error) {
	// Učitaj HDFS_NAMENODE_ADDRESS iz .env fajla
	hdfsNamenodeAddress := os.Getenv("HDFS_NAMENODE_ADDRESS")
	if hdfsNamenodeAddress == "" {
		return nil, fmt.Errorf("HDFS_NAMENODE_ADDRESS is not set in .env file")
	}

	// Konektovanje na HDFS namenode
	client, err := hdfs.NewClient(hdfs.ClientOptions{
		Addresses: []string{hdfsNamenodeAddress}, // Koristi promenljivu iz .env fajla
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to HDFS: %v", err)
	}
	defer client.Close()

	// Otvoriti fajl sa HDFS-a
	hdfsFile, err := client.Open(hdfsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file on HDFS: %v", err)
	}
	defer hdfsFile.Close()

	// Čitanje sadržaja fajla u memoriju
	fileContent, err := ioutil.ReadAll(hdfsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file from HDFS: %v", err)
	}

	return fileContent, nil
}
func ReadFilesFromHDFSDirectory(dirPath string, token string) ([]string, error) {
	// Učitaj HDFS_NAMENODE_ADDRESS iz .env fajla
	hdfsNamenodeAddress := os.Getenv("HDFS_NAMENODE_ADDRESS")
	if hdfsNamenodeAddress == "" {
		return nil, fmt.Errorf("HDFS_NAMENODE_ADDRESS is not set in .env file")
	}

	// Konektovanje na HDFS namenode
	client, err := hdfs.NewClient(hdfs.ClientOptions{
		Addresses: []string{hdfsNamenodeAddress}, // Koristi promenljivu iz .env fajla
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to HDFS: %v", err)
	}
	defer client.Close()

	// Učitaj listu fajlova iz direktorijuma
	files, err := client.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	// Sortiraj fajlove prema numeričkim ID-ovima u imenu
	sort.Slice(fileNames, func(i, j int) bool {
		iID := extractNumericID(fileNames[i])
		jID := extractNumericID(fileNames[j])
		return iID < jID
	})

	return fileNames, nil
}

// Pomocna funkcija za ekstrakciju numeričkog ID-a iz imena fajla
func extractNumericID(fileName string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(fileName)
	if match == "" {
		return 0
	}
	id, _ := strconv.Atoi(match)
	return id
}

func TaskExists(taskID string) (bool, error) {
	// Validacija i sanitizacija ulaza
	taskID = SanitizeInput(taskID)
	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return false, errors.New("invalid task ID format")
	}

	// Povezivanje sa MongoDB kolekcijom
	collection := db.Client.Database("testdb").Collection("tasks")

	// Provera da li zadatak postoji
	var existingTask models.Task
	err = collection.FindOne(context.TODO(), bson.M{"_id": taskObjectID}).Decode(&existingTask)

	// Ako je greška `mongo.ErrNoDocuments`, zadatak ne postoji
	if err == mongo.ErrNoDocuments {
		return false, nil
	}

	// Ako postoji druga greška, prijavi je
	if err != nil {
		return false, fmt.Errorf("error checking task existence: %v", err)
	}

	// Ako nema greške, zadatak postoji
	return true, nil
}

func sendToAnalyticsService(payload map[string]interface{}, token string) {
	url := "http://analytics-service:8080/analytics/status-change"

	// Konvertuj podatke u JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal analytics payload: %v", err)
		return
	}

	// Kreiraj HTTP POST zahtev
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Failed to create analytics request: %v", err)
		return
	}

	// Postavi Authorization header sa Bearer tokenom
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	// Kreiraj HTTP klijent i pošalji zahtev
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send analytics data: %v", err)
		return
	}
	defer resp.Body.Close()

	// Proveri statusni kod odgovora
	if resp.StatusCode != http.StatusOK {
		log.Printf("Analytics service returned non-OK status: %v", resp.StatusCode)
	}
}

func DeleteTaskByID(taskID string, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Poveži se na kolekciju "tasks" u MongoDB-u
	collection := db.Client.Database("testdb").Collection("tasks")

	// Pokušaj da parsiraš taskID kao ObjectID
	objID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return fmt.Errorf("invalid taskID format: %v", err)
	}

	// Log za taskID
	fmt.Println("Attempting to delete task with ObjectID:", objID)

	// Dohvati task iz baze kako bismo dobili task podataka
	var task models.Task
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&task)
	if err != nil {
		return fmt.Errorf("could not find task with ID %v: %v", taskID, err)
	}

	// 1. Pozovi brisanje workflow-a vezanog za ovaj task
	err = deleteWorkflow(taskID, token)
	if err != nil {
		return fmt.Errorf("failed to delete workflow for task with ID %s: %v", taskID, err)
	}

	// 2. Obriši task iz MongoDB-a
	filter := bson.M{"_id": objID}
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete task: %v", err)
	}

	// Proveri rezultat
	if result.DeletedCount == 0 {
		return fmt.Errorf("no task found with taskID: %s", taskID)
	}

	// Uspešno obrisano
	fmt.Println("Task deleted successfully")
	return nil
}

// deleteWorkflow šalje HTTP DELETE zahtev za brisanje workflow-a na workflow-service
func deleteWorkflow(taskID string, token string) error {
	// URL za GET zahtev da proverimo da li workflow postoji za ovaj task
	url := fmt.Sprintf("http://workflow-service:8080/check/%s", taskID)

	// Napravimo GET zahtev da proverimo da li workflow postoji
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Dodaj token u Authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Setuj HTTP klijent sa timeout-om
	client := &http.Client{Timeout: 10 * time.Second}

	// Pošaljemo zahtev
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Ako je status kod 404, znači da workflow za taj task ne postoji
	if resp.StatusCode == http.StatusNotFound {
		// Ako workflow ne postoji, ne radimo ništa i vraćamo nil (nema greške)
		fmt.Printf("No workflow found for task %s, skipping workflow deletion\n", taskID)
		return nil
	}

	// Ako je workflow pronađen, sada možemo da pošaljemo DELETE zahtev za brisanje workflow-a
	deleteUrl := fmt.Sprintf("http://workflow-service:8080/delete/%s", taskID)
	reqDelete, err := http.NewRequest("DELETE", deleteUrl, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %v", err)
	}

	// Dodaj token u Authorization header za DELETE zahtev
	reqDelete.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Pošaljemo DELETE zahtev za brisanje workflow-a
	respDelete, err := client.Do(reqDelete)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %v", err)
	}
	defer respDelete.Body.Close()

	// Proveri status kod odgovora DELETE zahteva
	if respDelete.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete workflow, received status: %s", respDelete.Status)
	}

	// Workflow uspešno obrisan
	fmt.Printf("Workflow for task %s deleted successfully\n", taskID)
	return nil
}
func UpdateTaskPosition(taskID string, position int, token string) error {
	// Validate taskID format
	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return errors.New("invalid task ID format")
	}

	// Connect to the MongoDB collection
	collection := db.Client.Database("testdb").Collection("tasks")

	// Update the task position
	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": taskObjectID},
		bson.M{"$set": bson.M{"position": position}},
	)
	if err != nil {
		return fmt.Errorf("failed to update task position: %v", err)
	}

	return nil
}
