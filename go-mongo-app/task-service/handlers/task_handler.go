package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/nats-io/nats.go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"task-service/db"
	"task-service/service"

	"github.com/gorilla/mux"
)

type KeyAccount struct{}

type KeyRole struct{}

type TasksHandler struct {
	logger   *log.Logger
	repo     *db.TaskRepo
	natsConn *nats.Conn
}

func NewTasksHandler(l *log.Logger, r *db.TaskRepo, natsConn *nats.Conn) *TasksHandler {
	return &TasksHandler{l, r, natsConn}
}
func (t *TasksHandler) UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID, ok := vars["taskId"]
	if !ok {
		http.Error(w, "Task ID is required in URL", http.StatusBadRequest)
		return
	}

	var requestBody struct {
		Status string `json:"status"`
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Pronađi zadatak
	task, err := service.GetTaskByID(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Provera za status "Work in Progress"
	if requestBody.Status == "work in progress" {
		// Proveri da li su svi zavisni zadaci u statusu "Work in Progress"
		for _, dependencyID := range task.DependsOn {
			dependencyIDStr := dependencyID.Hex() // Konvertuj ObjectID u string
			dependency, err := service.GetTaskByID(dependencyIDStr)
			if err != nil {
				http.Error(w, "Dependency not found", http.StatusInternalServerError)
				return
			}

			if dependency.Status != "work in progress" {
				msg := fmt.Sprintf("Cannot mark task as 'Work in Progress'. Dependency '%s' is not in 'work in progress' state.", dependency.Name)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
		}
	}

	// Provera za status "Done" i zavisnosti
	if requestBody.Status == "done" {
		// Proveri da li su svi zavisni zadaci u statusu "Done"
		for _, dependencyID := range task.DependsOn {
			dependencyIDStr := dependencyID.Hex() // Konvertuj ObjectID u string
			dependency, err := service.GetTaskByID(dependencyIDStr)
			if err != nil {
				http.Error(w, "Dependency not found", http.StatusInternalServerError)
				return
			}

			// Ako je zavisni zadatak u statusu "Work in Progress", onda može preći u "Done"
			if dependency.Status != "done" {
				msg := fmt.Sprintf("Cannot mark task as 'Done'. Dependency '%s' is not in 'done' state.", dependency.Name)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
		}
	}

	// Ako zadatak nema zavisnosti ili je uslov za status ispunjen, nastavi sa ažuriranjem statusa
	updatedTask, err := service.UpdateTaskStatus(taskID, requestBody.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nc, err := Conn()
	if err != nil {
		log.Println("Error connecting to NATS:", err)
		http.Error(w, "Failed to connect to message broker", http.StatusInternalServerError)
		return
	}
	defer nc.Close()
	message := struct {
		TaskName   string   `json:"taskName"`
		TaskStatus string   `json:"taskStatus"`
		MemberIds  []string `json:"memberIds"`
	}{
		TaskName:   task.Name,
		TaskStatus: requestBody.Status,
		MemberIds:  task.Users,
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return
	}

	subject := "task.status.update"
	err = nc.Publish(subject, jsonMessage)
	if err != nil {
		log.Println("Error publishing message to NATS:", err)
	}

	t.logger.Println("Task status update message sent to NATS")

	// Vraćanje odgovora sa ažuriranim zadatkom
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedTask); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
func (uh *TasksHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := service.GetTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (uh *TasksHandler) CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Parse project_id from URL using mux variables
	vars := mux.Vars(r)
	projectID, ok := vars["project_id"]
	if !ok {
		http.Error(w, "Project ID is required in URL", http.StatusBadRequest)
		return
	}

	// Parse the request body to get name, description, and dependsOn
	var taskInput struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		DependsOn   []string `json:"dependsOn"` // List of task IDs this task depends on
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&taskInput); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Transform the task name to lowercase
	taskInput.Name = strings.ToLower(taskInput.Name)
	// Sanitizacija unosa

	// Fetch all tasks for the given project to check for duplicate names
	tasks, err := service.GetTasksByProjectID(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if a task with the same name already exists in the project
	for _, task := range tasks {
		if task.Name == taskInput.Name {
			http.Error(w, "A task with this name already exists in the project", http.StatusConflict)
			return
		}
	}

	// Create the task using the service, passing projectID, name, description, and dependsOn
	task, err := service.CreateTask(projectID, taskInput.Name, taskInput.Description, taskInput.DependsOn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send the created task as JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// AddUserToTaskHandler handles adding a user to a task.
func (t *TasksHandler) AddUserToTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]
	userID := vars["userId"]

	err := service.AddUserToTask(taskID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	task, err := service.GetTaskByID(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	nc, err := Conn()
	if err != nil {
		log.Println("Error connecting to NATS:", err)
		http.Error(w, "Failed to connect to message broker", http.StatusInternalServerError)
		return
	}
	defer nc.Close()

	subject := "task.joined"

	message := struct {
		UserID   string `json:"userId"`
		TaskName string `json:"taskName"`
	}{
		UserID:   userID,
		TaskName: task.Name,
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return
	}

	err = nc.Publish(subject, jsonMessage)
	if err != nil {
		log.Println("Error publishing message to NATS:", err)
	}

	t.logger.Println("a message has been sent")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User added to task successfully"})
}
func Conn() (*nats.Conn, error) {
	conn, err := nats.Connect("nats://nats:4222")
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return conn, nil
}

// RemoveUserFromTaskHandler handles removing a user from a task.
func (t *TasksHandler) RemoveUserFromTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]
	userID := vars["userId"]

	err := service.RemoveUserFromTask(taskID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nc, err := Conn()
	if err != nil {
		log.Println("Error connecting to NATS:", err)
		http.Error(w, "Failed to connect to message broker", http.StatusInternalServerError)
		return
	}
	defer nc.Close()

	task, err := service.GetTaskByID(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	subject := "task.removed"

	message := struct {
		UserID   string `json:"userId"`
		TaskName string `json:"taskName"`
	}{
		UserID:   userID,
		TaskName: task.Name,
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return
	}

	err = nc.Publish(subject, jsonMessage)
	if err != nil {
		log.Println("Error publishing message to NATS:", err)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User removed from task successfully"})
}

// GetUsersForTaskHandler handles retrieving users for a specific task.
func (uh *TasksHandler) GetUsersForTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskID"]

	users, err := service.GetUsersForTask(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(users)
}

func (uh *TasksHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	task, err := service.GetTaskByID(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (uh *TasksHandler) CheckUserInTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Izvlačenje taskId i userId iz URL-a
	vars := mux.Vars(r)
	taskID, ok := vars["taskId"]
	if !ok {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	userID, ok := vars["userId"]
	if !ok {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Pozivanje servisne funkcije
	isMember, err := service.IsUserInTask(taskID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Slanje odgovora kao JSON
	response := map[string]bool{"isMember": isMember}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
func (uh *TasksHandler) AddDependencyHandler(w http.ResponseWriter, r *http.Request) {
	// Parsiranje URL parametara
	vars := mux.Vars(r)
	taskID := vars["task_id"]
	dependencyID := vars["dependency_id"]

	// Logovanje primljenih vrednosti
	fmt.Println("Received Task ID:", taskID)
	fmt.Println("Received Dependency ID:", dependencyID)

	// Pozivanje funkcije za dodavanje zavisnosti
	err := service.AddDependencyToTask(taskID, dependencyID)
	if err != nil {
		fmt.Println("Error adding dependency:", err) // Dodaj log za grešku
		http.Error(w, fmt.Sprintf("Error adding dependency: %v", err), http.StatusInternalServerError)
		return
	}

	// Vraćanje uspešnog odgovora
	response := map[string]string{
		"message":       "Dependency added successfully",
		"task_id":       taskID,
		"dependency_id": dependencyID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
func (uh *TasksHandler) GetTasksForProjectHandler(w http.ResponseWriter, r *http.Request) {
	// Parsiranje URL parametra (projectID)
	vars := mux.Vars(r)
	projectID := vars["project_id"]

	// Pozivanje funkcije koja vraća ID-eve taskova za projekat
	taskIDs, err := service.GetTaskIDsForProject(projectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching task IDs: %v", err), http.StatusInternalServerError)
		return
	}

	// Vraćanje uspešnog odgovora sa listom ID-eva taskova
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(taskIDs)
}
func (uh *TasksHandler) MiddlewareExtractUserFromHeader(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(rw http.ResponseWriter, h *http.Request) {
		// Retrieve the token from the Authorization header
		authHeader := h.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(rw, "No Authorization header found", http.StatusUnauthorized)
			uh.logger.Println("No Authorization header:", authHeader)
			return
		}

		// Expect the format "Bearer <token>"
		tokenString := ""
		if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
			tokenString = authHeader[7:]
		} else {
			http.Error(rw, "Invalid Authorization header format", http.StatusUnauthorized)
			uh.logger.Println("Invalid Authorization header format:", authHeader)
			return
		}

		// Extract userID and role from the token directly
		userID, role, err := uh.extractUserAndRoleFromToken(tokenString)
		if err != nil {
			uh.logger.Println("Token extraction failed:", err)
			http.Error(rw, `{"message": "Invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Log the userID and role
		uh.logger.Println("User ID is:", userID, "Role is:", role)

		// Add userID and role to the request context
		ctx := context.WithValue(h.Context(), KeyAccount{}, userID)
		ctx = context.WithValue(ctx, KeyRole{}, role)

		// Update the request with the new context
		h = h.WithContext(ctx)

		// Pass the request along the middleware chain
		next(rw, h)
	}
}

// Helper method to extract userID and role from JWT token
func (uh *TasksHandler) extractUserAndRoleFromToken(tokenString string) (userID string, role string, err error) {
	// Parse the token
	// Replace with your actual secret key
	secretKey := []byte(os.Getenv("TOKEN_SECRET"))

	// Parse and validate the token
	parsedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		// Validate the algorithm (ensure it's signed with HMAC)
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil || !parsedToken.Valid {
		return "", "", fmt.Errorf("invalid token: %v", err)
	}

	// Extract claims from the token
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", fmt.Errorf("invalid token claims")
	}

	// Extract userID and role from the claims
	userID, ok = claims["id"].(string)
	if !ok {
		return "", "", fmt.Errorf("userID not found in token")
	}

	role, ok = claims["role"].(string)
	if !ok {
		return "", "", fmt.Errorf("role not found in token")
	}

	return userID, role, nil
}

func (uh *TasksHandler) RoleRequired(next http.HandlerFunc, roles ...string) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) { // changed 'r' to 'req'
		// Extract the role from the request context
		role, ok := req.Context().Value(KeyRole{}).(string) // 'req' instead of 'r'
		if !ok {
			http.Error(rw, "Role not found in context", http.StatusForbidden)
			return
		}

		// Check if the user's role is in the list of required roles
		for _, r := range roles {
			if role == r {
				// If the role matches, pass the request to the next handler in the chain
				next(rw, req) // 'req' instead of 'r'
				return
			}
		}

		// If the role doesn't match any of the required roles, return a forbidden error
		http.Error(rw, "Forbidden", http.StatusForbidden)
	}
}
func (uh *TasksHandler) GetDependenciesForTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Izvlačenje `task_id` iz URL parametra
	taskID := mux.Vars(r)["task_id"]

	if taskID == "" {
		http.Error(w, "Missing task_id", http.StatusBadRequest)
		return
	}

	// Poziv funkcije za dobavljanje zavisnosti iz workflow-service
	dependencies, err := service.GetDependenciesFromWorkflowService(taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching dependencies: %v", err), http.StatusInternalServerError)
		return
	}

	// Slanje zavisnosti kao odgovor
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dependencies)
}
func (uh *TasksHandler) UpdateTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Parsiranje parametara iz URL-a
	vars := mux.Vars(r)
	taskID := vars["taskID"]

	// Parsiranje statusa iz zahteva
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validacija unosa
	if payload.Status == "" {
		http.Error(w, "Status is required", http.StatusBadRequest)
		return
	}

	// Ažuriranje statusa zadatka
	updatedTask, err := service.UpdateTaskStatus(taskID, payload.Status)
	if err != nil {
		if strings.Contains(err.Error(), "dependency task") {
			http.Error(w, err.Error(), http.StatusConflict) // Konflikt zbog zavisnosti
		} else if strings.Contains(err.Error(), "task not found") {
			http.Error(w, err.Error(), http.StatusNotFound) // Zadatak nije pronađen
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError) // Ostale greške
		}
		return
	}

	// Slanje odgovora sa ažuriranim zadatkom
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedTask)
}
func (uh *TasksHandler) UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	// Definiši maksimalnu veličinu fajla (npr. 10MB)
	const MaxFileSize = 5 * 1024 * 1024 // 10MB

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(MaxFileSize)
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			http.Error(w, "File is too large. Maximum size is 10MB.", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	taskID := r.FormValue("taskID")
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	exists, err := service.TaskExists(taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error checking task existence: %v", err), http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "Task does not exist", http.StatusNotFound)
		return
	}

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	hdfsDirPath := fmt.Sprintf("/user/hdfs/tasks/%s", taskID)
	var filePaths []string

	for _, fileHeader := range files {
		if err := validateFileSize(fileHeader, MaxFileSize); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusRequestEntityTooLarge)
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to open file: %v", err), http.StatusBadRequest)
			return
		}
		defer file.Close()

		localFilePath := "/tmp/" + fileHeader.Filename
		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
			return
		}

		err = ioutil.WriteFile(localFilePath, fileBytes, 0644)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save file locally: %v", err), http.StatusInternalServerError)
			return
		}

		err = service.UploadFileToHDFS(localFilePath, hdfsDirPath, fileHeader.Filename)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to upload file to HDFS: %v", err), http.StatusInternalServerError)
			return
		}

		hdfsFilePath := path.Join(hdfsDirPath, fileHeader.Filename)
		filePaths = append(filePaths, hdfsFilePath)
	}

	objectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid task ID: %v", err), http.StatusBadRequest)
		return
	}

	collection := db.Client.Database("testdb").Collection("tasks")
	_, err = collection.UpdateOne(
		r.Context(),
		bson.M{"_id": objectID},
		bson.M{"$push": bson.M{"filePaths": bson.M{"$each": filePaths}}},
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update task in MongoDB: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"message": "Files uploaded and task updated successfully",
	}
	json.NewEncoder(w).Encode(response)
}

// validateFileSize proverava da li je fajl prevelik.
func validateFileSize(fileHeader *multipart.FileHeader, maxSize int64) error {
	if fileHeader.Size > maxSize {
		return fmt.Errorf("File %s is too large. Maximum size is 1MB.", fileHeader.Filename)
	}
	return nil
}

func (uh *TasksHandler) DownloadFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	taskID := vars["taskID"]
	fileName := vars["fileName"]

	if fileName == "" {
		http.Error(w, "File name is required", http.StatusBadRequest)
		return
	}

	// Dekodiraj ime fajla
	decodedFileName, err := url.QueryUnescape(fileName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode file name: %v", err), http.StatusBadRequest)
		return
	}

	fmt.Printf("TaskID: %s, Original FileName: %s, Decoded FileName: %s\n", taskID, fileName, decodedFileName)

	// Proveri validnost taskID-a
	objectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid task ID: %v", err), http.StatusBadRequest)
		return
	}

	// Dohvati task iz baze
	collection := db.Client.Database("testdb").Collection("tasks")
	var task struct {
		FilePaths []string `bson:"filePaths"`
	}
	err = collection.FindOne(r.Context(), bson.M{"_id": objectID}).Decode(&task)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Nađi fajl
	var filePath string
	for _, path := range task.FilePaths {
		if filepath.Base(path) == decodedFileName {
			filePath = path
			break
		}
	}
	if filePath == "" {
		http.Error(w, fmt.Sprintf("File %s not found for task", decodedFileName), http.StatusNotFound)
		return
	}

	// Čitaj sadržaj fajla sa HDFS-a
	fileContent, err := service.ReadFileFromHDFS(filePath)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Postavi Content-Type i header
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(filePath)))

	_, err = w.Write(fileContent)
	if err != nil {
		http.Error(w, "Failed to send file", http.StatusInternalServerError)
	}
}
func (uh *TasksHandler) GetTaskFilesHandler(w http.ResponseWriter, r *http.Request) {
	// Čitanje taskID iz URL-a
	vars := mux.Vars(r)
	taskID, ok := vars["taskID"]
	if !ok || taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	// Putanja direktorijuma na HDFS-u za dati task
	dirPath := fmt.Sprintf("/user/hdfs/tasks/%s", taskID)

	files, err := service.ReadFilesFromHDFSDirectory(dirPath)
	if err != nil {
		if err.Error() == "failed to read directory: readdir /user/hdfs/tasks/"+taskID+": file does not exist" {
			files = []string{}
		} else {
			http.Error(w, fmt.Sprintf("Failed to read files from HDFS: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Postavljanje zaglavlja odgovora na JSON
	w.Header().Set("Content-Type", "application/json")

	// Slanje liste fajlova kao JSON u odgovoru
	if err := json.NewEncoder(w).Encode(files); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode files list: %v", err), http.StatusInternalServerError)
		return
	}
}
func (uh *TasksHandler) TaskExistsHandler(w http.ResponseWriter, r *http.Request) {
	// Ekstrakcija taskID iz tela zahteva
	var body struct {
		TaskID string `json:"task_id"`
	}

	// Dekodiramo JSON telo
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Pozivamo TaskExists funkciju da proverimo postojanje zadatka
	exists, err := service.TaskExists(body.TaskID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Vraćanje odgovora
	w.Header().Set("Content-Type", "application/json")
	if exists {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"exists": true})
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]bool{"exists": false})
	}
}

// DeleteTaskByIDHandler je HTTP handler za brisanje taska na osnovu taskID-a iz URL-a.
func (uh *TasksHandler) DeleteTaskByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Dohvati taskID iz URL parametara
	vars := mux.Vars(r)
	taskID, ok := vars["taskID"]
	if !ok || taskID == "" {
		http.Error(w, "taskID is required in the URL", http.StatusBadRequest)
		return
	}

	// Pozovi repository za brisanje taska po taskID-u
	err := service.DeleteTaskByID(taskID)
	if err != nil {
		http.Error(w, "Failed to delete task: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Uspešan odgovor
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Task deleted successfully"}`))
}
