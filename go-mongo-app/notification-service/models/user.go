package models

import (
	"github.com/gocql/gocql"
	"time"
)

// User predstavlja model za korisnika
type User struct {
	ID        gocql.UUID `json:"id"`         // UUID identifikator korisnika
	FirstName string     `json:"first_name"` // Ime korisnika
	LastName  string     `json:"last_name"`  // Prezime korisnika
	Email     string     `json:"email"`      // Email adresa
	CreatedAt time.Time  `json:"created_at"` // Datum kreiranja korisnika
}
