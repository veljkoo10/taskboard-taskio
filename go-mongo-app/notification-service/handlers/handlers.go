package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gocql/gocql"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"log"
	"net/http"
	"notification-service/models"
	"notification-service/repoNotification"
	"os"
	"strings"
	"time"
)

type KeyAccount struct{}

type KeyRole struct{}

// KeyNotification je ključ za kontekst
type KeyNotification struct{}

// NotificationHandler struktura
type NotificationHandler struct {
	logger *log.Logger
	repo   *repoNotification.NotificationRepo
}

// NewNotificationHandler kreira novi NotificationHandler
func NewNotificationHandler(l *log.Logger, r *repoNotification.NotificationRepo) *NotificationHandler {
	return &NotificationHandler{logger: l, repo: r}
}

func (n *NotificationHandler) CreateNotification(rw http.ResponseWriter, h *http.Request) {
	var notification models.Notification

	decoder := json.NewDecoder(h.Body)
	err := decoder.Decode(&notification)
	if err != nil {
		http.Error(rw, "Unable to decode json", http.StatusBadRequest)
		n.logger.Fatal(err)
		return
	}

	if err := notification.Validate(); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	err = n.repo.Create(&notification)
	if err != nil {
		http.Error(rw, "Failed to create notification", http.StatusInternalServerError)
		n.logger.Print("Error inserting notification:", err)
		return
	}

	rw.WriteHeader(http.StatusCreated)
	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(notification)
	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		n.logger.Fatal("Unable to encode response:", err)
	}
}

func (n *NotificationHandler) GetNotificationByID(rw http.ResponseWriter, h *http.Request) {
	vars := mux.Vars(h)
	id := vars["id"]

	notificationID, err := gocql.ParseUUID(id)
	if err != nil {
		http.Error(rw, "Invalid UUID format", http.StatusBadRequest)
		n.logger.Println("Invalid UUID format:", err)
		return
	}

	notification, err := n.repo.GetByID(notificationID)
	if err != nil {
		http.Error(rw, "Notification not found", http.StatusNotFound)
		n.logger.Println("Error fetching notification:", err)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(notification)
	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		n.logger.Fatal("Unable to encode response:", err)
	}
}

func (n *NotificationHandler) GetNotificationsByUserID(rw http.ResponseWriter, h *http.Request) {
	vars := mux.Vars(h)
	userID := vars["id"]

	n.logger.Println("User ID:", userID)

	notifications, err := n.repo.GetByUserID(userID)
	if err != nil {
		http.Error(rw, "Error fetching notifications", http.StatusInternalServerError)
		n.logger.Println("Error fetching notifications:", err)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(notifications)
	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		n.logger.Fatal("Unable to encode response:", err)
	}
}

func (n *NotificationHandler) UpdateNotificationStatus(rw http.ResponseWriter, h *http.Request) {
	vars := mux.Vars(h)
	id := vars["id"]

	notificationID, err := gocql.ParseUUID(id)
	if err != nil {
		http.Error(rw, "Invalid UUID format", http.StatusBadRequest)
		n.logger.Println("Invalid UUID format:", err)
		return
	}

	type statusRequest struct {
		Status    models.NotificationStatus `json:"status"`
		CreatedAt time.Time                 `json:"created_at"`
	}

	var req statusRequest
	decoder := json.NewDecoder(h.Body)
	err = decoder.Decode(&req)
	if err != nil {
		http.Error(rw, "Unable to decode JSON", http.StatusBadRequest)
		n.logger.Println("Error decoding JSON:", err)
		return
	}

	if req.Status != models.Unread && req.Status != models.Read {
		http.Error(rw, "Invalid status value", http.StatusBadRequest)
		return
	}

	userID, ok := h.Context().Value(KeyNotification{}).(string)
	if !ok {
		n.logger.Println("User id not found in context")
		http.Error(rw, "User id not found in context", http.StatusUnauthorized)
		return
	}

	err = n.repo.UpdateStatus(req.CreatedAt, userID, notificationID, req.Status)
	if err != nil {
		http.Error(rw, "Error updating notification status", http.StatusInternalServerError)
		n.logger.Println("Error updating notification status:", err)
		return
	}

	rw.WriteHeader(http.StatusNoContent)
}
func (n *NotificationHandler) GetAllNotifications(rw http.ResponseWriter, r *http.Request) {
	notifications, err := n.repo.GetAllNotifications()
	if err != nil {
		http.Error(rw, "Error fetching all notifications", http.StatusInternalServerError)
		n.logger.Println("Error fetching all notifications:", err)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(notifications)
	if err != nil {
		http.Error(rw, "Unable to convert notifications to JSON", http.StatusInternalServerError)
		n.logger.Fatal("Unable to encode response:", err)
	}
}
func (n *NotificationHandler) NotificationListener() {
	n.logger.Println("method started")
	nc, err := Conn()
	if err != nil {
		log.Fatal("Error connecting to NATS:", err)
	}
	defer nc.Close()

	subjectJoined := "project.joined"
	_, err = nc.Subscribe(subjectJoined, func(msg *nats.Msg) {
		fmt.Printf("User received notification: %s\n", string(msg.Data))

		var data struct {
			UserID      string `json:"userId"`
			ProjectName string `json:"projectName"`
		}

		err := json.Unmarshal(msg.Data, &data)
		if err != nil {
			log.Println("Error unmarshalling message:", err)
			return
		}

		fmt.Printf("User ID: %s, Project Name: %s\n", data.UserID, data.ProjectName)

		message := fmt.Sprintf("You have been added to the \"%s\" project", strings.Title(data.ProjectName))

		notification := models.Notification{
			UserID:    data.UserID,
			Message:   message,
			CreatedAt: time.Now(),
			Status:    models.Unread,
		}

		err = n.repo.Create(&notification)
		if err != nil {
			n.logger.Print("Error inserting notification:", err)
			return
		}
	})

	if err != nil {
		log.Println("Error subscribing to NATS subject:", err)
	}

	taskJoined := "task.joined"
	_, err = nc.Subscribe(taskJoined, func(msg *nats.Msg) {
		fmt.Printf("User received notification: %s\n", string(msg.Data))

		var data struct {
			UserID   string `json:"userId"`
			TaskName string `json:"taskName"`
		}

		err := json.Unmarshal(msg.Data, &data)
		if err != nil {
			log.Println("Error unmarshalling message:", err)
			return
		}

		fmt.Printf("User ID: %s, Task Name: %s\n", data.UserID, data.TaskName)

		message := fmt.Sprintf("You have been added to the \"%s\" task", strings.Title(data.TaskName))

		notification := models.Notification{
			UserID:    data.UserID,
			Message:   message,
			CreatedAt: time.Now(),
			Status:    models.Unread,
		}

		err = n.repo.Create(&notification)
		if err != nil {
			n.logger.Print("Error inserting notification:", err)
			return
		}
	})

	if err != nil {
		log.Println("Error subscribing to NATS subject:", err)
	}

	subjectRemoved := "project.removed"
	_, err = nc.Subscribe(subjectRemoved, func(msg *nats.Msg) {
		fmt.Printf("User received removal notification: %s\n", string(msg.Data))

		var data struct {
			UserID      string `json:"userId"`
			ProjectName string `json:"projectName"`
		}

		err := json.Unmarshal(msg.Data, &data)
		if err != nil {
			log.Println("Error unmarshalling message:", err)
			return
		}

		fmt.Printf("User ID: %s, Project Name: %s\n", data.UserID, data.ProjectName)

		message := fmt.Sprintf("You have been removed from the \"%s\" project", strings.Title(data.ProjectName))

		notification := models.Notification{
			UserID:    data.UserID,
			Message:   message,
			CreatedAt: time.Now(),
			Status:    models.Unread,
		}

		err = n.repo.Create(&notification)
		if err != nil {
			n.logger.Print("Error inserting notification:", err)
			return
		}
	})

	if err != nil {
		log.Println("Error subscribing to NATS subject:", err)
	}

	taskRemoved := "task.removed"
	_, err = nc.Subscribe(taskRemoved, func(msg *nats.Msg) {
		fmt.Printf("User received removal notification: %s\n", string(msg.Data))

		var data struct {
			UserID   string `json:"userId"`
			TaskName string `json:"taskName"`
		}

		err := json.Unmarshal(msg.Data, &data)
		if err != nil {
			log.Println("Error unmarshalling message:", err)
			return
		}

		fmt.Printf("User ID: %s, Task Name: %s\n", data.UserID, data.TaskName)

		message := fmt.Sprintf("You have been removed from the \"%s\" task", strings.Title(data.TaskName))

		notification := models.Notification{
			UserID:    data.UserID,
			Message:   message,
			CreatedAt: time.Now(),
			Status:    models.Unread,
		}

		err = n.repo.Create(&notification)
		if err != nil {
			n.logger.Print("Error inserting notification:", err)
			return
		}
	})
	if err != nil {
		log.Println("Error subscribing to NATS subject:", err)
	}

	statusUpdate := "task.status.update"
	_, err = nc.Subscribe(statusUpdate, func(msg *nats.Msg) {
		fmt.Printf("User received notification: %s\n", string(msg.Data))

		var update struct {
			TaskName   string   `json:"taskName"`
			TaskStatus string   `json:"taskStatus"`
			MemberIds  []string `json:"memberIds"`
		}

		if err := json.Unmarshal(msg.Data, &update); err != nil {
			n.logger.Printf("Error unmarshalling task status update message: %v", err)
			return
		}
		fmt.Printf("Received status update for Task %s: %s\n", update.TaskName, update.TaskStatus)

		message := fmt.Sprintf("The status of the \"%s\" task has been changed to \"%s\"", strings.Title(update.TaskName), strings.Title(update.TaskStatus))

		for _, memberID := range update.MemberIds {
			notification := models.Notification{
				UserID:    memberID,
				Message:   message,
				CreatedAt: time.Now(),
				Status:    models.Unread,
			}

			if err := n.repo.Create(&notification); err != nil {
				n.logger.Printf("Error inserting notification for user %s: %v", memberID, err)
				continue
			}

			n.logger.Printf("Notification sent to user %s\n", memberID)
		}
	})
	if err != nil {
		log.Println("Error subscribing to NATS subject:", err)
	}

	select {}
}

func Conn() (*nats.Conn, error) {
	conn, err := nats.Connect("nats://nats:4222")
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return conn, nil
}
func (n *NotificationHandler) MarkAsRead(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	// Validacija userID-a
	if userID == "" {
		http.Error(rw, "User ID is required", http.StatusBadRequest)
		n.logger.Println("User ID is empty")
		return
	}

	// Poziv na repo za ažuriranje
	err := n.repo.MarkAllAsRead(userID)
	if err != nil {
		http.Error(rw, "Failed to mark notifications as read", http.StatusInternalServerError)
		n.logger.Printf("Error marking notifications as read for user %s: %v", userID, err)
		return
	}

	rw.WriteHeader(http.StatusNoContent)
}
func (uh *NotificationHandler) MiddlewareExtractUserFromHeader(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
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
func (uh *NotificationHandler) extractUserAndRoleFromToken(tokenString string) (userID string, role string, err error) {
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

func (uh *NotificationHandler) RoleRequired(next http.HandlerFunc, roles ...string) http.HandlerFunc {
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
