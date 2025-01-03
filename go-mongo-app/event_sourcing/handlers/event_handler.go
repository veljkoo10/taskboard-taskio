package handlers

import (
	"encoding/json"
	"event_sourcing/models"
	"event_sourcing/repository"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

// EventHandler processes events for both HTTP and internal event processing.
type EventHandler struct {
	repo *repository.ESDBClient
}

// NewEventHandler creates a new EventHandler with a given repository.
func NewEventHandler(repo *repository.ESDBClient) *EventHandler {
	return &EventHandler{repo: repo}
}

// ProcessEventHandler will handle HTTP requests to process events (POST)
func (h *EventHandler) ProcessEventHandler(w http.ResponseWriter, r *http.Request) {
	var event model.Event
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&event); err != nil {
		http.Error(w, "Failed to decode event data", http.StatusBadRequest)
		return
	}

	message, err := h.processEvent(event)
	if err != nil {
		http.Error(w, "Failed to process event", http.StatusInternalServerError)
		return
	}

	if message != "" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(message))
	} else {
		http.Error(w, "Event type not handled", http.StatusBadRequest)
	}
}

// GetEventsHandler will handle HTTP requests to get events for a specific project (GET)
func (h *EventHandler) GetEventsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the projectID variable from the URL
	vars := mux.Vars(r)
	projectID := vars["projectID"]
	if projectID == "" {
		http.Error(w, "Missing projectID parameter", http.StatusBadRequest)
		return
	}

	// Fetch events for the given project
	events, err := h.repo.GetEventsByProjectID(projectID)
	if err != nil {
		http.Error(w, "Failed to retrieve events", http.StatusInternalServerError)
		return
	}

	// Respond with a JSON array of events
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(events); err != nil {
		http.Error(w, "Failed to encode events", http.StatusInternalServerError)
	}
}

func (h *EventHandler) processEvent(event model.Event) (string, error) {
	var message string
	switch event.Type {
	case model.MemberAddedType:
		if err := h.repo.StoreEvent(event.ProjectID, event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully added member to project"
	case model.MemberRemovedType:
		if err := h.repo.StoreEvent(event.ProjectID, event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully removed member from project"
	case model.MemberAddedTaskType:
		if err := h.repo.StoreEvent(event.ProjectID, event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully added member to task"
	case model.MemberRemovedTaskType:
		if err := h.repo.StoreEvent(event.ProjectID, event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully removed member from task"
	case model.TaskCreatedType:
		if err := h.repo.StoreEvent(event.ProjectID, event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully created task"
	case model.TaskStatusChangedType:
		if err := h.repo.StoreEvent(event.ProjectID, event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully changed task status"
	case model.DocumentAddedType:
		if err := h.repo.StoreEvent(event.ProjectID, event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully added document"
	default:
		log.Printf("Unhandled event type: %s\n", event.Type)
		return "", nil
	}

	return message, nil
}
