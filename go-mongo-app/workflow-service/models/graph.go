package models

import "github.com/google/uuid"

type Workflow struct {
	ID             uuid.UUID `json:"id"`
	TaskID         string    `json:"task_id"`
	ProjectID      string    `json:"project_id"`
	DependencyTask []string  `json:"dependency_task"`
	IsActive       bool      `json:"is_active"`
}
