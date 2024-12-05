package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
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

type KeyAccount struct{}

type KeyRole struct{}
type UserHandler struct {
	logger  *log.Logger
	service *service.UserService
}

func NewUserHandler(logger *log.Logger, service *service.UserService) *UserHandler {
	return &UserHandler{logger, service}
}

const (
	Manager = "Manager"
	Member  = "Member"
)

func (h *UserHandler) GetActiveUsers(w http.ResponseWriter, r *http.Request) {
	activeUsers, err := service.GetActiveUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activeUsers)
}
func (h *UserHandler) CheckUserExists(w http.ResponseWriter, r *http.Request) {
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

func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := service.GetUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
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
func (h *UserHandler) ConfirmUser(w http.ResponseWriter, r *http.Request) {
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
func (h *UserHandler) CheckUsername(w http.ResponseWriter, r *http.Request) {
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
func (h *UserHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
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

func (h *UserHandler) isBlacklisted(input string) (bool, error) {
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

func (h *UserHandler) HandleVerifyPassword(w http.ResponseWriter, r *http.Request) {
	// Parsiranje forme
	r.ParseForm()
	email := r.FormValue("email")
	newPassword := r.FormValue("newPassword")
	confirmPassword := r.FormValue("confirmPassword")
	r1 := true
	r2 := true
	r3 := true
	r4 := true

	// Proverite da li lozinka sadrži barem jedno veliko slovo
	matched, _ := regexp.MatchString("[A-Z]", newPassword)
	if !matched {
		r1 = false
	}

	// Proverite da li lozinka sadrži barem jedno malo slovo
	matched, _ = regexp.MatchString("[a-z]", newPassword)
	if !matched {
		r2 = false
	}

	// Proverite da li lozinka sadrži barem jedan broj
	matched, _ = regexp.MatchString("[0-9]", newPassword)
	if !matched {
		r3 = false
	}

	// Proverite da li lozinka sadrži barem jedan specijalni karakter
	matched, _ = regexp.MatchString(`[!@#$%^&*(),.?":{}|<>]`, newPassword)
	if !matched {
		r4 = false
	}
	// Ako lozinke nisu iste
	if newPassword != confirmPassword {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`
            <!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>Greška</title>
                <style>
                    body {
                        font-family: Arial, sans-serif;
                        margin: 0;
                        height: 100vh;
                        display: flex;
                        justify-content: center;
                        align-items: center;
                        background-color: rgba(0, 0, 0, 0.5);
                    }
                    .modal {
                        display: block;
                        position: relative;
                        z-index: 1;
                        width: 80%;
                        max-width: 400px;
                        padding: 20px;
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
                    <h1 class="error-title">Greška</h1>
                    <p class="error-message">Lozinke se ne podudaraju.</p>
                    <button onclick="closeModal()">OK</button>
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

	// Proveri da li je lozinka na crnoj listi
	isBlacklistedPassword, err := h.isBlacklisted(newPassword)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`
            <!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>Greška</title>
                <style>
                    body {
                        font-family: Arial, sans-serif;
                        margin: 0;
                        height: 100vh;
                        display: flex;
                        justify-content: center;
                        align-items: center;
                        background-color: rgba(0, 0, 0, 0.5);
                    }
                    .modal {
                        display: block;
                        position: relative;
                        z-index: 1;
                        width: 80%;
                        max-width: 400px;
                        padding: 20px;
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
                    <h1 class="error-title">Greška</h1>
                    <p class="error-message">Greška prilikom provere lozinke: ` + err.Error() + `</p>
                    <button onclick="goBack()">OK</button>
                </div>
                <script>
					function goBack() {
                		window.history.back();  // Vraća korisnika na prethodnu stranicu
            		}
                </script>
            </body>
            </html>
        `))
		return
	} else if isBlacklistedPassword {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`
            <!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>Greška</title>
                <style>
                    body {
                        font-family: Arial, sans-serif;
                        margin: 0;
                        height: 100vh;
                        display: flex;
                        justify-content: center;
                        align-items: center;
                        background-color: rgba(0, 0, 0, 0.5);
                    }
                    .modal {
                        display: block;
                        position: relative;
                        z-index: 1;
                        width: 80%;
                        max-width: 400px;
                        padding: 20px;
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
                    <h1 class="error-title">Greška</h1>
                    <p class="error-message">Lozinka je na crnoj listi. Molimo vas da izaberete drugu lozinku.</p>
                    <button onclick="goBack()">OK</button>
                </div>
                <script>
                    function closeModal() {
                        document.querySelector('.modal').style.display = 'none';
                    }

					function goBack() {
                		window.history.back();  // Vraća korisnika na prethodnu stranicu
            		}
                </script>
            </body>
            </html>
        `))
		return
	} else if !r1 || !r2 || !r3 || !r4 {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`
            <!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>Greška</title>
                <style>
                    body {
                        font-family: Arial, sans-serif;
                        margin: 0;
                        height: 100vh;
                        display: flex;
                        justify-content: center;
                        align-items: center;
                        background-color: rgba(0, 0, 0, 0.5);
                    }
                    .modal {
                        display: block;
                        position: relative;
                        z-index: 1;
                        width: 80%;
                        max-width: 400px;
                        padding: 20px;
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
                    <h1 class="error-title">Error</h1>
                    <p class="error-message">Password must contain at least one uppercase letter, lowercase letter, number and character.</p>
                    <button onclick="goBack()">OK</button>
                </div>
                <script>
                    function closeModal() {
                        document.querySelector('.modal').style.display = 'none';
                    }

					function goBack() {
                		window.history.back();  // Vraća korisnika na prethodnu stranicu
            		}
                </script>
            </body>
            </html>
        `))
		return
	}

	// Hash lozinke i nastavi sa procedurom
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`
            <!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>Greška</title>
                <style>
                    body {
                        font-family: Arial, sans-serif;
                        margin: 0;
                        height: 100vh;
                        display: flex;
                        justify-content: center;
                        align-items: center;
                        background-color: rgba(0, 0, 0, 0.5);
                    }
                    .modal {
                        display: block;
                        position: relative;
                        z-index: 1;
                        width: 80%;
                        max-width: 400px;
                        padding: 20px;
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
                    <h1 class="error-title">Greška</h1>
                    <p class="error-message">Greška prilikom heširanja lozinke: ` + err.Error() + `</p>
                    <button onclick="goBack()">OK</button>
                </div>
                <script>
                    function goBack() {
                		window.history.back();  // Vraća korisnika na prethodnu stranicu
            		}
                </script>
            </body>
            </html>
        `))
		return
	}

	// Ažuriraj lozinku u bazi
	collection := db.Client.Database("testdb").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = collection.UpdateOne(ctx, bson.M{"email": email}, bson.M{"$set": bson.M{"password": string(hashedPassword)}})
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`
            <!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>Greška</title>
                <style>
                    body {
                        font-family: Arial, sans-serif;
                        margin: 0;
                        height: 100vh;
                        display: flex;
                        justify-content: center;
                        align-items: center;
                        background-color: rgba(0, 0, 0, 0.5);
                    }
                    .modal {
                        display: block;
                        position: relative;
                        z-index: 1;
                        width: 80%;
                        max-width: 400px;
                        padding: 20px;
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
                    <h1 class="error-title">Greška</h1>
                    <p class="error-message">Greška prilikom ažuriranja lozinke: ` + err.Error() + `</p>
                    <button onclick="goBack()">OK</button>
                </div>
                <script>
                    function closeModal() {
                        document.querySelector('.modal').style.display = 'none';
                    }
            		function goBack() {
                		window.history.back();  // Vraća korisnika na prethodnu stranicu
            		}
                </script>
            </body>
            </html>
        `))
		return
	}

	// Uspešan odgovor
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
        <!DOCTYPE html>
        <html>
        <head>
            <title>Password Reset Success</title>
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

func (h *UserHandler) CheckUserActive(w http.ResponseWriter, r *http.Request) {
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
func (h *UserHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
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

func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
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
func (h *UserHandler) SendMagicLinkHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *UserHandler) VerifyMagicLinkHandler(w http.ResponseWriter, r *http.Request) {
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
func (h *UserHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
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

func (uh *UserHandler) MiddlewareExtractUserFromHeader(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
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
func (uh *UserHandler) extractUserAndRoleFromToken(tokenString string) (userID string, role string, err error) {
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

func (uh *UserHandler) RoleRequired(next http.HandlerFunc, roles ...string) http.HandlerFunc {
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
