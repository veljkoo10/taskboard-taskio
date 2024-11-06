package handlers

import (
	"encoding/json"
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
	var project models.Project
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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
		w.WriteHeader(http.StatusNotFound) // Not Found response
		w.Write([]byte("Project not found"))
	}
}
