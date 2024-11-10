package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"task-service/service"
)

func GetTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := service.GetTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// CreateTaskHandler je HTTP handler koji obrađuje POST zahteve za kreiranje novog Task-a.
func CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Parsiraj project_id iz URL-a koristeći mux varijable
	vars := mux.Vars(r)
	projectID, ok := vars["project_id"]
	if !ok {
		http.Error(w, "Project ID is required in URL", http.StatusBadRequest)
		return
	}

	// Kreiraj task koristeći servis
	task, err := service.CreateTask(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Pošalji kreirani task kao JSON odgovor
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

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User successfully added to task"))
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
