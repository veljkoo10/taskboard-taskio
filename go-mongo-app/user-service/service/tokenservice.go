package service

import (
	"fmt"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"os"
)

// User predstavlja korisnika u aplikaciji
type User struct {
	ID       primitive.ObjectID `json:"id"`
	Username string             `json:"username"`
	Role     string             `json:"role"`
	IsActive bool               `json:"isActive"`
}

// UserClaims sadrži podatke koji će biti upisani u JWT token
type UserClaims struct {
	ID       primitive.ObjectID `json:"id"`       // ID korisnika
	Role     string             `json:"role"`     // Rola korisnika
	IsActive bool               `json:"isActive"` // Da li je korisnik aktivan
	jwt.StandardClaims
}

// NewAccessToken kreira JWT token sa podacima o korisniku
func NewAccessToken(claims UserClaims) (string, error) { // Prima UserClaims kao argument
	// Kreira token sa UserClaims
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Potpisuje token koristeći tajni ključ
	return accessToken.SignedString([]byte(os.Getenv("TOKEN_SECRET")))
}

// ParseAccessToken parsira JWT token i vraća UserClaims
func ParseAccessToken(accessToken string) (*UserClaims, error) {
	// Parsira token i vraća UserClaims iz njega
	parsedAccessToken, err := jwt.ParseWithClaims(accessToken, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("TOKEN_SECRET")), nil
	})

	if err != nil {
		return nil, err
	}

	// Vraća parsed claims kao UserClaims
	claims, ok := parsedAccessToken.Claims.(*UserClaims)
	if !ok {
		return nil, fmt.Errorf("nevalidni token")
	}

	return claims, nil
}
