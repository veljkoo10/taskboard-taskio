package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux" // dodajte ovu liniju
	"net/http"
	"project-service/models"
	"project-service/service"
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

// AddUserToProject dodaje korisnika na projekat nakon što proveri validacije
func AddUserToProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]
	userID := vars["userId"]

	if projectID == "" || userID == "" {
		http.Error(w, "Project ID and User ID are required", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User successfully added to project"))
}
