package models

import (
	"github.com/gocql/gocql"
	"time"
)

type User struct {
	ID        gocql.UUID `json:"id"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	Email     string     `json:"email"`
	CreatedAt time.Time  `json:"created_at"`
}
