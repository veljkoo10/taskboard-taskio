package repoWorkflow

import (
	"context"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"workflow-service/models"
)

type WorkflowRepository struct {
	driver neo4j.Driver
}

func NewWorkflowRepository(driver neo4j.Driver) *WorkflowRepository {
	return &WorkflowRepository{driver: driver}
}

func (r *WorkflowRepository) CreateWorkflow(ctx context.Context, task models.Task) error {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	_, err := session.Run(
		"CREATE (t:Task {id: $id, name: $name, status: $status})",
		map[string]interface{}{
			"id":     task.ID,
			"name":   task.Name,
			"status": task.Status,
		},
	)
	return err
}

func (r *WorkflowRepository) GetAllWorkflows(ctx context.Context) ([]*models.Task, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Neo4j upit za vraćanje svih taskova
	result, err := session.Run(
		"MATCH (t:Task) RETURN t.id AS id, t.name AS name, t.status AS status",
		nil, // Nema parametara jer želimo sve taskove
	)
	if err != nil {
		return nil, fmt.Errorf("error running the query: %v", err)
	}

	var tasks []*models.Task

	// Iteriramo kroz rezultate i kreiramo taskove
	for result.Next() {
		// Vraćanje rezultata u model koristeći Get()
		record := result.Record()

		// Provera da li id, name, status postoje u rekordu
		idRaw, found := record.Get("id")
		if !found {
			return nil, fmt.Errorf("id not found in the result")
		}
		nameRaw, found := record.Get("name")
		if !found {
			return nil, fmt.Errorf("name not found in the result")
		}
		statusRaw, found := record.Get("status")
		if !found {
			return nil, fmt.Errorf("status not found in the result")
		}

		// Proveri da li su vrednosti odgovarajućeg tipa
		id, ok := idRaw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type for id: expected string, got %T", idRaw)
		}

		name, ok := nameRaw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type for name: expected string, got %T", nameRaw)
		}

		status, ok := statusRaw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type for status: expected string, got %T", statusRaw)
		}

		// Kreiranje Task modela sa proverenim vrednostima
		task := &models.Task{
			ID:     id,
			Name:   name,
			Status: status,
		}

		// Dodajemo task u listu
		tasks = append(tasks, task)
	}

	// Proveravamo da li je bilo grešaka prilikom iteracije
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over the result: %v", err)
	}

	return tasks, nil
}
