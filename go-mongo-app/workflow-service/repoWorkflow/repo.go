package repoWorkflow

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"io/ioutil"
	"log"
	"net/http"
	"workflow-service/models"
)

type WorkflowRepository struct {
	driver neo4j.Driver
}

func NewWorkflowRepository(driver neo4j.Driver) *WorkflowRepository {
	return &WorkflowRepository{driver: driver}
}

func (r *WorkflowRepository) CreateWorkflow(ctx context.Context, workflow models.Workflow) error {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Proveravamo da li workflow već postoji sa istim task_id i project_id
	result, err := session.Run(
		`MATCH (w:Workflow {task_id: $task_id, project_id: $project_id}) 
         RETURN w.dependency_task AS dependency_task`,
		map[string]interface{}{
			"task_id":    workflow.TaskID,
			"project_id": workflow.ProjectID, // Dodajemo ProjectID
		},
	)
	if err != nil {
		return fmt.Errorf("error checking for existing workflow: %v", err)
	}

	// Ako workflow postoji, dodajemo novu zavisnost
	if result.Next() {
		existingDeps, _ := result.Record().Get("dependency_task")
		var currentDeps []string

		if deps, ok := existingDeps.([]interface{}); ok {
			for _, dep := range deps {
				if depStr, ok := dep.(string); ok {
					currentDeps = append(currentDeps, depStr)
				}
			}
		}

		// Dodajemo nove zavisnosti ako nisu već prisutne
		for _, dep := range workflow.DependencyTask {
			if !contains(currentDeps, dep) {
				_, err = session.Run(
					`MATCH (w:Workflow {task_id: $task_id, project_id: $project_id})
                     SET w.dependency_task = w.dependency_task + $new_dep`,
					map[string]interface{}{
						"task_id":    workflow.TaskID,
						"project_id": workflow.ProjectID, // Koristimo ProjectID
						"new_dep":    dep,
					},
				)
				if err != nil {
					return fmt.Errorf("error adding dependency_task: %v", err)
				}
			}
		}
	} else {
		// Ako workflow ne postoji, kreiramo novi workflow
		_, err := session.Run(
			`CREATE (w:Workflow {id: $id, task_id: $task_id, dependency_task: $dependency_task, project_id: $project_id, is_active: $is_active})`,
			map[string]interface{}{
				"id":              uuid.New().String(),
				"task_id":         workflow.TaskID,
				"dependency_task": workflow.DependencyTask,
				"project_id":      workflow.ProjectID, // Dodajemo ProjectID
				"is_active":       workflow.IsActive,
			},
		)
		if err != nil {
			return fmt.Errorf("error creating workflow: %v", err)
		}
	}

	// Provera ciklusa ostaje nepromenjena
	err = r.checkAndHandleCycle(workflow)
	if err != nil {
		return fmt.Errorf("error handling cycle for workflow %s: %v", workflow.TaskID, err)
	}

	return nil
}

// Pomoćna funkcija za proveru postojanja elementa u listi
func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// Funkcija za proveru ciklusa i postavljanje is_active na false ako je potrebno
func (r *WorkflowRepository) checkAndHandleCycle(createdWorkflow models.Workflow) error {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Prolazimo kroz sve zavisnosti trenutnog workflow-a
	for _, depTaskID := range createdWorkflow.DependencyTask {
		// Proveravamo ciklus u zavisnostima
		result, err := session.Run(
			`MATCH (start:Workflow {task_id: $task_id}),
					  (end:Workflow {task_id: $dep_task_id})
			 MATCH path = (start)-[:DEPENDS_ON*]->(end)
			 WHERE start <> end
			 RETURN COUNT(path) AS path_count`,
			map[string]interface{}{
				"task_id":     createdWorkflow.TaskID,
				"dep_task_id": depTaskID,
			},
		)
		if err != nil {
			return fmt.Errorf("error checking for cycle: %v", err)
		}

		// Ako postoji ciklus, postavljamo is_active na false za oba workflow-a
		if result.Next() {
			pathCount, _ := result.Record().Get("path_count")
			if pathCount.(int64) > 0 {
				log.Printf("Cyclic dependency detected between task %s and %s", createdWorkflow.TaskID, depTaskID)

				// Ako postoji ciklus, postavljamo is_active na false za oba workflow-a
				_, err := session.Run(
					`MATCH (w:Workflow {task_id: $task_id})
					 SET w.is_active = false`,
					map[string]interface{}{
						"task_id": createdWorkflow.TaskID,
					},
				)
				if err != nil {
					return fmt.Errorf("error setting is_active=false for task %s: %v", createdWorkflow.TaskID, err)
				}

				return fmt.Errorf("Cyclic dependency detected, new workflow is deactivated")
			}
		}
	}

	return nil
}

func (r *WorkflowRepository) GetAllWorkflows(ctx context.Context) ([]*models.Workflow, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Izmenjeni Cypher upit da vraća i project_id
	result, err := session.Run(
		"MATCH (w:Workflow) RETURN w.id AS id, w.task_id AS task_id, w.dependency_task AS dependency_task, w.is_active AS is_active, w.project_id AS project_id",
		nil, // Nema parametara jer želimo sve workflow-e
	)
	if err != nil {
		return nil, fmt.Errorf("error running the query: %v", err)
	}

	var workflows []*models.Workflow

	for result.Next() {
		record := result.Record()

		idRaw, found := record.Get("id")
		if !found {
			return nil, fmt.Errorf("id not found in the result")
		}

		taskIDRaw, found := record.Get("task_id")
		if !found {
			return nil, fmt.Errorf("task_id not found in the result")
		}

		dependencyTaskRaw, found := record.Get("dependency_task")
		if !found {
			return nil, fmt.Errorf("dependency_task not found in the result")
		}

		isActiveRaw, found := record.Get("is_active")
		if !found {
			return nil, fmt.Errorf("is_active not found in the result")
		}

		projectIDRaw, found := record.Get("project_id")
		if !found {
			return nil, fmt.Errorf("project_id not found in the result")
		}

		id, ok := idRaw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type for id: expected string, got %T", idRaw)
		}

		taskID, ok := taskIDRaw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type for task_id: expected string, got %T", taskIDRaw)
		}

		// Prilagođeni kod za dependency_task
		var dependencyTask []string
		if depTasks, ok := dependencyTaskRaw.([]interface{}); ok {
			for _, dep := range depTasks {
				if depStr, ok := dep.(string); ok {
					dependencyTask = append(dependencyTask, depStr)
				} else {
					return nil, fmt.Errorf("invalid type for dependency_task element: expected string, got %T", dep)
				}
			}
		} else {
			return nil, fmt.Errorf("invalid type for dependency_task: expected []interface{}, got %T", dependencyTaskRaw)
		}

		isActive, ok := isActiveRaw.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid type for is_active: expected bool, got %T", isActiveRaw)
		}

		projectID, ok := projectIDRaw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type for project_id: expected string, got %T", projectIDRaw)
		}

		workflow := &models.Workflow{
			ID:             uuid.MustParse(id), // Konvertujemo string u UUID
			TaskID:         taskID,
			DependencyTask: dependencyTask, // Niz zavisnih taskova
			IsActive:       isActive,
			ProjectID:      projectID, // Dodajemo project_id
		}

		workflows = append(workflows, workflow)
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over the result: %v", err)
	}

	return workflows, nil
}

func (r *WorkflowRepository) ClearDatabase(ctx context.Context) error {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Cypher upit za brisanje svih čvorova i veza
	_, err := session.Run("MATCH (n) DETACH DELETE n", nil)
	if err != nil {
		return fmt.Errorf("error clearing database: %v", err)
	}
	return nil
}

func GetTaskFromTaskService(taskID string) (*models.Task, error) {
	// Definiši URL endpoint-a task-service koji sada koristi port 8080
	url := fmt.Sprintf("http://task-service:8080/tasks/%s", taskID)

	// Pošaljite GET zahtev prema task-service-u
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request to task-service: %v", err)
	}
	defer resp.Body.Close()

	// Ako status nije 200 OK, vrati grešku
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("task with id %s not found, received status 404 from task-service", taskID)
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("task with id %s not found, received status %v from task-service", taskID, resp.StatusCode)
	}

	// Učitaj telo odgovora
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Dekodiraj JSON odgovor u model Task
	var task models.Task
	if err := json.Unmarshal(body, &task); err != nil {
		return nil, fmt.Errorf("error unmarshaling task data: %v", err)
	}

	return &task, nil
}

func (r *WorkflowRepository) CheckTaskDependency(ctx context.Context, taskID string) (bool, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Prvo loguj pre nego što pokreneš upit
	log.Printf("Proveravam zavisnosti za task_id: %s", taskID)

	// Upit koji proverava da li postoji neki dependency task u workflow-u
	result, err := session.Run(
		"MATCH (w:Workflow) WHERE w.task_id = $task_id AND ANY(dep IN w.dependency_task WHERE dep <> $task_id) RETURN w.dependency_task AS dependency_tasks",
		map[string]interface{}{"task_id": taskID},
	)
	if err != nil {
		log.Printf("Greška prilikom pokretanja upita: %v", err)
		return false, fmt.Errorf("error running the query: %v", err)
	}

	// Ako nema zavisnosti
	if result.Next() {
		dependencyTasks, _ := result.Record().Get("dependency_tasks")
		log.Printf("Zavisni taskovi: %v", dependencyTasks) // Loguj koji su zavisni taskovi
		if dependencyTasks != nil {
			if tasks, ok := dependencyTasks.([]interface{}); ok && len(tasks) > 0 {
				log.Println("Postoje zavisnosti.")
				return true, nil
			}
		}
	}

	log.Println("Nema zavisnosti.")
	return false, nil
}

// Provera ciklusa u workflow zavisnostima
func (r *WorkflowRepository) CheckForCycle(taskID string, dependencyTasks []string) error {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Proveravamo svaki dependency task pre nego što ga dodamo
	for _, depTaskID := range dependencyTasks {
		// Cypher upit za proveru ciklusa u zavisnostima
		cycleCheckResult, err := session.Run(
			`MATCH (start:Workflow {task_id: $task_id}),
					  (end:Workflow {task_id: $dep_task_id})
			 MATCH path = (start)-[:DEPENDS_ON*]->(end)
			 WHERE start <> end
			 RETURN COUNT(path) AS path_count`,
			map[string]interface{}{
				"task_id":     taskID,
				"dep_task_id": depTaskID,
			},
		)
		if err != nil {
			log.Printf("Error checking cycle: %v", err)
			return fmt.Errorf("error checking for cycle: %v", err)
		}

		// Proveravamo da li postoji ciklus
		if cycleCheckResult.Next() {
			pathCount, _ := cycleCheckResult.Record().Get("path_count")
			if pathCount.(int64) > 0 {
				// Ako postoji ciklus, postavljamo is_active na false
				log.Printf("Cyclic dependency detected between task %s and %s", taskID, depTaskID)
				return fmt.Errorf("cyclic dependency detected between task %s and %s", taskID, depTaskID)
			}
		}
	}

	// Ako ne postoji ciklus, proces može da nastavi
	return nil
}
func (r *WorkflowRepository) GetTaskDependencies(ctx context.Context, taskID string) ([]string, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Logovanje pre pokretanja upita
	log.Printf("Dohvatam zavisne taskove za task_id: %s", taskID)

	// Upit za pronalazak zavisnih taskova
	result, err := session.Run(
		"MATCH (w:Workflow) WHERE w.task_id = $task_id RETURN w.dependency_task AS dependency_tasks",
		map[string]interface{}{"task_id": taskID},
	)
	if err != nil {
		log.Printf("Greška prilikom pokretanja upita: %v", err)
		return nil, fmt.Errorf("error running the query: %v", err)
	}

	var dependencies []string

	// Obrada rezultata
	if result.Next() {
		dependencyTasks, _ := result.Record().Get("dependency_tasks")
		if dependencyTasks != nil {
			if tasks, ok := dependencyTasks.([]interface{}); ok {
				for _, task := range tasks {
					if taskStr, ok := task.(string); ok {
						dependencies = append(dependencies, taskStr)
					}
				}
			}
		}
	}

	if err = result.Err(); err != nil {
		log.Printf("Greška prilikom obrade rezultata: %v", err)
		return nil, fmt.Errorf("error processing results: %v", err)
	}

	log.Printf("Zavisni taskovi za task_id %s: %v", taskID, dependencies)
	return dependencies, nil
}
func (r *WorkflowRepository) GetAllWorkflowsByProjectID(ctx context.Context, projectID string) ([]*models.Workflow, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Cypher upit za dohvaćanje svih workflow-e po project_id
	result, err := session.Run(
		`MATCH (w:Workflow {project_id: $project_id}) 
         RETURN w.id AS id, 
                w.task_id AS task_id, 
                w.dependency_task AS dependency_task, 
                w.project_id AS project_id, 
                w.is_active AS is_active`,
		map[string]interface{}{"project_id": projectID},
	)
	if err != nil {
		return nil, fmt.Errorf("error running the query: %v", err)
	}

	var workflows []*models.Workflow

	for result.Next() {
		record := result.Record()

		idRaw, _ := record.Get("id")
		taskIDRaw, _ := record.Get("task_id")
		dependencyTaskRaw, _ := record.Get("dependency_task")
		projectIDRaw, _ := record.Get("project_id")
		isActiveRaw, _ := record.Get("is_active")

		id, _ := idRaw.(string)
		taskID, _ := taskIDRaw.(string)
		projectID, _ := projectIDRaw.(string)
		isActive, _ := isActiveRaw.(bool)

		var dependencyTask []string
		if depTasks, ok := dependencyTaskRaw.([]interface{}); ok {
			for _, dep := range depTasks {
				if depStr, ok := dep.(string); ok {
					dependencyTask = append(dependencyTask, depStr)
				}
			}
		}

		workflow := &models.Workflow{
			ID:             uuid.MustParse(id),
			TaskID:         taskID,
			DependencyTask: dependencyTask,
			ProjectID:      projectID,
			IsActive:       isActive,
		}

		workflows = append(workflows, workflow)
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over the result: %v", err)
	}

	return workflows, nil
}
