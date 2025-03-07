package handlers

import (
	"analytics-service/db"
	"analytics-service/service"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type AnalyticsHandler struct {
	logger   *log.Logger
	repo     *db.AnalyticsRepo
	natsConn *nats.Conn
}
type KeyAccount struct{}

type KeyRole struct{}

const (
	Manager = "Manager"
	Member  = "Member"
)

func NewAnalyticsHandler(l *log.Logger, r *db.AnalyticsRepo, natsConn *nats.Conn) *AnalyticsHandler {
	return &AnalyticsHandler{l, r, natsConn}
}

// CountUserTasks - Handler za brojanje taskova korisnika
func (h *AnalyticsHandler) CountUserTasks(w http.ResponseWriter, r *http.Request) {
	// Dohvatanje user_id iz URL-a
	vars := mux.Vars(r)
	userID := vars["user_id"]
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
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

	// Pozivamo servis da izračuna broj taskova
	count, err := service.CountUserTasks(userID, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Slanje rezultata kao JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"task_count": count})
}

// TaskStatusHandler - Handler za brojanje taskova po statusu
func (h *AnalyticsHandler) CountUserTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Koristimo mux.Vars da dobijemo user_id iz URL-a
	vars := mux.Vars(r)
	userID := vars["user_id"] // user_id je parametar u ruti

	// Proveravamo da li je user_id prisutan
	if userID == "" {
		http.Error(w, "userID is required", http.StatusBadRequest)
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

	// Pozivamo servis koji broji taskove po statusima
	statusCount, err := service.CountUserTasksByStatus(userID, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Postavljamo Content-Type na application/json
	w.Header().Set("Content-Type", "application/json")

	// Vraćamo brojanje taskova po statusima kao JSON odgovor
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(statusCount); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// UserTasksAndProjectHandler - Handler za vraćanje taskova i projekta korisnika
func (h *AnalyticsHandler) UserTasksAndProjectHandler(w http.ResponseWriter, r *http.Request) {
	// Koristimo mux.Vars da dobijemo user_id iz URL-a
	vars := mux.Vars(r)
	userID := vars["user_id"] // user_id je parametar u ruti

	// Proveravamo da li je user_id prisutan
	if userID == "" {
		http.Error(w, "userID is required", http.StatusBadRequest)
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

	// Pozivamo servis koji vraća taskove i projekat korisnika
	data, err := service.GetUserTasksAndProject(userID, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Postavljamo Content-Type na application/json
	w.Header().Set("Content-Type", "application/json")

	// Vraćamo podatke o taskovima i projektu kao JSON odgovor
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// CheckIfProjectCompletedOnTime - Proverava da li su projekti korisnika završeni na vreme
func (h *AnalyticsHandler) CheckIfProjectCompletedOnTime(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

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

	// Dohvatanje projekata korisnika
	projects, err := service.GetUserProjects(userID, token)
	if err != nil {
		http.Error(w, "Failed to fetch user projects", http.StatusInternalServerError)
		return
	}

	result := []map[string]interface{}{}

	for _, project := range projects {
		// Provera statusa projekta
		isActive, err := service.CheckProjectStatus(project.ID.Hex(), token)
		if err != nil {
			http.Error(w, "Failed to fetch project status", http.StatusInternalServerError)
			return
		}

		// Parsiranje predviđenog datuma završetka
		expectedEndDate, err := time.Parse("2006-01-02", project.ExpectedEndDate)
		if err != nil {
			http.Error(w, "Invalid project expected end date format", http.StatusInternalServerError)
			return
		}

		completedOnTime := false
		if !isActive {
			// Ako projekat nije aktivan, uporedi trenutni datum sa očekivanim završetkom
			completedOnTime = time.Now().Before(expectedEndDate) || time.Now().Equal(expectedEndDate)
		}

		result = append(result, map[string]interface{}{
			"project":         project.Title,
			"completed":       !isActive,
			"completedOnTime": completedOnTime,
			"expectedEndDate": project.ExpectedEndDate,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleStatusChange handles status change events from task-service
func (h *AnalyticsHandler) HandleStatusChange(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		TaskID         string `json:"task_id"`
		PreviousStatus string `json:"previous_status"`
		NewStatus      string `json:"new_status"`
		Timestamp      string `json:"timestamp"`
	}

	// Parse payload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, payload.Timestamp)
	if err != nil {
		http.Error(w, "Invalid timestamp", http.StatusBadRequest)
		return
	}

	// Record status change using the service
	err = service.RecordStatusChange(payload.TaskID, payload.PreviousStatus, payload.NewStatus, timestamp)
	if err != nil {
		http.Error(w, "Failed to record status change: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleGetTaskAnalytics handles fetching analytics for a specific task
func (h *AnalyticsHandler) HandleGetTaskAnalytics(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	// Fetch task analytics using the service
	analytics, err := service.GetTaskAnalytics(taskID)
	if err != nil {
		http.Error(w, "Failed to fetch analytics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send response as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(analytics); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *AnalyticsHandler) GetUserTaskAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	// Ekstraktujemo userID iz URL parametara
	vars := mux.Vars(r)
	userID := vars["userID"]
	if userID == "" {
		http.Error(w, "userID is required", http.StatusBadRequest)
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

	// Pozivamo funkciju iz servisa
	analyticsList, err := service.GetUserTaskAnalytics(userID, token)
	if err != nil {
		http.Error(w, "Failed to fetch user task analytics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Vraćamo JSON odgovor
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(analyticsList); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}
func (uh *AnalyticsHandler) MiddlewareExtractUserFromHeader(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
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
func (h *AnalyticsHandler) extractUserAndRoleFromToken(tokenString string) (userID string, role string, err error) {
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

func (h *AnalyticsHandler) RoleRequired(next http.HandlerFunc, roles ...string) http.HandlerFunc {
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
