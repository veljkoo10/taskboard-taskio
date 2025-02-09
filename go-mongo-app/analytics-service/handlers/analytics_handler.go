package handlers

import (
	"analytics-service/db"
	"analytics-service/service"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"log"
	"net/http"
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
func CountUserTasks(w http.ResponseWriter, r *http.Request) {
	// Dohvatanje user_id iz URL-a
	vars := mux.Vars(r)
	userID := vars["user_id"]
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	// Pozivamo servis da izračuna broj taskova
	count, err := service.CountUserTasks(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Slanje rezultata kao JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"task_count": count})
}

// TaskStatusHandler - Handler za brojanje taskova po statusu
func CountUserTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Koristimo mux.Vars da dobijemo user_id iz URL-a
	vars := mux.Vars(r)
	userID := vars["user_id"] // user_id je parametar u ruti

	// Proveravamo da li je user_id prisutan
	if userID == "" {
		http.Error(w, "userID is required", http.StatusBadRequest)
		return
	}

	// Pozivamo servis koji broji taskove po statusima
	statusCount, err := service.CountUserTasksByStatus(userID)
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
func UserTasksAndProjectHandler(w http.ResponseWriter, r *http.Request) {
	// Koristimo mux.Vars da dobijemo user_id iz URL-a
	vars := mux.Vars(r)
	userID := vars["user_id"] // user_id je parametar u ruti

	// Proveravamo da li je user_id prisutan
	if userID == "" {
		http.Error(w, "userID is required", http.StatusBadRequest)
		return
	}

	// Pozivamo servis koji vraća taskove i projekat korisnika
	data, err := service.GetUserTasksAndProject(userID)
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
func CheckIfProjectCompletedOnTime(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	// Dohvatanje projekata korisnika
	projects, err := service.GetUserProjects(userID)
	if err != nil {
		http.Error(w, "Failed to fetch user projects", http.StatusInternalServerError)
		return
	}

	result := []map[string]interface{}{}

	for _, project := range projects {
		// Provera statusa projekta
		isActive, err := service.CheckProjectStatus(project.ID.Hex())
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
func HandleStatusChange(w http.ResponseWriter, r *http.Request) {
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
func HandleGetTaskAnalytics(w http.ResponseWriter, r *http.Request) {
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

func GetUserTaskAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	// Ekstraktujemo userID iz URL parametara
	vars := mux.Vars(r)
	userID := vars["userID"]
	if userID == "" {
		http.Error(w, "userID is required", http.StatusBadRequest)
		return
	}

	// Pozivamo funkciju iz servisa
	analyticsList, err := service.GetUserTaskAnalytics(userID)
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
