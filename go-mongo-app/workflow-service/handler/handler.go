package handler

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
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

func (h *WorkflowHandler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var workflowRequest struct {
		TaskID         string   `json:"task_id"`
		DependencyTask []string `json:"dependency_task"`
	}

	// Dekodiraj telo zahteva u strukturu
	if err := json.NewDecoder(r.Body).Decode(&workflowRequest); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding request: %v", err), http.StatusBadRequest)
		return
	}

	// Provera da li je task_id prazan
	if workflowRequest.TaskID == "" {
		http.Error(w, "task_id cannot be empty", http.StatusBadRequest)
		return
	}

	// Provera da li postoje zavisnosti
	if len(workflowRequest.DependencyTask) == 0 {
		http.Error(w, "dependency_task cannot be empty", http.StatusBadRequest)
		return
	}

	// Pre nego što dodaš novi workflow, proveri da li bi nove zavisnosti mogle stvoriti ciklus
	err := h.repo.CheckForCycle(workflowRequest.TaskID, workflowRequest.DependencyTask)
	if err != nil {
		// Ako postoji ciklus, odbijamo kreiranje workflow-a
		http.Error(w, fmt.Sprintf("Cycle detected: %v", err), http.StatusBadRequest)
		return
	}

	// Kreiraj novi workflow
	workflow := models.Workflow{
		TaskID:         workflowRequest.TaskID,
		DependencyTask: workflowRequest.DependencyTask,
		IsActive:       true, // Aktivni workflow
	}

	// Pozivamo metodu koja zapravo kreira workflow u bazi
	err = h.repo.CreateWorkflow(r.Context(), workflow)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating workflow: %v", err), http.StatusInternalServerError)
		return
	}

	// Uspešan odgovor sa HTTP status kodom 201 (Created)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // HTTP status za kreirani resurs
	response := map[string]string{
		"message": "Workflow dependency successfully created",
		"task_id": workflow.TaskID,
	}

	// JSON odgovor koji potvrđuje uspešno kreiranje
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

	// Pozovi repo metod za dobijanje task-a
	task, err := repoWorkflow.GetTaskFromTaskService(taskID)
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
