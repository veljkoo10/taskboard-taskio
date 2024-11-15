package security

import (
	"fmt"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"os"
	"time"
	"user-service/models"
)

type User struct {
	ID       primitive.ObjectID `json:"id"`
	Username string             `json:"username"`
	Role     string             `json:"role"`
	IsActive bool               `json:"isActive"`
}

type UserClaims struct {
	ID       primitive.ObjectID `json:"id"`
	Role     string             `json:"role"`
	IsActive bool               `json:"isActive"`
	jwt.StandardClaims
}

func NewAccessToken(claims UserClaims) (string, error) {
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return accessToken.SignedString([]byte(os.Getenv("TOKEN_SECRET")))
}
func GenerateMagicLink(user models.User) (string, error) {
	// Koristiš korisničke podatke, uključujući rolu
	claims := UserClaims{
		ID:       user.ID,
		Role:     user.Role, // Uzimaš rolu korisnika iz baze
		IsActive: user.IsActive,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: jwt.TimeFunc().Add(time.Hour).Unix(), // Token traje 1 sat
		},
	}

	token, err := NewAccessToken(claims)
	if err != nil {
		return "", err
	}

	// Generišemo magic link sa tokenom
	magicLink := fmt.Sprintf("http://localhost:4200/verify-magic-link?token=%s", token)
	return magicLink, nil
}

func ParseAccessToken(accessToken string) (*UserClaims, error) {
	parsedAccessToken, err := jwt.ParseWithClaims(accessToken, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("TOKEN_SECRET")), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := parsedAccessToken.Claims.(*UserClaims)
	if !ok {
		return nil, fmt.Errorf("nevalidni token")
	}

	return claims, nil
}
