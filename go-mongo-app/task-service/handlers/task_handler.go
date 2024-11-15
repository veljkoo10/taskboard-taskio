package handlers

import (
	"encoding/json"
	"net/http"
	"task-service/service"

	"github.com/gorilla/mux"
)

func UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
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

	updatedTask, err := service.UpdateTaskStatus(taskID, requestBody.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	// Parse the request body to get name and description
	var taskInput struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&taskInput); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Fetch all tasks for the given project to check for duplicate names
	tasks, err := service.GetTasksByProjectID(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if a task with the same name already exists
	for _, task := range tasks {
		if task.Name == taskInput.Name {
			http.Error(w, "A task with this name already exists in the project", http.StatusConflict)
			return
		}
	}

	// Create the task using the service, passing projectID, name, and description
	task, err := service.CreateTask(projectID, taskInput.Name, taskInput.Description)
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

func AddUserToTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]
	userID := vars["userId"]

	if err := service.AddUserToTask(taskID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Send a JSON response instead of plain text
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User successfully added to task",
	})
}

func RemoveUserFromTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]
	userID := vars["userId"]

	if err := service.RemoveUserFromTask(taskID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User successfully removed from task"))
}
