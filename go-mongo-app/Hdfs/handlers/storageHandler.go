package handlers

import (
	"Hdfs/storage"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type KeyProduct struct{}
type KeyAccount struct{}

type KeyRole struct{}
type StorageHandler struct {
	logger *log.Logger
	// NoSQL: injecting file hdfs
	store *storage.FileStorage
	// Environment variables
	defaultFilePath    string
	defaultFileContent string
}

// Injecting the logger makes this code much more testable.
func NewStorageHandler(l *log.Logger, s *storage.FileStorage) *StorageHandler {
	// Učitavamo vrednosti iz okruženja (ako ne postoje, koristićemo default vrednosti)
	defaultFilePath := os.Getenv("DEFAULT_FILE_PATH")
	if defaultFilePath == "" {
		defaultFilePath = "/tmp" // default
	}

	defaultFileContent := os.Getenv("DEFAULT_FILE_CONTENT")
	if defaultFileContent == "" {
		defaultFileContent = "Hola Mundo!" // default
	}

	return &StorageHandler{
		logger:             l,
		store:              s,
		defaultFilePath:    defaultFilePath,
		defaultFileContent: defaultFileContent,
	}
}

func (s *StorageHandler) CopyFileToStorage(rw http.ResponseWriter, h *http.Request) {
	fileName := h.FormValue("fileName")

	err := s.store.CopyLocalFile(fileName, fileName)

	if err != nil {
		http.Error(rw, "File hdfs exception", http.StatusInternalServerError)
		s.logger.Println("File hdfs exception: ", err)
		return
	}
}

func (s *StorageHandler) WriteFileToStorage(rw http.ResponseWriter, h *http.Request) {
	// Ekstraktujte fileName iz zahteva
	fileName := h.FormValue("fileName")
	if fileName == "" {
		http.Error(rw, "fileName is required", http.StatusBadRequest)
		return
	}

	// Koristi podrazumevani sadržaj iz environment varijable
	fileContent := s.defaultFileContent

	// Zapišite fajl u skladište
	err := s.store.WriteFile(fileContent, fileName)
	if err != nil {
		http.Error(rw, "File hdfs exception BACK", http.StatusInternalServerError)
		s.logger.Println("File hdfs exception: ", err)
		return
	}

	// Vratite uspešan odgovor
	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(map[string]string{"message": "File written successfully"})
}

func (s *StorageHandler) ReadFileFromStorage(rw http.ResponseWriter, h *http.Request) {
	// Ekstraktujte fileName iz zahteva
	fileName := h.FormValue("fileName")
	if fileName == "" {
		http.Error(rw, "fileName is required", http.StatusBadRequest)
		return
	}

	// Proverite da li je fajl kopiran (opciono)
	copied := h.FormValue("isCopied")
	isCopied := false
	if copied != "" {
		isCopied = true
	}

	// Pročitajte fajl iz skladišta
	fileContent, err := s.store.ReadFile(fileName, isCopied)
	if err != nil {
		http.Error(rw, "File hdfs exception", http.StatusInternalServerError)
		s.logger.Println("File hdfs exception: ", err)
		return
	}

	// Vratite sadržaj fajla kao odgovor
	rw.Header().Set("Content-Type", "text/plain")
	io.WriteString(rw, fileContent)
	s.logger.Printf("Content of file %s: %s\n", fileName, fileContent)
}

func (s *StorageHandler) WalkRoot(rw http.ResponseWriter, h *http.Request) {
	pathsArray := s.store.WalkDirectories()
	paths := strings.Join(pathsArray, "\n")
	io.WriteString(rw, paths)
}

func (s *StorageHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		s.logger.Println("Method [", h.Method, "] - Hit path :", h.URL.Path)

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}
func (uh *StorageHandler) MiddlewareExtractUserFromHeader(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(rw http.ResponseWriter, h *http.Request) {
		// Retrieve the token from the Authorization header
		authHeader := h.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(rw, "No Authorization header found", http.StatusUnauthorized)
			uh.logger.Println("No Authorization header:", authHeader)
			return
		}

		// Expect the format "Bearer <token>"
		tokenString := ""
		if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
			tokenString = authHeader[7:]
		} else {
			http.Error(rw, "Invalid Authorization header format", http.StatusUnauthorized)
			uh.logger.Println("Invalid Authorization header format:", authHeader)
			return
		}

		// Extract userID and role from the token directly
		userID, role, err := uh.extractUserAndRoleFromToken(tokenString)
		if err != nil {
			uh.logger.Println("Token extraction failed:", err)
			http.Error(rw, `{"message": "Invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Log the userID and role
		uh.logger.Println("User ID is:", userID, "Role is:", role)

		// Add userID and role to the request context
		ctx := context.WithValue(h.Context(), KeyAccount{}, userID)
		ctx = context.WithValue(ctx, KeyRole{}, role)

		// Update the request with the new context
		h = h.WithContext(ctx)

		// Pass the request along the middleware chain
		next(rw, h)
	}
}

// Helper method to extract userID and role from JWT token
func (h *StorageHandler) extractUserAndRoleFromToken(tokenString string) (userID string, role string, err error) {
	// Parse the token
	// Replace with your actual secret key
	secretKey := []byte(os.Getenv("TOKEN_SECRET"))

	// Parse and validate the token
	parsedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		// Validate the algorithm (ensure it's signed with HMAC)
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil || !parsedToken.Valid {
		return "", "", fmt.Errorf("invalid token: %v", err)
	}

	// Extract claims from the token
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", fmt.Errorf("invalid token claims")
	}

	// Extract userID and role from the claims
	userID, ok = claims["id"].(string)
	if !ok {
		return "", "", fmt.Errorf("userID not found in token")
	}

	role, ok = claims["role"].(string)
	if !ok {
		return "", "", fmt.Errorf("role not found in token")
	}

	return userID, role, nil
}

func (h *StorageHandler) RoleRequired(next http.HandlerFunc, roles ...string) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) { // changed 'r' to 'req'
		// Extract the role from the request context
		role, ok := req.Context().Value(KeyRole{}).(string) // 'req' instead of 'r'
		if !ok {
			http.Error(rw, "Role not found in context", http.StatusForbidden)
			return
		}

		// Check if the user's role is in the list of required roles
		for _, r := range roles {
			if role == r {
				// If the role matches, pass the request to the next handler in the chain
				next(rw, req) // 'req' instead of 'r'
				return
			}
		}

		// If the role doesn't match any of the required roles, return a forbidden error
		http.Error(rw, "Forbidden", http.StatusForbidden)
	}
}
