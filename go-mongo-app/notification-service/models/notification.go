package models

import (
	"github.com/gocql/gocql"
	"time"
)

// Notification predstavlja model za notifikaciju
type Notification struct {
	ID        gocql.UUID `json:"id"`         // UUID identifier for notification
	UserID    gocql.UUID `json:"user_id"`    // UUID of the user who will receive the notification
	Message   string     `json:"message"`    // The notification message
	IsActive  bool       `json:"is_active"`  // The status of the notification
	CreatedAt time.Time  `json:"created_at"` // Timestamp when the notification was created
}
