package models

type Workflow struct {
	DependencyTasks []string `json:"dependency_tasks"`
	TaskID          string   `json:"task_id"`
}
