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
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
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
func UpdateTaskStatus(taskID, status string) (*models.Task, error) {
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
	dependencies, err := GetDependenciesFromWorkflowService(taskID)
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

		// Ako je neki zavisni zadatak "pending", ne možeš promeniti status
		if dependentTask.Status == "pending" {
			return nil, fmt.Errorf("cannot change status: dependency task %s is pending", dependentTask.ID.Hex())
		}

		// Ako je neki zavisni zadatak u "work in progress", možeš preći samo u "work in progress"
		if dependentTask.Status == "work in progress" {
			if status != "work in progress" {
				return nil, fmt.Errorf("cannot change status: dependency task %s is in progress", dependentTask.ID.Hex())
			}
		}

		// Ako su svi zavisni zadaci u "done", možeš preći u "done"
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
func userExists(userID string) (bool, error) {
	// Sanitizacija korisničkog ID-a da bismo sprečili XSS napade
	userID = SanitizeInput(userID)

	// Validacija userID-a da ne sadrži neprihvatljive karaktere
	if !isValidUserID(userID) {
		return false, errors.New("invalid user ID format")
	}

	// Korišćenje URL encoding-a za sigurnost
	encodedUserID := url.QueryEscape(userID)

	// Kreiranje URL-a za poziv user-service
	url := fmt.Sprintf("http://user-service:8080/users/%s/exists", encodedUserID)

	// Logovanje za debugging
	fmt.Println("Requesting URL:", url)

	// Slanje HTTP GET zahteva
	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %v", err)
	}
	defer resp.Body.Close()

	// Provera statusnog koda odgovora
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("received non-OK response: %s", body)
	}

	// Čitanje tela odgovora
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %v", err)
	}

	// Parsiranje odgovora u mapu
	var result map[string]bool
	err = json.Unmarshal(body, &result)
	if err != nil {
		return false, fmt.Errorf("failed to parse response body: %v", err)
	}

	// Provera da li postoji "exists" polje u odgovoru
	exists, ok := result["exists"]
	if !ok {
		return false, fmt.Errorf("response missing 'exists' field")
	}

	// Povratak rezultata
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
	cursor, err := collection.Find(ctx, bson.M{"project_id": projectObjectID})
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

func CreateTask(projectID, name, description string, dependsOn []string) (*models.Task, error) {
	// Sanitizacija unosa
	projectID = SanitizeInput(projectID)
	name = SanitizeInput(name)
	description = SanitizeInput(description)

	// Sanitizacija svakog ID-a u dependsOn listi
	var sanitizedDependsOn []string
	for _, dep := range dependsOn {
		sanitizedDependsOn = append(sanitizedDependsOn, SanitizeInput(dep))
	}

	// Validacija formata projectID-a
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, errors.New("invalid project ID format")
	}

	// Povezivanje sa MongoDB kolekcijom
	collection := db.Client.Database("testdb").Collection("tasks")

	// Provera da li zadatak sa istim imenom već postoji u projektu
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

	// Kreiranje ObjectID liste za zavisnosti
	var dependsOnObjectIDs []primitive.ObjectID
	for _, dep := range sanitizedDependsOn {
		depID, err := primitive.ObjectIDFromHex(dep)
		if err != nil {
			return nil, fmt.Errorf("invalid dependsOn ID format: %v", err)
		}
		dependsOnObjectIDs = append(dependsOnObjectIDs, depID)
	}

	// Kreiranje novog zadatka
	task := models.Task{
		ID:          primitive.NewObjectID(),
		Name:        strings.ToLower(name),
		Description: description,
		Status:      "pending",
		Users:       []string{}, // Prazna lista korisnika
		Project_ID:  projectObjectID.Hex(),
		DependsOn:   dependsOnObjectIDs,
		FilePaths:   []string{},
	}

	// Ubacivanje novog zadatka u bazu podataka
	_, err = collection.InsertOne(context.TODO(), task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %v", err)
	}

	// Obavestiti project-service o novom zadatku
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

	// URL za pozivanje project-service-a
	projectServiceURL := fmt.Sprintf("http://project-service:8080/projects/%s/tasks/%s", projectObjectID.Hex(), task.ID.Hex())
	req, err := http.NewRequest("PUT", projectServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request to project-service: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to project-service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("project-service failed to update with new task details")
	}

	// Vraćanje kreiranog zadatka
	return &task, nil
}

func AddUserToTask(taskID string, userID string) error {
	// Sanitizacija unosa kako bi se sprečili XSS napadi
	taskID = SanitizeInput(taskID)
	userID = SanitizeInput(userID)

	// Provera da li korisnik postoji
	userExists, err := userExists(userID)
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

func RemoveUserFromTask(taskID string, userID string) error {
	taskID = SanitizeInput(taskID)
	userID = SanitizeInput(userID)

	userExists, err := userExists(userID)
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

func GetUsersForTask(taskID string) ([]models.User, error) {
	// Sanitizacija unosa
	taskID = SanitizeInput(taskID)

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

// IsUserInTask proverava da li je korisnik dodeljen zadatku
func IsUserInTask(taskID string, userID string) (bool, error) {
	// Sanitizacija unosa
	taskID = SanitizeInput(taskID)
	userID = SanitizeInput(userID)

	url := fmt.Sprintf("http://user-service:8080/users/%s", userID)
	fmt.Println("Requesting URL:", url) // Debug log

	// Pozivamo user-service
	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to get user info: %v", err)
	}
	defer resp.Body.Close()

	// Dekodiramo odgovor u objekat user
	var user models.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return false, fmt.Errorf("failed to decode user info: %v", err)
	}

	fmt.Printf("Fetched User: %+v\n", user) // Debug log

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
		if id == userID || user.Role == "Manager" {
			return true, nil
		}
	}

	return false, nil
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
func GetTaskIDsForProject(projectID string) ([]string, error) {
	// URL za pozivanje project-service-a
	url := fmt.Sprintf("http://project-service:8080/projects/%s", projectID)

	// HTTP GET zahtev za project-service
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project from project-service: %v", err)
	}
	defer resp.Body.Close()

	// Proveri statusni kod
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch project, status: %d", resp.StatusCode)
	}

	// Definiši strukturu za odgovor
	var project struct {
		Tasks []string `json:"tasks"`
	}

	// Parsiraj odgovor
	err = json.NewDecoder(resp.Body).Decode(&project)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task IDs: %v", err)
	}

	return project.Tasks, nil
}
func GetDependenciesFromWorkflowService(taskID string) (*models.Workflow, error) {
	url := fmt.Sprintf("http://workflow-service:8084/workflow/%s/dependencies", taskID)

	fmt.Println(url)
	var workflow models.Workflow

	for i := 0; i < 3; i++ { // Pokušaj 3 puta
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("error making request to workflow-service: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("no dependencies found for task_id %s (status 404)", taskID)
		} else if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error fetching dependencies: received status %v", resp.StatusCode)
		}

		if err := json.NewDecoder(resp.Body).Decode(&workflow); err != nil {
			return nil, fmt.Errorf("error decoding response: %v", err)
		}

		return &workflow, nil // Ako je uspešno, odmah vrati rezultat
	}

	return nil, fmt.Errorf("failed to fetch dependencies after 3 attempts")
}

func UploadFileToHDFS(localFilePath, hdfsDirPath, fileName string) error {
	// Konektovanje na HDFS namenode
	client, err := hdfs.NewClient(hdfs.ClientOptions{
		Addresses: []string{"namenode:8020"}, // Adresa namenode servera
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

	// Kreirati direktorijum putem MkdirAll za celu putanju
	err = client.MkdirAll(hdfsDirPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directories on HDFS: %v", err)
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

func ReadFileFromHDFS(hdfsPath string) ([]byte, error) {
	// Konektovanje na HDFS namenode
	client, err := hdfs.NewClient(hdfs.ClientOptions{
		Addresses: []string{"namenode:8020"}, // Adresa namenode servera
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
func ReadFilesFromHDFSDirectory(dirPath string) ([]string, error) {
	// Konektovanje na HDFS namenode
	client, err := hdfs.NewClient(hdfs.ClientOptions{
		Addresses: []string{"namenode:8020"}, // Adresa namenode servera
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

	return fileNames, nil
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
