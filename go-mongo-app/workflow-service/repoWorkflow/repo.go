package repoWorkflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"workflow-service/models"
)

type WorkflowRepository struct {
	driver neo4j.Driver
}

func NewWorkflowRepository(driver neo4j.Driver) *WorkflowRepository {
	return &WorkflowRepository{driver: driver}
}

func (r *WorkflowRepository) CreateWorkflow(ctx context.Context, workflow models.Workflow) error {
	// Proveravamo da li osnovni task postoji
	exists, err := taskExists(workflow.TaskID)
	if err != nil {
		return fmt.Errorf("error checking if task exists: %v", err)
	}
	if !exists {
		return fmt.Errorf("task with task_id %s does not exist", workflow.TaskID)
	}

	// Proveravamo da li neki od dependency_task postoji
	for _, dep := range workflow.DependencyTask {
		exists, err := taskExists(dep)
		if err != nil {
			return fmt.Errorf("error checking if dependency task exists: %v", err)
		}
		if !exists {
			return fmt.Errorf("dependency task with task_id %s does not exist", dep)
		}
	}

	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Proveri postojeće zavisnosti
	existingDeps := []string{}
	result, err := session.Run(
		`MATCH (w:Workflow {task_id: $task_id, project_id: $project_id}) 
		 RETURN w.dependency_task AS dependency_task`,
		map[string]interface{}{
			"task_id":    workflow.TaskID,
			"project_id": workflow.ProjectID,
		},
	)
	if err != nil {
		return fmt.Errorf("error checking for existing workflow: %v", err)
	}

	if result.Next() {
		if deps, ok := result.Record().Get("dependency_task"); ok {
			if depList, valid := deps.([]interface{}); valid {
				for _, dep := range depList {
					if depStr, isString := dep.(string); isString {
						existingDeps = append(existingDeps, depStr)
					}
				}
			}
		}
	}

	// Kombinuj nove i postojeće zavisnosti
	newDeps := []string{}
	for _, dep := range workflow.DependencyTask {
		if !contains(existingDeps, dep) {
			newDeps = append(newDeps, dep)
		}
	}

	// Simuliraj zavisnosti i proveri ciklus
	simulatedDeps := append(existingDeps, newDeps...)
	if err := r.CheckForCycle(ctx, workflow.TaskID, simulatedDeps); err != nil {
		return fmt.Errorf("cycle detected: %v", err)
	}

	// Ako nema ciklusa, ažuriraj ili kreiraj čvor
	if len(existingDeps) > 0 {
		for _, dep := range newDeps {
			_, err = session.Run(
				`MATCH (w:Workflow {task_id: $task_id, project_id: $project_id})
				 SET w.dependency_task = w.dependency_task + $new_dep`,
				map[string]interface{}{
					"task_id":    workflow.TaskID,
					"project_id": workflow.ProjectID,
					"new_dep":    dep,
				},
			)
			if err != nil {
				return fmt.Errorf("error adding dependency_task: %v", err)
			}
		}
	} else {
		_, err := session.Run(
			`CREATE (w:Workflow {id: $id, task_id: $task_id, dependency_task: $dependency_task, project_id: $project_id, is_active: $is_active})`,
			map[string]interface{}{
				"id":              uuid.New().String(),
				"task_id":         workflow.TaskID,
				"dependency_task": workflow.DependencyTask,
				"project_id":      workflow.ProjectID,
				"is_active":       workflow.IsActive,
			},
		)
		if err != nil {
			return fmt.Errorf("error creating workflow: %v", err)
		}
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

func (r *WorkflowRepository) CheckForCycle(ctx context.Context, startTaskID string, dependencies []string) error {
	// Kreiraj graf iz baze
	graph := make(map[string][]string)

	// Popuni graf zavisnostima iz baze
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.Run(`MATCH (w:Workflow) RETURN w.task_id AS task_id, w.dependency_task AS dependency_task`, nil)
	if err != nil {
		return fmt.Errorf("error fetching workflows: %v", err)
	}

	for result.Next() {
		// Umesto GetByIndex, koristi Get
		taskID, ok := result.Record().Get("task_id")
		if !ok {
			return fmt.Errorf("task_id not found in record")
		}

		deps, ok := result.Record().Get("dependency_task")
		if !ok {
			return fmt.Errorf("dependency_task not found in record")
		}

		// Pretpostavimo da je deps lista (ako nije, moraćeš da prilagodiš ovo)
		if depList, valid := deps.([]interface{}); valid {
			for _, dep := range depList {
				if depStr, isString := dep.(string); isString {
					graph[taskID.(string)] = append(graph[taskID.(string)], depStr)
				}
			}
		}
	}

	// Dodaj novu zavisnost u graf za privremenu proveru
	graph[startTaskID] = append(graph[startTaskID], dependencies...)

	// Pokreni DFS
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var dfs func(task string) bool
	dfs = func(task string) bool {
		if stack[task] {
			return true // Ciklus pronađen
		}
		if visited[task] {
			return false
		}

		visited[task] = true
		stack[task] = true

		for _, neighbor := range graph[task] {
			if dfs(neighbor) {
				return true
			}
		}

		stack[task] = false
		return false
	}

	if dfs(startTaskID) {
		return fmt.Errorf("cyclic dependency detected")
	}

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
func taskExists(taskID string) (bool, error) {
	// URL do task_service
	url := "http://task-service:8080/tasks/exists" // Pretpostavka da koristiš HTTP POST za provjeru

	// Priprema tela zahteva
	requestBody := map[string]string{"task_id": taskID}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("failed to create request body: %v", err)
	}

	// Slanje HTTP POST zahteva task_service-u
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("failed to send request to task_service: %v", err)
	}
	defer resp.Body.Close()

	// Čitanje odgovora
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %v", err)
	}

	// Provera statusnog koda
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("received non-OK response: %s", body)
	}

	// Parsiranje odgovora
	var result map[string]bool
	err = json.Unmarshal(body, &result)
	if err != nil {
		return false, fmt.Errorf("failed to parse response body: %v", err)
	}

	// Ekstrakcija polja 'exists'
	exists, ok := result["exists"]
	if !ok {
		return false, fmt.Errorf("response missing 'exists' field")
	}

	return exists, nil
}

// DeleteWorkflowsByTaskID briše sve workflow-e koji imaju određeni TaskID.
func (r *WorkflowRepository) DeleteWorkflowsByTaskID(taskID string) error {
	session := r.driver.NewSession(neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite, // Ispravan AccessMode
	})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := `
        MATCH (w:Workflow {task_id: $taskID})
        DETACH DELETE w
    `
		params := map[string]interface{}{
			"taskID": taskID,
		}
		_, err := tx.Run(query, params)
		return nil, err
	})

	return err
}

func (r WorkflowRepository) FindWorkflowByTaskID(taskID string) (bool, error) {
	// Otvori sesiju za Neo4j
	session := r.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	// Pokreni transakciju za pretragu
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println(ctx)

	query := `
		MATCH (w:Workflow {task_id: $taskID})
		RETURN w LIMIT 1
	`

	// Izvrši transakciju
	found := false
	_, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(query, map[string]interface{}{
			"taskID": taskID,
		})
		if err != nil {
			return nil, fmt.Errorf("query execution failed: %v", err)
		}

		// Proveri da li postoje rezultati
		if result.Next() {
			found = true
		}
		return nil, nil
	})

	if err != nil {
		return false, fmt.Errorf("failed to search for workflow with task_id %s: %v", taskID, err)
	}

	return found, nil
}
