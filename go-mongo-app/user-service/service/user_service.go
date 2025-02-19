package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"user-service/db"
	"user-service/models"
	"user-service/notification"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	user   *db.Mongo
	logger *log.Logger
}

func NewUserService(user *db.Mongo, logger *log.Logger) *UserService {
	return &UserService{user, logger}
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

// sanitizeInput uklanja potencijalno opasne HTML tagove
func sanitizeInput(input string) string {
	sanitized := html.EscapeString(strings.TrimSpace(input))
	return sanitized
}

// validateUsername proverava da li username sadrži samo dozvoljene karaktere
func validateUsername(username string) (string, error) {
	// Dozvoljeni karakteri: slova, brojevi, donja crta i tačka, dužine 3-20 karaktera
	validUsernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_.]{3,20}$`)

	if !validUsernameRegex.MatchString(username) {
		return "", errors.New("invalid username format: only letters, numbers, underscores, and dots are allowed (3-20 characters)")
	}

	return username, nil
}

// sanitizeEmail proverava da li email sadrži samo validne karaktere
func sanitizeEmail(email string) string {
	// Regularni izraz za dozvoljene karaktere u email adresi
	validEmailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-@]+$`)

	if validEmailRegex.MatchString(email) {
		return email // Email je validan, nema potrebe za promenama
	}
	return "" // Ako email sadrži nedozvoljene karaktere, vraćamo praznu vrednost
}

var emailConfig = notification.EmailConfig{
	From:     os.Getenv("EMAIL_FROM"),
	Password: os.Getenv("EMAIL_PASSWORD"),
	SMTPHost: os.Getenv("SMTP_HOST"),
	SMTPPort: os.Getenv("SMTP_PORT"),
}

func GetActiveUsers() ([]models.User, error) {
	collection := db.Client.Database("testdb").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Case-insensitive match for 'role' with "member" or "Member"
	filter := bson.M{
		"isActive": true,
		"role": bson.M{
			"$in": []string{"member", "Member"},
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var activeUsers []models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		activeUsers = append(activeUsers, user)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return activeUsers, nil
}

func UserExists(userID string) (bool, error) {
	userCollection := db.Client.Database("testdb").Collection("users")
	var user models.User

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, errors.New("invalid user ID format")
	}

	err = userCollection.FindOne(context.TODO(), bson.M{"_id": userObjectID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return false, nil
	} else if err != nil {
		return false, err // Other errors
	}

	return true, nil
}

func GetUsers() ([]models.User, error) {
	collection := db.Client.Database("testdb").Collection("users")
	var users []models.User

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}

func GetUserByID(userID string) (models.User, error) {
	collection := db.Client.Database("testdb").Collection("users")

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return models.User{}, errors.New("invalid user ID format")
	}

	var user models.User
	err = collection.FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return models.User{}, errors.New("user not found")
	}

	return user, nil
}

func validateUser(user models.User) error {
	if user.Username == "" || user.Email == "" || user.Password == "" || user.Name == "" || user.Surname == "" {
		return errors.New("all fields (username, email, password, name, surname) are required")
	}

	if !isValidEmail(user.Email) {
		return errors.New("invalid email format")
	}
	if !isPasswordValid(user.Password) {
		return errors.New("invalid password format: password must be at least 8 characters long and contain an uppercase letter, a lowercase letter, a number, and a special character")
	}

	existingUserByUsername, err := FindUserByUsername(user.Username)
	if err == nil && existingUserByUsername.Username != "" {
		return errors.New("username already exists")
	}

	existingUserByEmail, err := FindUserByEmail(user.Email)
	if err == nil && existingUserByEmail.Email != "" {
		return errors.New("email already exists")
	}

	return nil
}
func isPasswordValid(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#~$%^&*(),.?":{}|<>]`).MatchString(password)

	return hasUpper && hasLower && hasNumber && hasSpecial
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}
func EmailExists(email string) (bool, error) {
	collection := db.Client.Database("testdb").Collection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
func UsernameExists(username string) (bool, error) {
	collection := db.Client.Database("testdb").Collection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"username": username}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
func RegisterUser(user models.User) (string, error) {
	// Provera da li je lozinka na blacklisti
	isBlacklistedPassword, err := isBlacklisted(user.Password)
	if err != nil {
		return "", fmt.Errorf("error checking password blacklist: %v", err)
	}
	if isBlacklistedPassword {
		return "", errors.New("Password is used too often")
	}

	// Sanitizacija korisničkih podataka
	user.Username = sanitizeInput(user.Username)
	user.Email = sanitizeInput(user.Email)
	user.Name = sanitizeInput(user.Name)
	user.Surname = sanitizeInput(user.Surname)

	// Validacija korisničkog imena
	sanitizedUsername, err := validateUsername(user.Username)
	if err != nil {
		return "", err
	}
	user.Username = sanitizedUsername

	// Sanitizacija emaila
	sanitizedEmail := sanitizeEmail(user.Email)
	if sanitizedEmail == "" {
		return "", errors.New("invalid email format")
	}
	user.Email = sanitizedEmail

	if err := validateUser(user); err != nil {
		return "", err
	}

	collection := db.Client.Database("testdb").Collection("users")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Hashiranje lozinke
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	user.Password = string(hashedPassword)
	user.IsActive = false

	// Unos korisnika u bazu
	_, err = collection.InsertOne(ctx, user)
	if err != nil {
		return "", err
	}

	// Generisanje tokena za potvrdu naloga
	token := generateToken()

	// Postavljanje vremena isteka tokena (3 minuta od trenutnog vremena)
	expiresAt := time.Now().UTC().Add(3 * time.Minute)

	// Čuvanje tokena u kolekciji za potvrdu
	resetCollection := db.Client.Database("testdb").Collection("confirmations")
	resetData := bson.M{
		"email":     user.Email,
		"token":     token,
		"expiresAt": expiresAt,
	}

	_, err = resetCollection.InsertOne(ctx, resetData)
	if err != nil {
		return "", fmt.Errorf("failed to store confirmation token: %v", err)
	}

	// Slanje emaila sa linkom za potvrdu
	subject := "Thanks for registering"
	body := fmt.Sprintf("Your registration is successful! Click the following link to activate your account: https://localhost/taskio/confirm?email=%s&token=%s", user.Email, token)
	err = notification.SendEmail(user.Email, subject, body, emailConfig)
	if err != nil {
		return "Registration successful, but failed to send confirmation email", nil
	}

	return "Registration successful. Please check your email to confirm registration.", nil
}

func FindUserByUsername(username string) (models.User, error) {
	// Validacija korisničkog imena
	validatedUsername, err := validateUsername(username) // Koristi novu validaciju
	if err != nil {
		return models.User{}, err // Vraća grešku ako username nije validan
	}

	collection := db.Client.Database("testdb").Collection("users")
	var user models.User

	err = collection.FindOne(context.TODO(), bson.M{"username": validatedUsername}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return models.User{}, nil // Korisnik nije pronađen
	}
	if err != nil {
		return models.User{}, err // Druga greška pri pretrazi baze
	}

	return user, nil
}

func FindUserByEmail(email string) (models.User, error) {
	collection := db.Client.Database("testdb").Collection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return models.User{}, nil
	}
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}
func ConfirmUser(email, token string) error {
	// Pronalaženje potvrde u bazi
	resetCollection := db.Client.Database("testdb").Collection("confirmations")
	filter := bson.M{"email": email, "token": token}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var resetData bson.M
	err := resetCollection.FindOne(ctx, filter).Decode(&resetData)
	if err != nil {
		return errors.New("invalid or expired confirmation link")
	}

	// Provera da li je token istekao
	expiresAtRaw, ok := resetData["expiresAt"]
	if !ok {
		return errors.New("expiresAt field not found in confirmation record")
	}

	expiresAt, ok := expiresAtRaw.(primitive.DateTime)
	if !ok {
		return errors.New("expiresAt field is not a valid date")
	}

	// Provera da li je token istekao u odnosu na trenutno vreme
	if time.Now().UTC().After(expiresAt.Time()) {
		return errors.New("confirmation link has expired")
	}

	// Ažuriranje korisnika kao potvrđenog
	collection := db.Client.Database("testdb").Collection("users")
	update := bson.M{"$set": bson.M{"isActive": true}}
	updateFilter := bson.M{"email": email}

	_, err = collection.UpdateOne(ctx, updateFilter, update)
	if err != nil {
		return err
	}

	// Brisanje tokena iz kolekcije nakon potvrde
	_, err = resetCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	return nil
}
func IsUserActive(email string) (bool, error) {
	collection := db.Client.Database("testdb").Collection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil // User does not exist
		}
		return false, err
	}
	return user.IsActive, nil
}
func ResetPassword(email string, method string) (string, error) {
	// Kolekcija korisnika (users)
	usersCollection := db.Client.Database("testdb").Collection("users")

	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Tražimo korisnika na osnovu email-a u kolekciji "users"
	err := usersCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return "", fmt.Errorf("The user with this email does not exist")
	}

	// Ako korisnik nije aktivan, ne dozvoljavamo resetovanje lozinke
	if !user.IsActive {
		return "", fmt.Errorf("The user is not active, you cannot reset the password")
	}

	// Proveriti da li već postoji važeći token za korisnika
	resetCollection := db.Client.Database("testdb").Collection("password_resets")
	var existingResetData bson.M
	err = resetCollection.FindOne(ctx, bson.M{"email": email}).Decode(&existingResetData)

	// Ako postoji aktivan token koji nije istekao, ne šaljemo novi
	if err == nil {
		expiresAt, ok := existingResetData["expiresAt"].(time.Time)
		if ok && expiresAt.After(time.Now().UTC()) {
			log.Println("Token already exists and is valid. Not sending email again.")
			return "", fmt.Errorf("A password reset token has already been sent. Please check your email.")
		} else if err != nil {
			// Ako je token istekao, obrišite ga pre slanja novog
			_, err = resetCollection.DeleteOne(ctx, bson.M{"email": email})
			if err != nil {
				return "", fmt.Errorf("Failed to delete expired reset token: %v", err)
			}
		}
	}

	if method == http.MethodPost {
		token := generateToken() // Generiši token za resetovanje lozinke
		expiresAt := time.Now().UTC().Add(3 * time.Minute)

		resetData := bson.M{
			"email":     email,
			"token":     token,
			"expiresAt": expiresAt,
		}

		// Čuvanje tokena u kolekciji "password_resets"
		_, err = resetCollection.InsertOne(ctx, resetData)
		if err != nil {
			return "", fmt.Errorf("Failed to store password reset token")
		}

		err = SendPasswordResetEmail(email, token)
		if err != nil {
			return "", fmt.Errorf("Error sending password reset email: %v", err)
		}
	}
	return "The password reset email has been successfully sent. Check your email.", nil
}

// Funkcija za generisanje nasumičnog tokena (ovo možeš prilagoditi prema potrebama)
func generateToken() string {
	b := make([]byte, 16) // 16 bajtova za duži i sigurniji token
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err) // Upravljanje greškom
	}
	return fmt.Sprintf("%x", b) // Vraća token kao heksadecimalni string
}

func SendPasswordResetEmail(email string, token string) error {
	subject := "Password reset"
	body := "Click the following link to reset your password: http://localhost/taskio/reset-password?email=" + email + "&token=" + token

	// Logovanje pre slanja emaila
	log.Println("Sending password reset email to:", email)

	err := notification.SendEmail(email, subject, body, emailConfig)
	if err != nil {
		log.Println("Error sending email:", err)
	}
	return err
}

func LoginUser(user models.User) (models.User, error) {
	collection := db.Client.Database("testdb").Collection("users")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var dbUser models.User
	err := collection.FindOne(ctx, bson.M{"username": user.Username}).Decode(&dbUser)
	if err == mongo.ErrNoDocuments {
		return models.User{}, fmt.Errorf("Invalid username or password")
	}
	if err != nil {
		return models.User{}, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password))
	if err != nil {
		return models.User{}, fmt.Errorf("Invalid username or password")
	}

	return dbUser, nil
}

func ChangePassword(userID, oldPassword, newPassword string) error {
	// Dohvati korisnika iz baze prema ID-u
	user, err := GetUserByID(userID)
	if err != nil {
		return errors.New("user not found") // Korisnik nije pronađen
	}

	// Poredi unetu staru lozinku sa hashovanom lozinkom u bazi
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword))
	if err != nil {
		return errors.New("incorrect old password") // Stara lozinka nije tačna
	}

	// Provjera da li je lozinka na blacklisti
	isBlacklistedPassword, err := isBlacklisted(newPassword)
	if err != nil {
		return fmt.Errorf("error checking password blacklist: %v", err)
	}
	if isBlacklistedPassword {
		return errors.New("Password is used too often")
	}

	// Hashiraj novu lozinku
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("could not hash the new password") // Problem sa generisanjem hash-a
	}

	// Ažuriraj lozinku u bazi podataka
	collection := db.Client.Database("testdb").Collection("users")
	update := bson.M{"$set": bson.M{"password": hashedPassword}}
	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": user.ID}, update)
	if err != nil {
		return errors.New("failed to update password") // Greška prilikom ažuriranja lozinke
	}

	return nil // Uspešno promenjena lozinka
}
func SendMagicLinkEmail(email, magicLink string) error {
	subject := "Magic Link for Login"
	body := "Click the following link to login: " + magicLink
	err := notification.SendEmail(email, subject, body, emailConfig)
	return err
}
func DeactivateUser(userID string) error {
	collection := db.Client.Database("testdb").Collection("users")

	// Pretvori string userID u ObjectID
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID format")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ažuriraj IsActive polje na false
	update := bson.M{"$set": bson.M{"isActive": false}}
	_, err = collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		return err
	}

	return nil
}
