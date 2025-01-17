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

type KeyNotification struct{}

type NotificationHandler struct {
	logger *log.Logger
	repo   *repoNotification.NotificationRepo
}

func NewNotificationHandler(l *log.Logger, r *repoNotification.NotificationRepo) *NotificationHandler {
	return &NotificationHandler{logger: l, repo: r}
}

func (n *NotificationHandler) FetchNotificationByID(rw http.ResponseWriter, h *http.Request) {
	vars := mux.Vars(h)
	id := vars["id"]

	notificationID, err := gocql.ParseUUID(id)
	if err != nil {
		http.Error(rw, "Invalid UUID format", http.StatusBadRequest)
		n.logger.Println("Invalid UUID format:", err)
		return
	}

	notification, err := n.repo.FetchByID(notificationID)
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

func (n *NotificationHandler) FetchNotificationsByUser(rw http.ResponseWriter, h *http.Request) {
	vars := mux.Vars(h)
	userID := vars["id"]

	n.logger.Println("User ID:", userID)

	notifications, err := n.repo.FetchByUserID(userID)
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

func (n *NotificationHandler) FetchAllNotifications(rw http.ResponseWriter, r *http.Request) {
	notifications, err := n.repo.FetchAllNotifications()
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

func (n *NotificationHandler) UpdateNotification(rw http.ResponseWriter, h *http.Request) {
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

func (n *NotificationHandler) MarkNotificationsAsRead(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	if userID == "" {
		http.Error(rw, "User ID is required", http.StatusBadRequest)
		n.logger.Println("User ID is empty")
		return
	}

	err := n.repo.MarkAllAsRead(userID)
	if err != nil {
		http.Error(rw, "Failed to mark notifications as read", http.StatusInternalServerError)
		n.logger.Printf("Error marking notifications as read for user %s: %v", userID, err)
		return
	}

	rw.WriteHeader(http.StatusNoContent)
}

func Conn() (*nats.Conn, error) {
	conn, err := nats.Connect("nats://nats:4222")
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return conn, nil
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

func (uh *NotificationHandler) MiddlewareExtractUserFromHeader(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(rw http.ResponseWriter, h *http.Request) {
		authHeader := h.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(rw, "No Authorization header found", http.StatusUnauthorized)
			uh.logger.Println("No Authorization header:", authHeader)
			return
		}

		tokenString := ""
		if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
			tokenString = authHeader[7:]
		} else {
			http.Error(rw, "Invalid Authorization header format", http.StatusUnauthorized)
			uh.logger.Println("Invalid Authorization header format:", authHeader)
			return
		}

		userID, role, err := uh.extractUserAndRoleFromToken(tokenString)
		if err != nil {
			uh.logger.Println("Token extraction failed:", err)
			http.Error(rw, `{"message": "Invalid token"}`, http.StatusUnauthorized)
			return
		}

		uh.logger.Println("User ID is:", userID, "Role is:", role)

		ctx := context.WithValue(h.Context(), KeyAccount{}, userID)
		ctx = context.WithValue(ctx, KeyRole{}, role)

		h = h.WithContext(ctx)

		next(rw, h)
	}
}

func (uh *NotificationHandler) extractUserAndRoleFromToken(tokenString string) (userID string, role string, err error) {
	secretKey := []byte(os.Getenv("TOKEN_SECRET"))

	parsedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil || !parsedToken.Valid {
		return "", "", fmt.Errorf("invalid token: %v", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", fmt.Errorf("invalid token claims")
	}

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
	return func(rw http.ResponseWriter, req *http.Request) {
		role, ok := req.Context().Value(KeyRole{}).(string)
		if !ok {
			http.Error(rw, "Role not found in context", http.StatusForbidden)
			return
		}

		for _, r := range roles {
			if role == r {
				next(rw, req)
				return
			}
		}

		http.Error(rw, "Forbidden", http.StatusForbidden)
	}
}
