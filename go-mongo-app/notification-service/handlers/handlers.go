package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"log"
	"net/http"
	"notification-service/models"
	"notification-service/repoNotification"
	"strings"
	"time"
)

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
