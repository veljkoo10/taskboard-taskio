package security

import (
	"fmt"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"os"
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
