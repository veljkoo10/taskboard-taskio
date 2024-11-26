package handlers

import (
	"context"
	"encoding/json"
	"github.com/gocql/gocql"
	"log"
	"net/http"
	"notification-service/models"
	"notification-service/repoNotification"
	"time"

	"github.com/gorilla/mux"
)

// KeyNotification je ključ za kontekst
type KeyNotification struct{}

// NotificationHandler struktura
type NotificationHandler struct {
	logger *log.Logger
	repo   *repoNotification.NotificationRepo // Reference na NotificationRepo iz repoNotification
}

// NewNotificationHandler kreira novi NotificationHandler sa prosleđenim logerom i repo-om
func NewNotificationHandler(l *log.Logger, r *repoNotification.NotificationRepo) *NotificationHandler {
	return &NotificationHandler{logger: l, repo: r}
}

// CreateUserHandler unosi novog korisnika u bazu
func (nh *NotificationHandler) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FirstName string    `json:"first_name"`
		LastName  string    `json:"last_name"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
	}

	// Parsiraj JSON telo zahteva
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Kreiraj User objekat
	user := models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		CreatedAt: req.CreatedAt,
	}

	// Pozovi repo za unos korisnika u bazu
	err = nh.repo.InsertUser(&user)
	if err != nil {
		http.Error(w, "Failed to insert user", http.StatusInternalServerError)
		return
	}

	// Uspešan odgovor
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "User created",
	})
	if err != nil {
		http.Error(w, "Unable to encode response", http.StatusInternalServerError)
	}
}

// Middleware za deserializaciju
func (n *NotificationHandler) MiddlewareNotificationDeserialization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		notification := &models.Notification{} // Use pointer here
		err := json.NewDecoder(h.Body).Decode(notification)
		if err != nil {
			http.Error(rw, "Unable to decode JSON", http.StatusBadRequest)
			n.logger.Fatal(err)
			return
		}
		ctx := context.WithValue(h.Context(), KeyNotification{}, notification)
		h = h.WithContext(ctx)
		next.ServeHTTP(rw, h)
	})
}

// Middleware za postavljanje Content-Type header-a
func (n *NotificationHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		rw.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(rw, h)
	})
}

// GetNotifications vraća sve notifikacije za korisnika
func (n *NotificationHandler) GetNotifications(rw http.ResponseWriter, h *http.Request) {
	vars := mux.Vars(h)
	userIDStr := vars["user_id"]

	// Konvertuj userID u gocql.UUID
	userID, err := gocql.ParseUUID(userIDStr)
	if err != nil {
		n.logger.Println("Invalid user ID format:", err)
		http.Error(rw, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	// Pozovi repo da dobijemo notifikacije za korisnika
	notifications, err := n.repo.GetNotificationsByUser(userID)
	if err != nil {
		n.logger.Println("Database exception:", err)
		http.Error(rw, "Unable to fetch notifications", http.StatusInternalServerError)
		return
	}

	// Pošaljemo notifikacije kao JSON odgovor
	err = json.NewEncoder(rw).Encode(notifications)
	if err != nil {
		n.logger.Println("Unable to encode notifications to JSON:", err)
		http.Error(rw, "Unable to encode notifications", http.StatusInternalServerError)
		return
	}
}

// CreateNotificationHandler kreira novu notifikaciju
func (nh *NotificationHandler) CreateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    string    `json:"user_id"`
		Message   string    `json:"message"`
		IsActive  bool      `json:"is_active"`
		CreatedAt time.Time `json:"created_at"` // Koristi time.Time
	}

	// Parsiraj JSON telo zahteva
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Konvertuj UserID iz string u gocql.UUID
	userID, err := gocql.ParseUUID(req.UserID)
	if err != nil {
		http.Error(w, "Invalid user_id format", http.StatusBadRequest)
		return
	}

	// Kreiraj Notification objekat
	notification := models.Notification{
		UserID:    userID,
		Message:   req.Message,
		IsActive:  req.IsActive,
		CreatedAt: req.CreatedAt, // Direktno koristi time.Time iz req
	}

	// Pozovi repo za unos notifikacije u bazu
	err = nh.repo.InsertNotification(&notification)
	if err != nil {
		http.Error(w, "Failed to insert notification", http.StatusInternalServerError)
		return
	}

	// Generisanje UUID za novu notifikaciju
	notificationID, _ := gocql.RandomUUID()

	// Uspešan odgovor sa statusom i ID-em kreirane notifikacije
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "Notification created",
		"id":     notificationID, // Vraćanje ID-a nove notifikacije
	})
	if err != nil {
		http.Error(w, "Unable to encode response", http.StatusInternalServerError)
		return
	}
}

// GetAllNotificationsHandler vraća sve notifikacije
func (h *NotificationHandler) GetAllNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Println("Received GET request for /notifications/GetAll")

	// Dobijanje notifikacija iz repozitorijuma
	notifications, err := h.repo.GetAllNotifications()
	if err != nil {
		h.logger.Printf("Error fetching notifications: %v", err)
		http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	// Proveravamo da li ima notifikacija
	if len(notifications) == 0 {
		h.logger.Println("No notifications found")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`)) // Prazan JSON niz ako nema podataka
		return
	}

	h.logger.Printf("Returning %d notifications", len(notifications))

	// Postavljanje zaglavlja i vraćanje podataka u JSON formatu
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(notifications); err != nil {
		h.logger.Printf("Error encoding notifications to JSON: %v", err)
		http.Error(w, "Failed to encode notifications", http.StatusInternalServerError)
	}
}
