package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"user-service/db"
	"user-service/models"
	"user-service/security"
	"user-service/service"

	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"

	"github.com/gorilla/mux"
)

func GetActiveUsers(w http.ResponseWriter, r *http.Request) {
	activeUsers, err := service.GetActiveUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activeUsers)
}
func CheckUserExists(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	if userID == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	exists, err := service.UserExists(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"exists": exists})
}

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

	// Decode the request body for JSON requests
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
	} else {
		// For web requests, get email from URL
		requestBody.Email = r.URL.Query().Get("email")
	}

	if requestBody.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// Call your service to send a reset email link
	response, err := service.ResetPassword(requestBody.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return a response without exposing any sensitive data
	if r.Header.Get("Content-Type") == "application/json" {
		jsonResponse := map[string]string{
			"message": response,
		}
		json.NewEncoder(w).Encode(jsonResponse)
	} else {
		// Display HTML form for password reset
		htmlForm := `
		<!DOCTYPE html>
<html>
<head>
	<title>Reset Password</title>
	<style>
		/* Opšti stilovi */
		body {
			font-family: Arial, sans-serif;
			background-color: #ffffff;
			display: flex;
			justify-content: center;
			align-items: center;
			height: 100vh;
			margin: 0;
		}

		/* Stil za glavni kontejner */
		.container {
			text-align: center;
			width: 100%;
			max-width: 400px;
			padding: 20px;
			background-color: #f9f9f9;
			border: 1px solid #e0e0e0;
			border-radius: 8px;
			box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
		}

		h1 {
			color: #2e7d32; /* Tamno zelena */
			font-size: 24px;
			margin-bottom: 20px;
		}

		form {
			margin-top: 10px;
		}

		label {
			display: block;
			font-size: 14px;
			margin-bottom: 6px;
			color: #4caf50;
		}

		input[type="password"] {
			width: 100%;
			padding: 10px;
			margin-bottom: 20px;
			border: 1px solid #cccccc;
			border-radius: 4px;
			box-sizing: border-box;
			font-size: 16px;
		}

		button {
			width: 100%;
			padding: 12px;
			background-color: #4caf50; /* Svetlija zelena */
			color: #ffffff;
			border: none;
			border-radius: 4px;
			font-size: 16px;
			cursor: pointer;
			transition: background-color 0.3s;
		}

		button:hover {
			background-color: #388e3c; /* Tamnija zelena */
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>Reset Your Password</h1>
		<form method="POST" action="/taskio/verify-password">
			<input type="hidden" name="email" value="` + requestBody.Email + `">
			<label for="newPassword">New Password:</label>
			<input type="password" id="newPassword" name="newPassword" required>
			<label for="confirmPassword">Confirm Password:</label>
			<input type="password" id="confirmPassword" name="confirmPassword" required>
			<button type="submit">Submit</button>
		</form>
	</div>
</body>
</html>

		`

		// Set the Content-Type header to HTML and write the HTML form
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlForm))
	}
}

func isBlacklisted(input string) (bool, error) {
	// Proveri trenutni radni direktorijum
	dir, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("Greška prilikom dobijanja radnog direktorijuma: %v", err)
	}
	fmt.Println("Trenutni radni direktorijum:", dir)

	file, err := os.Open("/root/service/blacklist.txt")
	if err != nil {
		return false, fmt.Errorf("Greška prilikom otvaranja fajla: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file) // Kreira scanner za čitanje fajla liniju po liniju
	for scanner.Scan() {
		// Uklanja whitespace sa linija za pouzdaniju provjeru
		line := strings.TrimSpace(scanner.Text())
		if line == input {
			return true, nil // Nađen je unos u blacklisti
		}
	}

	// Provjerava greške prilikom čitanja fajla
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("Greška prilikom čitanja fajla: %v", err)
	}

	return false, nil // Nema poklapanja
}

func HandleVerifyPassword(w http.ResponseWriter, r *http.Request) {
	// Parsiranje forme
	r.ParseForm()
	email := r.FormValue("email")
	newPassword := r.FormValue("newPassword")
	confirmPassword := r.FormValue("confirmPassword")

	// Proveri da li su lozinke iste
	if newPassword != confirmPassword {
		http.Error(w, "Lozinke se ne podudaraju", http.StatusBadRequest)
		return
	}

	isBlacklistedPassword, err := isBlacklisted(newPassword)
	if err != nil {
		http.Error(w, fmt.Sprintf("Greška prilikom provere lozinke: %v", err), http.StatusInternalServerError)
		return
	}
	if isBlacklistedPassword {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>Error</title>
				<style>
					body {
						font-family: Arial, sans-serif;
						background-color: #ffffff;
						display: flex;
						justify-content: center;
						align-items: center;
						height: 100vh;
						margin: 0;
					}
					.modal {
						display: block;
						position: fixed;
						z-index: 1;
						padding-top: 100px;
						left: 0;
						top: 0;
						width: 100%;
						height: 100%;
						background-color: rgba(0,0,0,0.5);
					}
					.modal-content {
						margin: auto;
						padding: 20px;
						width: 80%;
						max-width: 400px;
						background-color: #f9f9f9;
						border-radius: 8px;
						text-align: center;
						box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
					}
					.error-title {
						color: #d32f2f;
						font-size: 24px;
						margin-bottom: 20px;
					}
					.error-message {
						color: #333;
						font-size: 16px;
						margin-bottom: 20px;
					}
					button {
						background-color: #d32f2f;
						color: white;
						padding: 10px 20px;
						border: none;
						border-radius: 5px;
						cursor: pointer;
						font-size: 16px;
					}
					button:hover {
						background-color: #b71c1c;
					}
				</style>
			</head>
			<body>
				<div class="modal">
					<div class="modal-content">
						<h1 class="error-title">Error</h1>
						<p class="error-message">Password is blacklisted. Please choose a different password.</p>
						<button onclick="closeModal()">OK</button>
					</div>
				</div>
				<script>
					function closeModal() {
						document.querySelector('.modal').style.display = 'none';
					}
				</script>
			</body>
			</html>
		`))
		return
	}

	// Hash lozinke
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Greška prilikom heširanja lozinke", http.StatusInternalServerError)
		return
	}

	// Ažuriraj lozinku u bazi
	collection := db.Client.Database("testdb").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = collection.UpdateOne(ctx, bson.M{"email": email}, bson.M{"$set": bson.M{"password": string(hashedPassword)}})
	if err != nil {
		http.Error(w, "Greška prilikom ažuriranja lozinke", http.StatusInternalServerError)
		return
	}

	// Uspešan odgovor
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Reset Password Success</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					background-color: #ffffff;
					display: flex;
					justify-content: center;
					align-items: center;
					height: 100vh;
					margin: 0;
				}
				.container {
					text-align: center;
					width: 100%;
					max-width: 400px;
					padding: 20px;
					background-color: #f9f9f9;
					border: 1px solid #e0e0e0;
					border-radius: 8px;
					box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
				}
				h1 {
					color: #2e7d32;
					font-size: 24px;
					margin-bottom: 20px;
				}
				p {
					color: #333;
					font-size: 16px;
					margin-bottom: 20px;
				}
				a {
					color: #4caf50;
					text-decoration: none;
					font-weight: bold;
				}
				a:hover {
					text-decoration: underline;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<h1>Password Reset Successful</h1>
				<p>Your password has been successfully reset. You can now <a href="http://localhost:4200/login">log in</a> with your new password.</p>
			</div>
		</body>
		</html>
	`))
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
		"user_id":      authUser.ID,
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
func SendMagicLinkHandler(w http.ResponseWriter, r *http.Request) {
	// Pokušaj da uzmeš email iz query parametra
	email := r.URL.Query().Get("email")
	username := r.URL.Query().Get("username")

	// Ako nema email-a ili username-a u query parametru, proveri JSON telo
	if email == "" || username == "" {
		var user struct {
			Email    string `json:"email"`
			Username string `json:"username"`
		}

		// Parsiranje tela zahteva ako email ili username nisu prosleđeni u query
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "Nevalidan input", http.StatusBadRequest)
			return
		}
		email = user.Email
		username = user.Username
	}

	// Ako email ili username nisu prosleđeni ni u query ni u JSON telu, vrati grešku
	if email == "" || username == "" {
		http.Error(w, "Email and Username must be forwarded", http.StatusBadRequest)
		return
	}

	// Pronalaženje korisnika po email-u
	userData, err := service.FindUserByEmail(email)
	if err != nil {
		http.Error(w, "Korisnik nije pronađen", http.StatusNotFound)
		log.Printf("Error finding user with email %s: %v", email, err)
		return
	}

	// Provera da li je korisnik aktivan
	if !userData.IsActive {
		http.Error(w, "Korisnik nije aktivan", http.StatusForbidden)
		return
	}

	// Provera da li se email i username podudaraju
	if userData.Username != username {
		http.Error(w, "Username and email do not match", http.StatusBadRequest)
		return
	}

	// Generisanje magic link-a
	magicLink, err := security.GenerateMagicLink(userData)
	if err != nil {
		http.Error(w, "Error generating magic link", http.StatusInternalServerError)
		log.Printf("Error generating magic link for user %v: %v", userData, err)
		return
	}

	// Slanje magic link-a putem email-a
	err = service.SendMagicLinkEmail(email, magicLink)
	if err != nil {
		http.Error(w, "Error sending email", http.StatusInternalServerError)
		log.Printf("Error sending magic link to email %s: %v", email, err)
		return
	}

	// Uspešan odgovor
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "The magic link has been sent to the email",
	})
}

func VerifyMagicLinkHandler(w http.ResponseWriter, r *http.Request) {
	// Dohvati token iz URL-a
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		http.Error(w, "Token not found", http.StatusBadRequest)
		return
	}

	// Dekodiraj i verifikuj token
	claims, err := security.ParseAccessToken(tokenString) // Pozivamo funkciju za dekodiranje
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		log.Println("Error decoding token:", err)
		return
	}

	// Ako je token validan, koristi klaime za dalje akcije
	log.Printf("Korisnik ID: %s, Rola: %s\n", claims.ID, claims.Role)

	// Generisanje novog access token-a koji korisnik može koristiti nakon verifikacije
	accessToken, err := security.NewAccessToken(*claims) // Dereferencirajte claims
	if err != nil {
		http.Error(w, "Error generating new token", http.StatusInternalServerError)
		return
	}

	// Vraćanje odgovora sa novim access token-om i korisničkim informacijama
	response := map[string]interface{}{
		"access_token": accessToken, // Novi access token
		"role":         claims.Role,
		"user_id":      claims.ID, // Korisnički ID
	}

	// Pošaljite odgovor sa podacima u JSON formatu
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

}
func DeactivateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	if userID == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	err := service.DeactivateUser(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User deactivated successfully"})
}
