package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"time"
	"user-service/db"
	"user-service/models"
	"user-service/security"
	"user-service/service"

	"github.com/gorilla/mux"
)

func GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := service.GetUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func GetUserByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	if userID == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	user, err := service.GetUserByID(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	message, err := service.RegisterUser(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}
func ConfirmUser(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Missing email", http.StatusBadRequest)
		return
	}

	err := service.ConfirmUser(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html")

	htmlResponse := `
        <!DOCTYPE html>
        <html lang="en">
        <head>
            <meta charset="UTF-8">
            <meta name="viewport" content="width=device-width, initial-scale=1.0">
            <title>Account Confirmation</title>
            <style>
                body {
                    display: flex;
                    justify-content: center;
                    align-items: center;
                    height: 100vh;
                    margin: 0;
                    font-family: Arial, sans-serif;
                    background-color: #f4f4f9;
                }
                .message {
                    font-size: 2em;
                    color: #4CAF50;
                    text-align: center;
                }
            </style>
        </head>
        <body>
            <div class="message">
                Account confirmed successfully!
            </div>
        </body>
        </html>
    `

	w.Write([]byte(htmlResponse))
}
func CheckEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Missing email", http.StatusBadRequest)
		return
	}

	exists, err := service.EmailExists(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"exists": exists})
}
func CheckUsername(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Missing username", http.StatusBadRequest)
		return
	}

	exists, err := service.UsernameExists(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"exists": exists})
}
func HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		Email string `json:"email"`
	}

	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
	} else {
		requestBody.Email = r.URL.Query().Get("email")
	}

	if requestBody.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	response, err := service.ResetPassword(requestBody.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.Client.Database("testdb").Collection("users")
	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = collection.FindOne(ctx, bson.M{"email": requestBody.Email}).Decode(&user)
	if err != nil {
		http.Error(w, "No user found with this email", http.StatusBadRequest)
		return
	}

	if r.Header.Get("Content-Type") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		jsonResponse := map[string]string{
			"message":  response,
			"password": user.Password,
		}
		json.NewEncoder(w).Encode(jsonResponse)
	} else {
		w.Header().Set("Content-Type", "text/html")
		htmlForm := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Username Verification</title>
		</head>
		<body>
			<h1>Enter your username</h1>
			<form method="POST" action="/verify-username">
				<input type="hidden" name="email" value="` + user.Email + `">
				<label for="username">Username:</label>
				<input type="text" id="username" name="username" required>
				<button type="submit">Send</button>
			</form>
		</body>
		</html>
	`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlForm))

	}
}
func HandleVerifyUsername(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	email := r.FormValue("email")
	username := r.FormValue("username")

	collection := db.Client.Database("testdb").Collection("users")
	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := collection.FindOne(ctx, bson.M{"email": email, "username": username}).Decode(&user)
	if err != nil {
		http.Error(w, "It's not your username", http.StatusBadRequest)
		return
	}

	htmlResponse := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Password Reset</title>
	</head>
	<body>
		<h1>Password Reset Successful</h1>
		<p>Your current password is: %s</p>
	</body>
	</html>
`
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, htmlResponse, user.Password)
}

func CheckUserActive(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Missing email", http.StatusBadRequest)
		return
	}

	isActive, err := service.IsUserActive(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"active": isActive})

}
func LoginUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authUser, err := service.LoginUser(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	isActive, err := service.IsUserActive(authUser.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !isActive {
		http.Error(w, "User account is inactive", http.StatusForbidden)
		return
	}

	claims := security.UserClaims{
		ID:       authUser.ID,
		Role:     authUser.Role,
		IsActive: authUser.IsActive,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
		},
	}

	accessToken, err := security.NewAccessToken(claims)
	if err != nil {
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": accessToken,
		"role":         authUser.Role,
	})
}

func ChangePassword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	if userID == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	var requestBody struct {
		OldPassword     string `json:"oldPassword"`
		NewPassword     string `json:"newPassword"`
		ConfirmPassword string `json:"confirmPassword"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if requestBody.NewPassword != requestBody.ConfirmPassword {
		http.Error(w, "New passwords do not match", http.StatusBadRequest)
		return
	}

	err := service.ChangePassword(userID, requestBody.OldPassword, requestBody.NewPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Password changed successfully"})
}
