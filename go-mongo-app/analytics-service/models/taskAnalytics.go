package models

import "time"

// TaskAnalytics struct for tracking time spent in each task state
type TaskAnalytics struct {
	TaskID           string           `bson:"task_id" json:"task_id"`
	StatusTimes      map[string]int64 `bson:"status_times" json:"status_times"` // Seconds spent in each state
	LastStatusChange time.Time        `bson:"last_status_change" json:"last_status_change"`
}
