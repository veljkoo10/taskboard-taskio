package handlers

import (
	"context"
	"encoding/json"
	"event_sourcing/models"
	"event_sourcing/repository"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strings"
)

type KeyAccount struct{}

type KeyRole struct{}

// EventHandler processes events for both HTTP and internal event processing.
type EventHandler struct {
	logger *log.Logger
	repo   *repository.ESDBClient
}

// NewEventHandler creates a new EventHandler with a given repository.
func NewEventHandler(repo *repository.ESDBClient, logger *log.Logger) *EventHandler {
	return &EventHandler{logger: logger, repo: repo}
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
func (h *EventHandler) GetAllEventsHandler(w http.ResponseWriter, r *http.Request) {
	events, err := h.repo.GetAllEvents()
	if err != nil {
		// Log error without sending a server error to the client
		log.Printf("Error fetching events: %v", err)
		// Return empty events without failure
		events = []model.Event{}
	}

	// If no events were found, return a custom message
	if len(events) == 0 {
		// Send a message indicating no events were found
		w.WriteHeader(http.StatusOK) // No error, just an empty state
		w.Write([]byte("No events found"))
		return
	}

	// If events were found, respond with the events
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(events); err != nil {
		// Handle failure to encode the events
		http.Error(w, "Failed to encode events", http.StatusInternalServerError)
	}
}

// Optional: Filter events by ProjectID in memory.
func FilterEventsByProjectID(events []model.Event, projectID string) []model.Event {
	var filtered []model.Event
	for _, event := range events {
		if event.ProjectID == projectID {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func (h *EventHandler) processEvent(event model.Event) (string, error) {
	var message string
	switch event.Type {
	case model.MemberAddedType:
		if err := h.repo.StoreEvent(event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully added member to project"
	case model.MemberRemovedType:
		if err := h.repo.StoreEvent(event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully removed member from project"
	case model.MemberAddedTaskType:
		if err := h.repo.StoreEvent(event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully added member to task"
	case model.MemberRemovedTaskType:
		if err := h.repo.StoreEvent(event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully removed member from task"
	case model.TaskCreatedType:
		if err := h.repo.StoreEvent(event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully created task"
	case model.TaskStatusChangedType:
		if err := h.repo.StoreEvent(event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully changed task status"
	case model.DocumentAddedType:
		if err := h.repo.StoreEvent(event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		message = "Successfully added document"
	case model.ProjectCreatedType:
		if err := h.repo.StoreEvent(event); err != nil {
			log.Printf("Failed to store event: %v", err)
			return "", err
		}
		// The message will indicate that the project was created successfully
		message = "Successfully created project"
	default:
		log.Printf("Unhandled event type: %s\n", event.Type)
		return "", nil
	}

	return message, nil
}

func (uh *EventHandler) MiddlewareExtractUserFromHeader(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
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
func (h *EventHandler) extractUserAndRoleFromToken(tokenString string) (userID string, role string, err error) {
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

func (h *EventHandler) RoleRequired(next http.HandlerFunc, roles ...string) http.HandlerFunc {
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
