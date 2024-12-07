package handler

import (
	"encoding/json"
	"net/http"
	"workflow-service/models"
	"workflow-service/repoWorkflow"
)

type WorkflowHandler struct {
	repo *repoWorkflow.WorkflowRepository
}

func NewWorkflowHandler(repo *repoWorkflow.WorkflowRepository) *WorkflowHandler {
	return &WorkflowHandler{repo: repo}
}

func (h *WorkflowHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var task models.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := h.repo.CreateWorkflow(r.Context(), task); err != nil {
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Task created successfully"))
}
func (h *WorkflowHandler) GetWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	// Dobavi task ID iz URL-a

	// Pozovi repo metodu za dobijanje task-a
	task, err := h.repo.GetAllWorkflows(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Vrati task kao JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}
