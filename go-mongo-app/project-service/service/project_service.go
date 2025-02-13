package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"project-service/db"
	"project-service/models"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProjectService struct {
	user   *db.Mongo
	logger *log.Logger
}

func NewProjectService(user *db.Mongo, logger *log.Logger) *ProjectService {
	return &ProjectService{user, logger}
}
func GetUserDetails(userIDs []string) ([]models.User, error) {
	var users []models.User

	for _, userID := range userIDs {
		url := fmt.Sprintf("http://user-service:8080/users/%s", userID)
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user details for %s: %v", userID, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("received non-OK response for user %s: %s", userID, resp.Status)
		}

		var user models.User
		err = json.NewDecoder(resp.Body).Decode(&user)
		if err != nil {
			return nil, fmt.Errorf("failed to decode user data for %s: %v", userID, err)
		}

		users = append(users, user)
	}

	return users, nil
}

func GetUsersForProject(projectID string) ([]string, error) {
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, errors.New("invalid project ID")
	}

	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project
	err = collection.FindOne(context.TODO(), bson.M{"_id": projectObjectID}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("project not found")
	} else if err != nil {
		return nil, err
	}

	return project.Users, nil
}
func GetProjectIDByTitle(title string) (string, error) {
	// Sanitizacija unosa
	title = sanitizeInput(title)

	// Validacija unosa
	if !isValidRegexInput(title) {
		return "", errors.New("invalid title format")
	}

	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project

	filter := bson.M{"title": bson.M{"$regex": primitive.Regex{Pattern: "^" + title + "$", Options: "i"}}}
	err := collection.FindOne(context.TODO(), filter).Decode(&project)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", nil
		}
		return "", err
	}

	return project.ID.Hex(), nil
}

func GetProjectsByUserID(userID string) ([]models.Project, error) {
	collection := db.Client.Database("testdb").Collection("projects")
	var projects []models.Project

	filter := bson.M{"users": userID}
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if err = cursor.All(context.TODO(), &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

func userExists(userID string) (bool, error) {
	url := fmt.Sprintf("http://user-service:8080/users/%s/exists", userID)
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

func GetAllProjects() ([]models.Project, error) {
	collection := db.Client.Database("testdb").Collection("projects")
	var projects []models.Project

	cursor, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if err = cursor.All(context.TODO(), &projects); err != nil {
		return nil, err
	}

	return projects, nil
}
func GetProjectByTitleAndManager(title string, managerID string) (bool, error) {
	title = sanitizeInput(title)

	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project

	normalizedTitle := strings.ToLower(title)

	filter := bson.M{
		"title":     normalizedTitle,
		"managerId": managerID,
	}

	err := collection.FindOne(context.TODO(), filter).Decode(&project)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
func CreateProject(project models.Project) (string, error) {
	// Sanitizacija i normalizacija
	project.Title = sanitizeInput(strings.ToLower(project.Title))
	project.Description = sanitizeInput(project.Description)

	// Validacija ulaza
	if !isValidTitle(project.Title) {
		return "", errors.New("invalid project title format")
	}
	if len(project.Title) > 100 {
		return "", errors.New("title exceeds maximum length of 100 characters")
	}
	if len(project.Description) > 1000 {
		return "", errors.New("description exceeds maximum length of 1000 characters")
	}

	// Validacija datuma
	expectedEndDate, err := time.Parse("2006-01-02", project.ExpectedEndDate)
	if err != nil {
		return "", errors.New("invalid expected end date format, must be YYYY-MM-DD")
	}

	// Provera dodatnih pravila
	if err := validateProject(project, expectedEndDate); err != nil {
		return "", err
	}

	// Spremanje u bazu sa sanitizovanim podacima
	collection := db.Client.Database("testdb").Collection("projects")
	safeProject := bson.M{
		"title":             project.Title,
		"description":       project.Description,
		"expected_end_date": project.ExpectedEndDate,
		"manager_id":        project.ManagerID,
		"users":             project.Users,
		"min_people":        project.MinPeople,
		"max_people":        project.MaxPeople,
		"createdAt":         time.Now(),
	}

	result, err := collection.InsertOne(context.TODO(), safeProject)
	if err != nil {
		return "", err
	}

	// Vraćanje generisanog ID-a
	projectID := result.InsertedID.(primitive.ObjectID).Hex()
	return projectID, nil
}

func validateProject(project models.Project, expectedEndDate time.Time) error {
	if project.MinPeople < 1 || project.MaxPeople < project.MinPeople {
		return errors.New("invalid min/max people values")
	}
	if expectedEndDate.Before(time.Now()) {
		return errors.New("Expected date must be in the future")
	}
	exists, err := projectExists(project.Title, project.ManagerID)
	if err != nil {
		return err // Return any error encountered during the check
	}
	if exists {
		return errors.New("Project with this name already exists for the same manager")
	}

	return nil
}

func projectExists(title, managerID string) (bool, error) {
	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project

	filter := bson.M{
		"title":      title,
		"manager_id": managerID,
	}

	err := collection.FindOne(context.TODO(), filter).Decode(&project)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func AddUsersToProject(projectID string, userIDs []string) error {
	for _, userID := range userIDs {
		userExists, err := userExists(userID)
		if err != nil {
			return err
		}
		if !userExists {
			return fmt.Errorf("user %s not found", userID)
		}
	}

	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return errors.New("invalid project ID format")
	}

	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project
	err = collection.FindOne(context.TODO(), bson.M{"_id": projectObjectID}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return errors.New("project not found")
	} else if err != nil {
		return err
	}

	if len(project.Users)+len(userIDs) > project.MaxPeople {
		return errors.New("adding these users exceeds the max number of users for this project")
	}

	for _, userID := range userIDs {
		for _, existingUserID := range project.Users {
			if existingUserID == userID {
				return fmt.Errorf("user %s is already a member of this project", userID)
			}
		}

		_, err := collection.UpdateOne(
			context.TODO(),
			bson.M{"_id": projectObjectID},
			bson.M{"$addToSet": bson.M{"users": userID}},
		)
		if err != nil {
			return fmt.Errorf("failed to add user %s to project: %v", userID, err)
		}
	}

	return nil
}

func countProjectUsers(projectID string) (int, error) {
	collection := db.Client.Database("testdb").Collection("projects")
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return 0, errors.New("invalid project ID")
	}

	var project models.Project
	err = collection.FindOne(context.TODO(), bson.M{"_id": projectObjectID}).Decode(&project)
	if err != nil {
		return 0, err
	}

	return len(project.Users), nil
}

func GetProjectByID(projectID string) (*models.Project, error) {
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, errors.New("invalid project ID")
	}

	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project
	err = collection.FindOne(context.TODO(), bson.M{"_id": projectObjectID}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("project not found")
	} else if err != nil {
		return nil, err
	}

	return &project, nil
}

func RemoveUsersFromProject(projectID string, userIDs []string) error {
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return errors.New("invalid project ID format")
	}

	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project
	err = collection.FindOne(context.TODO(), bson.M{"_id": projectObjectID}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return errors.New("project not found")
	} else if err != nil {
		return err
	}

	// Check if all users exist in the project
	for _, userID := range userIDs {
		userFound := false
		for _, existingUserID := range project.Users {
			if existingUserID == userID {
				userFound = true
				break
			}
		}
		if !userFound {
			return fmt.Errorf("user %s is not a member of this project", userID)
		}
	}

	// Remove users from the project
	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": projectObjectID},
		bson.M{"$pull": bson.M{"users": bson.M{"$in": userIDs}}},
	)
	if err != nil {
		return fmt.Errorf("failed to remove users from project: %v", err)
	}

	return nil
}

func GetProjectByTitle(title string) (bool, error) {
	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project

	filter := bson.M{"title": bson.M{"$regex": primitive.Regex{Pattern: "^" + title + "$", Options: "i"}}}
	err := collection.FindOne(context.TODO(), filter).Decode(&project)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil // Ako projekat nije pronađen
		}
		return false, err
	}

	return true, nil
}

func AddTaskToProject(projectID string, taskID string) error {
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return errors.New("invalid project ID format")
	}

	taskObjectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return errors.New("invalid task ID format")
	}

	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project
	err = collection.FindOne(context.TODO(), bson.M{"_id": projectObjectID}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return errors.New("project not found")
	} else if err != nil {
		return err
	}

	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": projectObjectID},
		bson.M{"$addToSet": bson.M{"tasks": taskObjectID.Hex()}},
	)
	if err != nil {
		return err
	}

	return nil
}

func IsActiveProject(projectID string) (bool, error) {
	// Lista za skladištenje statusa
	var taskStatuses []string
	var pendingTasks []string
	var inProgressTasks []string
	var doneTasks []string

	// Konvertovanje projectID u ObjectID
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		fmt.Printf("Invalid project ID format: %v\n", err)
		return false, fmt.Errorf("invalid project ID: %v", err)
	}

	// Dohvatanje projekta iz baze
	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project
	err = collection.FindOne(context.TODO(), bson.M{"_id": projectObjectID}).Decode(&project)
	if err != nil {
		fmt.Printf("Failed to find project: %v\n", err)
		return false, fmt.Errorf("failed to find project: %v", err)
	}

	// Proverite da li ima taskova
	if len(project.Tasks) == 0 {
		fmt.Println("No tasks found for the project")
		return true, nil
	}

	// Iteriraj kroz sve taskove i dohvati status
	for _, taskID := range project.Tasks {
		taskStatus, err := getTaskStatus(taskID)
		if err != nil {
			fmt.Printf("Failed to fetch status for task %s: %v\n", taskID, err)
			return false, fmt.Errorf("failed to fetch status for task %s: %v", taskID, err)
		}

		// Dodaj status u odgovarajuću listu
		taskStatuses = append(taskStatuses, taskStatus)
		switch taskStatus {
		case "pending":
			pendingTasks = append(pendingTasks, taskStatus)
		case "work in progress":
			inProgressTasks = append(inProgressTasks, taskStatus)
		case "done":
			doneTasks = append(doneTasks, taskStatus)
		}
	}

	// Logika za odlučivanje
	fmt.Printf("Pending: %d, In Progress: %d, Done: %d\n", len(pendingTasks), len(inProgressTasks), len(doneTasks))
	if len(pendingTasks) == 0 && len(inProgressTasks) == 0 && len(doneTasks) != 0 {
		return false, nil
	}

	return true, nil
}

// getTaskStatus - dobija status zadatka sa task-servisa
func getTaskStatus(taskID string) (string, error) {
	url := fmt.Sprintf("http://task-service:8080/tasks/%s", taskID)

	// Šaljemo GET zahtev na task-servis
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to send request to task service: %v", err)
	}
	defer resp.Body.Close()

	// Proveri statusni kod odgovora
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error response from task service: status code %d", resp.StatusCode)
	}

	// Parsiranje odgovora
	var responseBody struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return "", fmt.Errorf("failed to decode response body: %v", err)
	}

	return responseBody.Status, nil
}

func sanitizeInput(input string) string {
	input = strings.TrimSpace(input)
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, `"`, "&quot;")
	input = strings.ReplaceAll(input, `'`, "&#39;")
	return input
}

func isValidRegexInput(input string) bool {
	// Proverite da li unos sadrži samo dozvoljene karaktere
	return regexp.MustCompile(`^[a-zA-Z0-9\s]+$`).MatchString(input)
}

func isValidTitle(title string) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9\s]+$`).MatchString(title)
}

// DeleteProjectByID briše projekat iz MongoDB-a i briše sve povezane taskove.
func DeleteProjectByID(projectID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Poveži se na kolekciju "projects" u MongoDB-u
	collection := db.Client.Database("testdb").Collection("projects")

	// Pokušaj da parsiraš projectID kao ObjectID
	objID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return fmt.Errorf("invalid projectID format: %v", err)
	}

	// Log za projectID
	fmt.Println("Attempting to delete project with ObjectID:", objID)

	// Dohvati projekat iz baze kako bismo dobili listu taskova
	var project models.Project
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&project)
	if err != nil {
		return fmt.Errorf("could not find project with ID %v: %v", projectID, err)
	}

	// 2. Pozovi brisanje svih taskova vezanih za projekat
	for _, taskID := range project.Tasks {
		// Napravi DELETE zahtev za svaki task
		err := deleteTask(taskID)
		if err != nil {
			return fmt.Errorf("failed to delete task with ID %s: %v", taskID, err)
		}
	}

	// 3. Obriši projekat iz MongoDB-a
	filter := bson.M{"_id": objID}
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete project: %v", err)
	}

	// Proveri rezultat
	if result.DeletedCount == 0 {
		return fmt.Errorf("no project found with projectID: %s", projectID)
	}

	// Uspešno obrisano
	fmt.Println("Project deleted successfully")
	return nil
}

// deleteTask šalje HTTP DELETE zahtev za brisanje taska na task-service
func deleteTask(taskID string) error {
	// URL za DELETE zahtev
	url := fmt.Sprintf("http://task-service:8080/tasks/delete/%s", taskID)

	// Napravimo DELETE zahtev
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Setuj HTTP klijent sa timeout-om
	client := &http.Client{Timeout: 10 * time.Second}

	// Pošaljemo zahtev
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Proveri status kod odgovora
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete task, received status: %s", resp.Status)
	}

	// Task uspešno obrisan
	fmt.Printf("Task %s deleted successfully\n", taskID)
	return nil
}
