package handlers

import (
	"analytics-service/db"
	"analytics-service/service"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"log"
	"net/http"
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
