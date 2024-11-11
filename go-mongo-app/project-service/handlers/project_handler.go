package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"project-service/models"
	"project-service/service"

	"github.com/gorilla/mux" // dodajte ovu liniju
)

func GetProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := service.GetAllProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func CreateProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	managerID := vars["managerId"]

	var project models.Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if managerID == "" {
		http.Error(w, "Manager ID is required", http.StatusBadRequest)
		return
	}

	project.ManagerID = managerID
	project.Users = append(project.Users, managerID)

	createdProject, err := service.CreateProject(project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdProject)
}

func AddUserToProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]
	userID := vars["userId"]

	if err := service.AddUserToProject(projectID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User successfully added to project"))
}

func GetProjectByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	project, err := service.GetProjectByID(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

func RemoveUserFromProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]
	userID := vars["userId"]

	if err := service.RemoveUserFromProject(projectID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User successfully removed from project"))
}
func HandleCheckProjectByTitle(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		Title string `json:"title"`
	}

	// Decode the request body
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if project exists by title
	exists, err := service.GetProjectByTitle(requestBody.Title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if exists {
		w.WriteHeader(http.StatusOK) // OK response
		w.Write([]byte("Project exists"))
	} else {
		w.WriteHeader(http.StatusOK) // Not Found response
		w.Write([]byte("Project not found"))
	}
}

// AddTaskToProjectHandler - Handler za dodavanje task-a u projekat
func AddTaskToProjectHandler(w http.ResponseWriter, r *http.Request) {
	// Preuzimanje projectID i taskID iz URL parametara
	vars := mux.Vars(r) // Koristi mux.Vars() za preuzimanje parametara iz URL-a
	projectID := vars["projectID"]
	taskID := vars["taskID"]

	// Ažuriranje projekta sa novim task-om
	err := service.AddTaskToProject(projectID, taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add task to project: %v", err), http.StatusInternalServerError)
		return
	}

	// Uspešan odgovor
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Task successfully added to project"))
}
