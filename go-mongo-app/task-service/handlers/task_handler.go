package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"log"
	"net/http"
	"strings"
	"task-service/db"
	"task-service/service"

	"github.com/gorilla/mux"
)

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
func GetTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := service.GetTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
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
func GetUsersForTaskHandler(w http.ResponseWriter, r *http.Request) {
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

func GetTaskByID(w http.ResponseWriter, r *http.Request) {
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

func CheckUserInTaskHandler(w http.ResponseWriter, r *http.Request) {
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
func AddDependencyHandler(w http.ResponseWriter, r *http.Request) {
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
func GetTasksForProjectHandler(w http.ResponseWriter, r *http.Request) {
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
