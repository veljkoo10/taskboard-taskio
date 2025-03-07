package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strings"
	"workflow-service/models"
	"workflow-service/repoWorkflow"
)

type KeyAccount struct{}

type KeyRole struct{}
type WorkflowHandler struct {
	logger *log.Logger
	repo   *repoWorkflow.WorkflowRepository
}

func NewWorkflowHandler(repo *repoWorkflow.WorkflowRepository, logger *log.Logger) *WorkflowHandler {
	return &WorkflowHandler{logger: logger, repo: repo}
}

func (h *WorkflowHandler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var workflowRequest struct {
		TaskID         string   `json:"task_id"`
		DependencyTask []string `json:"dependency_task"`
		ProjectID      string   `json:"project_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&workflowRequest); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding request: %v", err), http.StatusBadRequest)
		return
	}

	if workflowRequest.TaskID == "" {
		http.Error(w, "task_id cannot be empty", http.StatusBadRequest)
		return
	}
	if workflowRequest.ProjectID == "" {
		http.Error(w, "project_id cannot be empty", http.StatusBadRequest)
		return
	}
	if len(workflowRequest.DependencyTask) == 0 {
		http.Error(w, "dependency_task cannot be empty", http.StatusBadRequest)
		return
	}

	err := h.repo.CheckForCycle(r.Context(), workflowRequest.TaskID, workflowRequest.DependencyTask)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cycle detected: %v", err), http.StatusBadRequest)
		return
	}

	workflow := models.Workflow{
		TaskID:         workflowRequest.TaskID,
		DependencyTask: workflowRequest.DependencyTask,
		ProjectID:      workflowRequest.ProjectID,
		IsActive:       true,
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

	err = h.repo.CreateWorkflow(r.Context(), workflow, token)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating workflow: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	response := map[string]string{
		"message":    "Workflow dependency successfully created",
		"task_id":    workflow.TaskID,
		"project_id": workflow.ProjectID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
	}
}

// GetWorkflowHandler vraća sve workflow-e.
func (h *WorkflowHandler) GetWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	// Dobavi sve workflow-e
	workflows, err := h.repo.GetAllWorkflows(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Vrati sve workflow-e kao JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(workflows); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
	}
}

func (h *WorkflowHandler) GetTaskByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Uzmi task ID iz URL-a
	taskID := mux.Vars(r)["id"]

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

	// Pozovi repo metod za dobijanje task-a
	task, err := repoWorkflow.GetTaskFromTaskService(taskID, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Pošaljemo zadatak kao JSON odgovor
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// CheckDependencyHandler proverava da li task ima zavisnost.
func (h *WorkflowHandler) CheckDependencyHandler(w http.ResponseWriter, r *http.Request) {
	// Uzmi task ID iz URL-a
	taskID := mux.Vars(r)["task_id"]

	// Loguj task_id da proveriš da li je ispravno došao
	log.Printf("Received task_id: %s", taskID)

	// Pozovi funkciju koja proverava zavisnost
	hasDependency, err := h.repo.CheckTaskDependency(r.Context(), taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error checking dependency: %v", err), http.StatusInternalServerError)
		return
	}

	// Ako postoji zavisnost, vraćamo true, inače false
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"has_dependency": hasDependency})
}
func (h *WorkflowHandler) GetTaskDependenciesHandler(w http.ResponseWriter, r *http.Request) {
	// Uzmi task ID iz URL-a
	taskID, ok := mux.Vars(r)["task_id"]
	if !ok || taskID == "" {
		http.Error(w, "Missing or invalid task_id", http.StatusBadRequest)
		return
	}

	// Pozovi funkciju iz repozitorijuma koja dobavlja zavisne taskove
	dependencies, err := h.repo.GetTaskDependencies(r.Context(), taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving dependencies: %v", err), http.StatusInternalServerError)
		return
	}

	// Vraćanje zavisnih taskova kao JSON odgovor
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"task_id":          taskID,
		"dependency_tasks": dependencies,
	})
}
func (h *WorkflowHandler) GetFlowByProjectIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID, ok := vars["project_id"]
	if !ok {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	flows, err := h.repo.GetAllWorkflowsByProjectID(r.Context(), projectID)
	if err != nil {
		http.Error(w, "Error fetching workflows: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(flows); err != nil {
		http.Error(w, "Error encoding response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteWorkflowByTaskIDHandler je HTTP handler za brisanje workflow-a na osnovu taskID-a iz URL-a.
func (h *WorkflowHandler) DeleteWorkflowByTaskIDHandler(w http.ResponseWriter, r *http.Request) {
	// Dohvati taskID iz URL parametara
	vars := mux.Vars(r)
	taskID, ok := vars["task_id"]
	if !ok || taskID == "" {
		http.Error(w, "taskID is required in the URL", http.StatusBadRequest)
		return
	}

	// Pozovi repository za brisanje workflow-a po taskID-u
	err := h.repo.DeleteWorkflowsByTaskID(taskID)
	if err != nil {
		http.Error(w, "Failed to delete workflows: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Uspešan odgovor
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Workflows deleted successfully"}`))
}

func (h WorkflowHandler) GetWorkflowByTaskIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID, ok := vars["task_id"]
	if !ok || taskID == "" {
		http.Error(w, "taskID is required in the URL", http.StatusBadRequest)
		return
	}

	// Pozovi repo metodu za pretragu workflow-a
	found, err := h.repo.FindWorkflowByTaskID(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Vrati rezultat
	if found {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Workflow found"})
	} else {
		http.Error(w, "Workflow not found", http.StatusNotFound)
	}
}
func (uh *WorkflowHandler) MiddlewareExtractUserFromHeader(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
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
func (h *WorkflowHandler) extractUserAndRoleFromToken(tokenString string) (userID string, role string, err error) {
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

func (h *WorkflowHandler) RoleRequired(next http.HandlerFunc, roles ...string) http.HandlerFunc {
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
