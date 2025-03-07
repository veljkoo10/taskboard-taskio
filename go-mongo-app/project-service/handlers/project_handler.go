package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/nats-io/nats.go"
	"log"
	"net/http"
	"os"
	"project-service/db"
	"project-service/models"
	"project-service/service"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type ProjectHandler struct {
	logger   *log.Logger
	repo     *db.ProjectRepo
	natsConn *nats.Conn
}
type KeyAccount struct{}

type KeyRole struct{}

const (
	Manager = "Manager"
	Member  = "Member"
)

func NewProjectsHandler(l *log.Logger, r *db.ProjectRepo, natsConn *nats.Conn) *ProjectHandler {
	return &ProjectHandler{l, r, natsConn}
}
func (h *ProjectHandler) GetUsersForProjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	usersIDs, err := service.GetUsersForProject(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "No Authorization header found", http.StatusUnauthorized)
		h.logger.Println("No Authorization header:", authHeader)
		return
	}

	// Expect the format "Bearer <token>"
	token := ""

	if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
		token = authHeader[7:]
	} else {
		http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
		h.logger.Println("Invalid Authorization header format:", authHeader)
		return
	}

	users, err := service.GetUserDetails(usersIDs, token)
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

func (h *ProjectHandler) GetProjectIDByTitle(w http.ResponseWriter, r *http.Request) {
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

func (h *ProjectHandler) GetProjectsByUserID(w http.ResponseWriter, r *http.Request) {
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
func (h *ProjectHandler) GetProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := service.GetAllProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	projectID, err := service.CreateProject(project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generisanje događaja za kreirani projekat
	currentTime := time.Now().Add(1 * time.Hour)
	formattedTime := currentTime.Format(time.RFC3339)

	event := map[string]interface{}{
		"type": "Project Created",
		"time": formattedTime,
		"event": map[string]interface{}{
			"projectId": projectID, // Ispravno korišćenje generisanog ID-a
			"managerId": managerID,
			"title":     project.Title, // Ispravno polje
		},
		"projectId": projectID,
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "No Authorization header found", http.StatusUnauthorized)
		h.logger.Println("No Authorization header:", authHeader)
		return
	}

	// Expect the format "Bearer <token>"
	token := ""

	if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
		token = authHeader[7:]
	} else {
		http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
		h.logger.Println("Invalid Authorization header format:", authHeader)
		return
	}

	// Slanje događaja u bazu
	if err := h.sendEventToDatabase(event, token); err != nil {
		http.Error(w, "Failed to send event to analytics service", http.StatusInternalServerError)
		return
	}

	h.logger.Println("Project created and event sent:", projectID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"projectId": projectID,
		"message":   "Project successfully created",
	})
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

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "No Authorization header found", http.StatusUnauthorized)
		p.logger.Println("No Authorization header:", authHeader)
		return
	}

	// Expect the format "Bearer <token>"
	token := ""

	if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
		token = authHeader[7:]
	} else {
		http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
		p.logger.Println("Invalid Authorization header format:", authHeader)
		return
	}

	if err := service.AddUsersToProject(projectID, requestBody.UserIDs, token); err != nil {
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

		currentTime := time.Now().Add(1 * time.Hour)
		formattedTime := currentTime.Format(time.RFC3339)

		event := map[string]interface{}{
			"type": "Member Added to Project",
			"time": formattedTime,
			"event": map[string]interface{}{
				"memberId":  uid,
				"projectId": projectID,
			},
			"projectId": projectID,
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "No Authorization header found", http.StatusUnauthorized)
			p.logger.Println("No Authorization header:", authHeader)
			return
		}

		// Expect the format "Bearer <token>"
		token := ""

		if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
			token = authHeader[7:]
		} else {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			p.logger.Println("Invalid Authorization header format:", authHeader)
			return
		}

		if err := p.sendEventToDatabase(event, token); err != nil {
			http.Error(w, "Error sending event to analytics service", http.StatusInternalServerError)
			return
		}

	}
	p.logger.Println("a message has been sent")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Users successfully added to project"))
}
func (p *ProjectHandler) sendEventToDatabase(event interface{}, token string) error {
	analyticsServiceURL := fmt.Sprintf("http://event_sourcing:8080/event/append")

	// Marshal the event into JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshalling event: %v", err)
		return err
	}

	// Create a new POST request with the event data
	req, err := http.NewRequest("POST", analyticsServiceURL, bytes.NewBuffer(eventData))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return err
	}

	// Set the appropriate headers for the request
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token)) // Dodaj token u Authorization header

	// Use the default HTTP client (without TLS)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error sending request to analytics service: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to send event to analytics service: %s", resp.Status)
		return fmt.Errorf("failed to send event to analytics service: %s", resp.Status)
	}

	return nil
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

func (h *ProjectHandler) GetProjectByID(w http.ResponseWriter, r *http.Request) {
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

		currentTime := time.Now().Add(1 * time.Hour)
		formattedTime := currentTime.Format(time.RFC3339)

		event := map[string]interface{}{
			"type": "Member Removed from Project",
			"time": formattedTime,
			"event": map[string]interface{}{
				"memberId":  uid,
				"projectId": projectID,
			},
			"projectId": projectID,
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "No Authorization header found", http.StatusUnauthorized)
			p.logger.Println("No Authorization header:", authHeader)
			return
		}

		// Expect the format "Bearer <token>"
		token := ""

		if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
			token = authHeader[7:]
		} else {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			p.logger.Println("Invalid Authorization header format:", authHeader)
			return
		}

		// Send the event to the analytic service
		if err := p.sendEventToDatabase(event, token); err != nil {
			http.Error(w, "Failed to send event to analytics service", http.StatusInternalServerError)
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

func (h *ProjectHandler) HandleCheckProjectByTitle(w http.ResponseWriter, r *http.Request) {
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

func (h *ProjectHandler) AddTaskToProjectHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *ProjectHandler) IsActiveProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "No Authorization header found", http.StatusUnauthorized)
		h.logger.Println("No Authorization header:", authHeader)
		return
	}

	// Expect the format "Bearer <token>"
	token := ""

	if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
		token = authHeader[7:]
	} else {
		http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
		h.logger.Println("Invalid Authorization header format:", authHeader)
		return
	}

	// Pozovi servis za dobijanje statusa svih taskova u projektu
	status, err := service.IsActiveProject(projectID, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Vrati rezultat u JSON formatu
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"result": status})
}
func (uh *ProjectHandler) MiddlewareExtractUserFromHeader(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
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
func (uh *ProjectHandler) extractUserAndRoleFromToken(tokenString string) (userID string, role string, err error) {
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

func (uh *ProjectHandler) RoleRequired(next http.HandlerFunc, roles ...string) http.HandlerFunc {
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

func (uh *ProjectHandler) DeleteProjectByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Dohvati projectID iz URL parametara
	vars := mux.Vars(r)
	projectID, ok := vars["projectID"]
	if !ok || projectID == "" {
		http.Error(w, "projectID is required in the URL", http.StatusBadRequest)
		return
	}

	// Fetch the project details before deleting it
	project, err := service.GetProjectByID(projectID)
	if err != nil {
		http.Error(w, "Failed to fetch project details: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch the list of users associated with the project
	userIDs, err := service.GetUsersForProject(projectID)
	if err != nil {
		http.Error(w, "Failed to fetch users for project: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send notifications to all users that they have been removed from the project
	for _, userID := range userIDs {
		subject := "project.removed2"
		message := struct {
			UserID      string `json:"userId"`
			ProjectName string `json:"projectName"`
		}{
			UserID:      userID,
			ProjectName: project.Title,
		}

		if err := uh.sendNotification(subject, message); err != nil {
			http.Error(w, "Failed to send notification to user: "+userID, http.StatusInternalServerError)
			return
		}
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "No Authorization header found", http.StatusUnauthorized)
		uh.logger.Println("No Authorization header:", authHeader)
		return
	}

	// Expect the format "Bearer <token>"
	token := ""

	if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
		token = authHeader[7:]
	} else {
		http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
		uh.logger.Println("Invalid Authorization header format:", authHeader)
		return
	}

	// Pozovi repository za brisanje project po taskID-u
	err = service.DeleteProjectByID(projectID, token)
	if err != nil {
		http.Error(w, "Failed to delete project: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Uspešan odgovor
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"project deleted successfully"}`))
}
