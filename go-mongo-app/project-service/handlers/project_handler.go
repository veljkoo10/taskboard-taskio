package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"log"
	"net/http"
	"os"
	"project-service/db"
	"project-service/models"
	"project-service/service"
	"strings"

	"github.com/gorilla/mux"
)

type ProjectHandler struct {
	logger   *log.Logger
	repo     *db.ProjectRepo
	natsConn *nats.Conn
}

func NewProjectsHandler(l *log.Logger, r *db.ProjectRepo, natsConn *nats.Conn) *ProjectHandler {
	return &ProjectHandler{l, r, natsConn}
}
func GetUsersForProjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	usersIDs, err := service.GetUsersForProject(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	users, err := service.GetUserDetails(usersIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(users)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func GetProjectIDByTitle(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		Title string `json:"title"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	projectID, err := service.GetProjectIDByTitle(requestBody.Title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if projectID == "" {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"projectId": projectID})
}

func GetProjectsByUserID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	projects, err := service.GetProjectsByUserID(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}
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

	project.Tasks = []string{}

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

func (p *ProjectHandler) AddUsersToProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	var requestBody struct {
		UserIDs []string `json:"userIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := service.AddUsersToProject(projectID, requestBody.UserIDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	project, err := service.GetProjectByID(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nc, err := Conn()
	if err != nil {
		http.Error(w, "Failed to connect to message broker", http.StatusInternalServerError)
		p.logger.Println("Error connecting to NATS:", err)
		return
	}
	defer nc.Close()

	for _, uid := range requestBody.UserIDs {
		subject := "project.joined"

		message := struct {
			UserID      string `json:"userId"`
			ProjectName string `json:"projectName"`
		}{
			UserID:      uid,
			ProjectName: project.Title,
		}

		jsonMessage, err := json.Marshal(message)
		if err != nil {
			log.Println("Error marshalling message:", err)
			continue
		}

		err = nc.Publish(subject, jsonMessage)
		if err != nil {
			log.Println("Error publishing message to NATS:", err)
		}
	}
	p.logger.Println("a message has been sent")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Users successfully added to project"))
}

func Conn() (*nats.Conn, error) {
	connection := os.Getenv("NATS_URL")
	conn, err := nats.Connect(connection)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return conn, nil
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

func (p *ProjectHandler) RemoveUsersFromProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	var requestBody struct {
		UserIDs []string `json:"userIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Call service to remove users from the project
	if err := service.RemoveUsersFromProject(projectID, requestBody.UserIDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	project, err := service.GetProjectByID(projectID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, uid := range requestBody.UserIDs {

		subject := "project.removed"
		message := struct {
			UserID      string `json:"userId"`
			ProjectName string `json:"projectName"`
		}{
			UserID:      uid,
			ProjectName: project.Title,
		}

		if err := p.sendNotification(subject, message); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	p.logger.Println("a message has been sent")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Users successfully removed from project"))
}

func (p *ProjectHandler) sendNotification(subject string, message interface{}) error {
	nc, err := Conn()
	if err != nil {
		log.Println("Error connecting to NATS:", err)
		return fmt.Errorf("failed to connect to message broker: %w", err)
	}
	defer nc.Close()

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return fmt.Errorf("error marshalling message: %w", err)
	}

	err = nc.Publish(subject, jsonMessage)
	if err != nil {
		log.Println("Error publishing message to NATS:", err)
		return fmt.Errorf("error publishing message to NATS: %w", err)
	}

	p.logger.Println("Notification sent:", subject)
	return nil
}

func HandleCheckProjectByTitle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	managerID := vars["managerId"]

	var requestBody struct {
		Title string `json:"title"`
	}

	// Parsiranje JSON tela zahteva
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if managerID == "" {
		http.Error(w, "Manager ID is required", http.StatusBadRequest)
		return
	}

	normalizedTitle := strings.ToLower(requestBody.Title)

	fmt.Println("Original Title:", requestBody.Title)
	fmt.Println("Normalized Title:", normalizedTitle)

	// Poziv funkcije za proveru postojanja projekta u bazi
	exists, err := service.GetProjectByTitleAndManager(normalizedTitle, managerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Slanje odgovora klijentu
	if exists {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Project exists"))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Project not found"))
	}
}

func AddTaskToProjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectID"]
	taskID := vars["taskID"]

	err := service.AddTaskToProject(projectID, taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add task to project: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Task successfully added to project"))
}

func IsActiveProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	// Pozovi servis za dobijanje statusa svih taskova u projektu
	status, err := service.IsActiveProject(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Vrati rezultat u JSON formatu
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"result": status})
}
