package model

import "time"

// EventType defines the type of event
type EventType string

const (
	MemberAddedType       EventType = "Member Added to Project"
	MemberRemovedType     EventType = "Member Removed from Project"
	MemberAddedTaskType   EventType = "Member Added to Task"
	MemberRemovedTaskType EventType = "Member Removed from Task"
	TaskCreatedType       EventType = "Task Created"
	TaskStatusChangedType EventType = "Task Status Changed"
	DocumentAddedType     EventType = "Document Added"
	ProjectCreatedType    EventType = "Project Created"
)

// Event represents a generic event with a type and time
type Event struct {
	Type      EventType `json:"type"`
	Time      time.Time `json:"time"`
	Event     any       `json:"event"`
	ProjectID string    `json:"projectId"`
}

// MemberAddedToProjectEvent represents an event when a member is added to a project
type MemberAddedToProjectEvent struct {
	MemberID  string `json:"memberId"`
	ProjectID string `json:"projectId"`
}

// MemberRemovedFromProjectEvent represents an event when a member is removed from a project
type MemberRemovedFromProjectEvent struct {
	MemberID  string `json:"memberId"`
	ProjectID string `json:"projectId"`
}

// MemberAddedToTaskEvent represents an event when a member is added to a task
type MemberAddedToTaskEvent struct {
	MemberID string `json:"memberId"`
	TaskID   string `json:"taskId"`
}

// MemberRemovedFromTaskEvent represents an event when a member is removed from a task
type MemberRemovedFromTaskEvent struct {
	MemberID string `json:"memberId"`
	TaskID   string `json:"taskId"`
}

// TaskCreatedEvent represents an event when a new task is created in a project
type TaskCreatedEvent struct {
	TaskID    string `json:"taskId"`
	ProjectID string `json:"projectId"`
}

// TaskStatusChangedEvent represents an event when the status of a task changes
type TaskStatusChangedEvent struct {
	TaskID         string `json:"taskId"`
	ProjectID      string `json:"projectId"`
	PreviousStatus string `json:"previousStatus"`
	CurrentStatus  string `json:"currentStatus"`
	MemberID       string `json:"memberId"`
}

// DocumentAddedEvent represents an event when a document is added to a task
type DocumentAddedEvent struct {
	TaskID    string `json:"taskId"`
	ProjectID string `json:"projectId"`
	FilePath  string `json:"filePath"`
	MemberID  string `json:"memberId"`
}

// ProjectCreatedEvent represents an event when a new project is created
type ProjectCreatedEvent struct {
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name"`
	ManagerID string    `json:"managerId"`
	CreatedAt time.Time `json:"createdAt"`
}
